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
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/lib/pq"
)

const (
	defaultPostgresVersion = "17-alpine"
	dbName                 = "postgres"
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

	shared bool
}

// sharedPostgres holds one Postgres container per version, shared by all tests in the test process.
var sharedPostgres = &postgresServers{}

// RunWithPostgres runs the tests of a package and stops the shared Postgres containers afterwards. Call it from
// TestMain. It turns on container sharing even when Ryuk is disabled, because it stops the containers itself.
func RunWithPostgres(m *testing.M) int {
	sharedPostgres.managed.Store(true)
	defer sharedPostgres.close()

	return m.Run()
}

type postgresServers struct {
	managed atomic.Bool

	mu      sync.Mutex
	servers map[string]*postgresServer
}

// enabled reports whether tests share containers. Without RunWithPostgres, nothing stops a shared container, so
// sharing then relies on Ryuk to remove the containers after the test process exits.
func (s *postgresServers) enabled() bool {
	return s.managed.Load() || !testcontainers.ReadConfig().Config.RyukDisabled
}

func (s *postgresServers) server(version string) *postgresServer {
	version = postgresVersion(version)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.servers == nil {
		s.servers = map[string]*postgresServer{}
	}

	server, ok := s.servers[version]
	if !ok {
		server = &postgresServer{version: version}
		s.servers[version] = server
	}

	return server
}

func (s *postgresServers) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, server := range s.servers {
		server.close()
	}
	s.servers = nil
	s.managed.Store(false)
}

// postgresServer is one shared container. Each test gets its own database, copied from a template database
// per init script.
type postgresServer struct {
	version   string
	once      sync.Once
	container *PostgresDBContainer
	admin     *sql.DB
	err       error
	databases atomic.Int64

	mu        sync.Mutex
	templates map[string]*postgresTemplate
}

type postgresTemplate struct {
	once sync.Once
	name string
	err  error
}

func (s *postgresServer) start() {
	ctx := context.Background()

	s.container, s.err = setupPostgresDB(ctx, s.version, "")
	if s.err != nil {
		s.err = fmt.Errorf("starting shared postgres container: %w", s.err)
		return
	}
	s.container.shared = true

	s.admin, s.err = sql.Open("postgres", databaseURL(s.container.DBHost, strconv.Itoa(s.container.DBPort), dbName))
}

// createDatabase creates a database for one test. With an init script, the database is a copy of the template
// database for that script.
func (s *postgresServer) createDatabase(ctx context.Context, initSQLScript string) (*PostgresDBContainer, error) {
	s.once.Do(s.start)
	if s.err != nil {
		return nil, s.err
	}

	name := fmt.Sprintf("test_%d", s.databases.Add(1))
	query := "CREATE DATABASE " + name
	if initSQLScript != "" {
		template, err := s.template(ctx, initSQLScript)
		if err != nil {
			return nil, err
		}
		query += " TEMPLATE " + template
	}

	if _, err := s.admin.ExecContext(ctx, query); err != nil {
		return nil, fmt.Errorf("creating database %s: %w", name, err)
	}

	container := *s.container
	container.DBName = name

	return &container, nil
}

func (s *postgresServer) template(ctx context.Context, initSQLScript string) (string, error) {
	s.mu.Lock()
	if s.templates == nil {
		s.templates = map[string]*postgresTemplate{}
	}
	template, ok := s.templates[initSQLScript]
	if !ok {
		template = &postgresTemplate{name: fmt.Sprintf("echoprobe_template_%d", len(s.templates)+1)}
		s.templates[initSQLScript] = template
	}
	s.mu.Unlock()

	template.once.Do(func() {
		template.err = s.createTemplate(ctx, template.name, initSQLScript)
	})

	return template.name, template.err
}

// createTemplate runs the init script with psql inside the container, the same way the Postgres image runs
// scripts from /docker-entrypoint-initdb.d/.
func (s *postgresServer) createTemplate(ctx context.Context, name, initSQLScript string) error {
	if _, err := s.admin.ExecContext(ctx, "CREATE DATABASE "+name); err != nil {
		return fmt.Errorf("creating template database %s: %w", name, err)
	}

	scriptPath := "/tmp/" + name + ".sql"
	if err := s.container.CopyFileToContainer(ctx, initSQLScript, scriptPath, 0o644); err != nil {
		return fmt.Errorf("copying init script %s: %w", initSQLScript, err)
	}

	cmd := []string{"psql", "-v", "ON_ERROR_STOP=1", "-U", dbUsername, "-d", name, "-f", scriptPath}
	code, output, err := s.container.Exec(ctx, cmd, tcexec.Multiplexed())
	if err != nil {
		return fmt.Errorf("running init script %s: %w", initSQLScript, err)
	}
	if code != 0 {
		out, _ := io.ReadAll(output)
		return fmt.Errorf("running init script %s: exit code %d: %s", initSQLScript, code, out)
	}

	return nil
}

func (s *postgresServer) dropDatabase(ctx context.Context, name string) error {
	_, err := s.admin.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", name))
	return err
}

func (s *postgresServer) close() {
	if s.admin != nil {
		_ = s.admin.Close()
	}

	if s.container != nil {
		if err := s.container.Terminate(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "error detected during container termination: %v\n", err)
		}
	}
}

func postgresVersion(version string) string {
	if version == "" {
		return defaultPostgresVersion
	}

	return version
}

// fixturePath returns the absolute path of a file under the 'fixtures' directory next to the calling test file.
func fixturePath(name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", nil
	}

	executionPath, err := testpath()
	if err != nil {
		return "", fmt.Errorf("resolving fixture path: %w", err)
	}

	return filepath.Join(executionPath, "fixtures", name), nil
}

// setupPostgresDB sets up a postgres database test container.
// Init SQL scripts are mounted into /docker-entrypoint-initdb.d/ so Postgres
// executes them during its own initialization, before accepting TCP connections.
func setupPostgresDB(ctx context.Context, version, initSQLScript string) (*PostgresDBContainer, error) {
	req := testcontainers.ContainerRequest{
		Image: "postgres:" + postgresVersion(version),
		Env: map[string]string{
			"POSTGRES_USER":     dbUsername,
			"POSTGRES_PASSWORD": dbPassword,
		},
		ExposedPorts: []string{dbPort},
		WaitingFor:   wait.ForSQL(dbPort, "postgres", dbURL),
	}

	if initSQLScript != "" {
		req.Files = []testcontainers.ContainerFile{
			{
				HostFilePath:      initSQLScript,
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
		DBName:     dbName,
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
