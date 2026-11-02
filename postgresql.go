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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/lib/pq"
)

const (
	defaultPostgresVersion = "17-alpine"
	dbName                 = "postgres"
	templateDBName         = "echoprobe_template"
	dbUsername             = "postgres"
	dbPassword             = "password"
	dbPort                 = "5432/tcp"
)

// PostgresDBContainer holds all the necessary information for postgres database test container.
type PostgresDBContainer struct {
	testcontainers.Container

	DBHost     string
	DBPort     int
	DBName     string
	DBUsername string
	DBPassword string
}

// PostgresOptions configures the Postgres container that RunWithPostgres shares between all tests of a package.
// The InitSQLScript is applied once to the template database that every test database is copied from.
type PostgresOptions struct {
	Version       string
	InitSQLScript string
}

// sharedPostgres is set by RunWithPostgres. When nil, every test starts its own container.
var sharedPostgres *postgresPool

// RunWithPostgres runs the tests of a package against one shared Postgres container and returns the exit code.
// Call it from TestMain. Each test that uses IntegrationTestWithPostgres gets its own database, copied from a
// template database. The container starts when the first test needs it and stops after all tests have run.
func RunWithPostgres(m *testing.M, opts PostgresOptions) int {
	sharedPostgres = &postgresPool{opts: opts}
	defer func() {
		sharedPostgres.close()
		sharedPostgres = nil
	}()

	return m.Run()
}

// postgresPool hands out a fresh database per test from one shared container.
type postgresPool struct {
	opts      PostgresOptions
	once      sync.Once
	container *PostgresDBContainer
	admin     *sql.DB
	err       error
	counter   atomic.Int64
}

func (p *postgresPool) start() {
	ctx := context.Background()

	p.container, p.err = setupPostgresDB(ctx, p.opts.Version, templateDBName, p.opts.InitSQLScript)
	if p.err != nil {
		p.err = fmt.Errorf("starting shared postgres container: %w", p.err)
		return
	}

	p.admin, p.err = sql.Open("postgres", databaseURL(p.container.DBHost, strconv.Itoa(p.container.DBPort), dbName))
}

// createDatabase copies the template database into a new database with a unique name.
func (p *postgresPool) createDatabase(ctx context.Context) (*PostgresDBContainer, error) {
	p.once.Do(p.start)
	if p.err != nil {
		return nil, p.err
	}

	name := fmt.Sprintf("test_%d", p.counter.Add(1))
	if _, err := p.admin.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", name, templateDBName)); err != nil {
		return nil, fmt.Errorf("creating database %s: %w", name, err)
	}

	container := *p.container
	container.DBName = name

	return &container, nil
}

func (p *postgresPool) dropDatabase(ctx context.Context, name string) error {
	_, err := p.admin.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", name))
	return err
}

func (p *postgresPool) close() {
	if p.admin != nil {
		_ = p.admin.Close()
	}

	if p.container != nil {
		if err := p.container.Terminate(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "error detected during container termination: %v\n", err)
		}
	}
}

// conflict reports a per-test option that differs from the package options. Empty per-test values inherit them.
func (p PostgresOptions) conflict(o IntegrationTestWithPostgres) error {
	if o.Version != "" && postgresVersion(o.Version) != postgresVersion(p.Version) {
		return fmt.Errorf(
			"IntegrationTestWithPostgres.Version %q differs from the RunWithPostgres version %q: "+
				"leave it empty, use the same value, or move the test to a package without RunWithPostgres",
			o.Version, postgresVersion(p.Version),
		)
	}

	if o.InitSQLScript != "" && o.InitSQLScript != p.InitSQLScript {
		return fmt.Errorf(
			"IntegrationTestWithPostgres.InitSQLScript %q differs from the RunWithPostgres init script %q: "+
				"leave it empty, use the same value, or move the test to a package without RunWithPostgres",
			o.InitSQLScript, p.InitSQLScript,
		)
	}

	return nil
}

func postgresVersion(version string) string {
	if version == "" {
		return defaultPostgresVersion
	}

	return version
}

// setupPostgresDB sets up a postgres database test container.
// Init SQL scripts are mounted into /docker-entrypoint-initdb.d/ so Postgres
// executes them in the database named by database during its own initialization,
// before accepting TCP connections.
func setupPostgresDB(ctx context.Context, version, database, initSQLScript string) (*PostgresDBContainer, error) {
	req := testcontainers.ContainerRequest{
		Image: "postgres:" + postgresVersion(version),
		Env: map[string]string{
			"POSTGRES_USER":     dbUsername,
			"POSTGRES_PASSWORD": dbPassword,
			"POSTGRES_DB":       database,
		},
		ExposedPorts: []string{dbPort},
		WaitingFor:   wait.ForSQL(dbPort, "postgres", dbURL),
	}

	if strings.TrimSpace(initSQLScript) != "" {
		executionPath, err := testpath()
		if err != nil {
			return nil, fmt.Errorf("resolving fixture path: %w", err)
		}

		req.Files = []testcontainers.ContainerFile{
			{
				HostFilePath:      filepath.Join(executionPath, "fixtures", initSQLScript),
				ContainerFilePath: "/docker-entrypoint-initdb.d/" + filepath.Base(initSQLScript),
				FileMode:          0644,
			},
		}
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	hostIP, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	mappedPort, err := container.MappedPort(ctx, dbPort)
	if err != nil {
		return nil, err
	}

	return &PostgresDBContainer{
		Container:  container,
		DBHost:     hostIP,
		DBPort:     int(mappedPort.Num()),
		DBName:     database,
		DBUsername: dbUsername,
		DBPassword: dbPassword,
	}, nil
}

// dbURL returns the URL of the default postgres database.
func dbURL(host string, port string) string {
	return databaseURL(host, port, dbName)
}

// databaseURL returns the URL of the given postgres database.
func databaseURL(host, port, database string) string {
	port, _, _ = strings.Cut(port, "/")
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUsername, dbPassword, host, port, database,
	)
}
