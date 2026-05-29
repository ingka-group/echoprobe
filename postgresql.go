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
	"io"
	"strings"
	"time"

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
func setupPostgresDB(ctx context.Context, initSQLScript ...string) (*PostgresDBContainer, error) {
	req := testcontainers.ContainerRequest{
		Image: "postgres:latest",
		Env: map[string]string{
			"POSTGRES_USER":     dbUsername,
			"POSTGRES_PASSWORD": dbPassword,
		},
		ExposedPorts: []string{dbPort},
		// The Postgres Docker image has a multi-phase startup: it starts temporarily to
		// initialize the database cluster, shuts down, then starts again for real.
		// wait.ForSQL can latch onto the first (temporary) startup, causing subsequent
		// commands to fail during the restart window. Waiting for the "ready" log message
		// twice ensures we only proceed after the final startup.
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	// If init script path is provided, initialize the database using the script.
	if strings.TrimSpace(initSQLScript[0]) != "" {
		err = initDB(container, initSQLScript[0])
		if err != nil {
			return nil, err
		}
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

// initDB initializes the database using the provided script.
func initDB(container testcontainers.Container, filename string) error {
	executionPath, err := testpath()
	if err != nil {
		return err
	}

	containerPath := fmt.Sprintf("/%s", filename)

	// Copy the script from the host path to the container
	err = container.CopyFileToContainer(
		context.Background(),
		fmt.Sprintf(
			"%s/fixtures/%s", executionPath, filename,
		),
		containerPath,
		0544,
	)
	if err != nil {
		return err
	}

	// Execute the script with ON_ERROR_STOP so psql exits non-zero on any SQL error.
	exitCode, output, err := container.Exec(context.Background(), []string{
		"bash",
		"-c",
		fmt.Sprintf(
			"export PGPASSWORD=%s && psql --set ON_ERROR_STOP=1 -U %s -d %s -f %s",
			dbPassword, dbUsername, dbName, containerPath,
		),
	})
	if err != nil {
		return fmt.Errorf("initDB exec failed: %w", err)
	}

	if exitCode != 0 {
		out, _ := io.ReadAll(output)
		return fmt.Errorf("initDB script exited with code %d: %s", exitCode, string(out))
	}

	return nil
}
