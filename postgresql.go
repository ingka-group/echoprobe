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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/lib/pq"
)

const (
	dbName     = "postgres"
	dbUsername = "postgres"
	dbPassword = "password"
	dbPort     = "5432/tcp"
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

// setupPostgresDB sets up a postgres database test container.
// Init SQL scripts are mounted into /docker-entrypoint-initdb.d/ so Postgres
// executes them during its own initialization, before accepting TCP connections.
func setupPostgresDB(ctx context.Context, initSQLScript ...string) (*PostgresDBContainer, error) {
	req := testcontainers.ContainerRequest{
		Image: "postgres:latest",
		Env: map[string]string{
			"POSTGRES_USER":     dbUsername,
			"POSTGRES_PASSWORD": dbPassword,
		},
		ExposedPorts: []string{dbPort},
		WaitingFor:   wait.ForSQL(dbPort, "postgres", dbURL),
	}

	if len(initSQLScript) > 0 && strings.TrimSpace(initSQLScript[0]) != "" {
		executionPath, err := testpath()
		if err != nil {
			return nil, fmt.Errorf("resolving fixture path: %w", err)
		}

		req.Files = []testcontainers.ContainerFile{
			{
				HostFilePath:      filepath.Join(executionPath, "fixtures", initSQLScript[0]),
				ContainerFilePath: "/docker-entrypoint-initdb.d/" + filepath.Base(initSQLScript[0]),
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
		DBPort:     mappedPort.Int(),
		DBName:     dbName,
		DBUsername: dbUsername,
		DBPassword: dbPassword,
	}, nil
}

// dbURL returns the postgres database URL.
func dbURL(host string, port nat.Port) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUsername, dbPassword, host, port.Port(), dbName,
	)
}
