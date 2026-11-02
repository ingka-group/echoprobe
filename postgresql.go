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

	"github.com/testcontainers/testcontainers-go"
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

func postgresVersion(version string) string {
	if version == "" {
		return defaultPostgresVersion
	}

	return version
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
		WaitingFor: wait.ForSQL(dbPort, "postgres", func(host, port string) string {
			return databaseURL(host, port, dbName)
		}),
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

// databaseURL returns the URL of the given postgres database.
func databaseURL(host, port, database string) string {
	port, _, _ = strings.Cut(port, "/")
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUsername, dbPassword, host, port, database,
	)
}
