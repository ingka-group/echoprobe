// Copyright © 2024 Ingka Holding B.V. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// You may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package echoprobe

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"iter"
	"maps"
	"os"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lib/pq"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
)

// sharedPostgres holds the Postgres servers that the tests of this process share, one per version.
var sharedPostgres postgresPool

// RunWithPostgres runs the tests of a package and stops the shared Postgres containers afterwards. Call it from
// TestMain. It turns on container sharing even when Ryuk is disabled, because it stops the containers itself.
func RunWithPostgres(m *testing.M) int {
	sharedPostgres.managed.Store(true)
	defer func() {
		var errs []error
		for server := range sharedPostgres.servers.values() {
			errs = append(errs, server.close())
		}
		if err := errors.Join(errs...); err != nil {
			fmt.Fprintf(os.Stderr, "echoprobe: stopping shared postgres containers: %v\n", err)
		}
	}()

	return m.Run()
}

// postgresPool starts a shared Postgres server per version on first use.
type postgresPool struct {
	managed atomic.Bool
	servers onceMap[string, *postgresServer]
}

// createDatabase creates a database for t on the shared server for version, and drops it when t finishes.
func (p *postgresPool) createDatabase(t testing.TB, version, initSQLScript string) (*PostgresDBContainer, error) {
	version = postgresVersion(version)

	server, err := p.servers.get(version, func() (*postgresServer, error) {
		return startPostgresServer(context.WithoutCancel(t.Context()), version)
	})
	if err != nil {
		return nil, err
	}

	container, err := server.createDatabase(t.Context(), initSQLScript)
	if err != nil {
		return nil, err
	}

	t.Cleanup(func() {
		// t.Context is already canceled when cleanups run.
		query := "DROP DATABASE IF EXISTS " + pq.QuoteIdentifier(container.DBName) + " WITH (FORCE)"
		if _, err := server.admin.ExecContext(context.Background(), query); err != nil {
			t.Logf("error detected while dropping database %q: %v", container.DBName, err)
		}
	})

	return container, nil
}

// postgresServer is a shared container. Each test gets its own database, copied from a template database per
// init script.
type postgresServer struct {
	container *PostgresDBContainer
	admin     *sql.DB
	databases atomic.Int64
	templates onceMap[string, string]
}

func startPostgresServer(ctx context.Context, version string) (*postgresServer, error) {
	container, err := setupPostgresDB(ctx, version, "")
	if err != nil {
		return nil, fmt.Errorf("starting shared postgres container: %w", err)
	}
	container.shared = true

	admin, err := sql.Open("postgres", databaseURL(container.DBHost, strconv.Itoa(container.DBPort), dbName))
	if err != nil {
		return nil, errors.Join(err, container.Terminate(ctx))
	}

	return &postgresServer{container: container, admin: admin}, nil
}

// createDatabase creates a database for one test. With an init script, the database is a copy of the template
// database for that script.
func (s *postgresServer) createDatabase(ctx context.Context, initSQLScript string) (*PostgresDBContainer, error) {
	name := s.newDatabaseName("test")
	query := "CREATE DATABASE " + pq.QuoteIdentifier(name)

	if initSQLScript != "" {
		template, err := s.templates.get(initSQLScript, func() (string, error) {
			return s.createTemplate(context.WithoutCancel(ctx), initSQLScript)
		})
		if err != nil {
			return nil, err
		}
		query += " TEMPLATE " + pq.QuoteIdentifier(template)
	}

	if _, err := s.admin.ExecContext(ctx, query); err != nil {
		return nil, fmt.Errorf("creating database %q: %w", name, err)
	}

	container := *s.container
	container.DBName = name

	return &container, nil
}

// createTemplate creates a template database and runs the init script in it with psql inside the container, the
// same way the Postgres image runs scripts from /docker-entrypoint-initdb.d/.
func (s *postgresServer) createTemplate(ctx context.Context, initSQLScript string) (string, error) {
	name := s.newDatabaseName("template")
	if _, err := s.admin.ExecContext(ctx, "CREATE DATABASE "+pq.QuoteIdentifier(name)); err != nil {
		return "", fmt.Errorf("creating template database %q: %w", name, err)
	}

	scriptPath := "/tmp/" + name + ".sql"
	if err := s.container.CopyFileToContainer(ctx, initSQLScript, scriptPath, 0o644); err != nil {
		return "", fmt.Errorf("copying init script %q: %w", initSQLScript, err)
	}

	cmd := []string{"psql", "-v", "ON_ERROR_STOP=1", "-U", dbUsername, "-d", name, "-f", scriptPath}
	code, output, err := s.container.Exec(ctx, cmd, tcexec.Multiplexed())
	if err != nil {
		return "", fmt.Errorf("running init script %q: %w", initSQLScript, err)
	}
	if code != 0 {
		out, _ := io.ReadAll(output)
		return "", fmt.Errorf("running init script %q: exit code %d: %s", initSQLScript, code, out)
	}

	return name, nil
}

func (s *postgresServer) newDatabaseName(prefix string) string {
	return prefix + "_" + strconv.FormatInt(s.databases.Add(1), 10)
}

func (s *postgresServer) close() error {
	return errors.Join(s.admin.Close(), s.container.Terminate(context.Background()))
}

// onceMap computes the value for each key once. Concurrent callers for the same key wait for the first one and
// share its result, including its error.
//
// Parallel tests ask for the same server or template at the same moment; without this, each would start its own
// container or build its own template. The mutex only guards the map lookup, so keys are computed concurrently:
// starting one Postgres version does not block a test that needs another.
type onceMap[K comparable, V any] struct {
	mu    sync.Mutex
	funcs map[K]func() (V, error)
}

func (m *onceMap[K, V]) get(key K, compute func() (V, error)) (V, error) {
	m.mu.Lock()
	fn, ok := m.funcs[key]
	if !ok {
		if m.funcs == nil {
			m.funcs = map[K]func() (V, error){}
		}
		fn = sync.OnceValues(compute)
		m.funcs[key] = fn
	}
	m.mu.Unlock()

	return fn()
}

// values yields the values that were computed without an error.
func (m *onceMap[K, V]) values() iter.Seq[V] {
	return func(yield func(V) bool) {
		m.mu.Lock()
		funcs := slices.Collect(maps.Values(m.funcs))
		m.mu.Unlock()

		for _, fn := range funcs {
			if v, err := fn(); err == nil && !yield(v) {
				return
			}
		}
	}
}
