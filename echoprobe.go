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
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/testcontainers/testcontainers-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// IntegrationTest is a struct that holds all the necessary information for integration testing.
type IntegrationTest struct {
	T           *testing.T
	Db          interface{}
	Echo        *echo.Echo
	Fixtures    *Fixtures
	Container   *PostgresDBContainer
	BqContainer *BigqueryEmulatorContainer
	Mock        *Mock

	opts []IntegrationTestOption
}

// NewIntegrationTest prepares database for integration testing.
func NewIntegrationTest(t *testing.T, opts ...IntegrationTestOption) *IntegrationTest {
	it := &IntegrationTest{
		T:        t,
		Echo:     echo.New(),
		Fixtures: &Fixtures{},
	}

	for _, o := range opts {
		o.setup(it)
	}

	it.opts = opts

	return it
}

// TearDown cleans up the database after integration testing.
func (it *IntegrationTest) TearDown() {
	for _, o := range it.opts {
		o.tearDown(it)
	}
}

// IntegrationTestOption is an interface for integration test options.
type IntegrationTestOption interface {
	setup(*IntegrationTest)
	tearDown(*IntegrationTest)
}

// IntegrationTestWithPostgres is an option for integration testing that sets up a postgres database test container.
// In the InitSQLScript a SQL script filename can be passed to initialize the database. The script should be located
// under a 'fixtures' directory where the _test.go file is located. An optional gorm config can also be passed.
//
// When Ryuk is enabled, or the package runs its tests through RunWithPostgres, tests share one container per Version.
// Each test then gets its own database, copied from a template database per InitSQLScript, and the database is
// dropped when the test finishes.
type IntegrationTestWithPostgres struct {
	Version       string
	InitSQLScript string
	Config        *gorm.Config
}

func (o IntegrationTestWithPostgres) setup(it *IntegrationTest) {
	// sanity check
	if o.Config == nil {
		o.Config = &gorm.Config{}
	}

	var initSQLScript string
	if strings.TrimSpace(o.InitSQLScript) != "" {
		executionPath, err := testpath()
		if err != nil {
			it.T.Fatalf("database setup error: resolving fixture path: %v", err)
		}
		initSQLScript = filepath.Join(executionPath, "fixtures", o.InitSQLScript)
	}

	// Without RunWithPostgres, nothing stops a shared container, so sharing relies on Ryuk to remove it.
	var dbContainer *PostgresDBContainer
	var err error
	if sharedPostgres.managed.Load() || !testcontainers.ReadConfig().Config.RyukDisabled {
		dbContainer, err = sharedPostgres.createDatabase(it.T, o.Version, initSQLScript)
	} else {
		dbContainer, err = setupPostgresDB(context.Background(), o.Version, initSQLScript)
	}
	if err != nil {
		it.T.Fatalf("database setup error: %v", err)
	}

	it.Container = dbContainer

	dsn := databaseURL(dbContainer.DBHost, strconv.Itoa(dbContainer.DBPort), dbContainer.DBName)
	db, err := gorm.Open(postgres.Open(dsn), o.Config)
	if err != nil {
		it.T.Fatalf("database connection error: %v", err)
	}

	if dbContainer.shared {
		it.T.Cleanup(func() {
			if sqlDB, err := db.DB(); err == nil {
				_ = sqlDB.Close()
			}
		})
	}

	it.Db = db
}

func (o IntegrationTestWithPostgres) tearDown(it *IntegrationTest) {
	// A shared container outlives the test; its database is dropped by the cleanup registered in setup.
	if it.Container.shared {
		return
	}

	err := it.Container.Terminate(context.Background())
	if err != nil {
		it.T.Logf("error detected during container termination: %v", err)
	}
}

// IntegrationTestWithMocks is an option for integration testing that allows mocking
// The mocks should be placed in a 'mocks' directory where the _test.go file is located.
type IntegrationTestWithMocks struct {
	BaseURL string
}

func (o IntegrationTestWithMocks) setup(it *IntegrationTest) {
	it.Mock = NewMock(o.BaseURL)
}

func (o IntegrationTestWithMocks) tearDown(it *IntegrationTest) {
	it.Mock.TearDown()
}

// IntegrationTestWithBigQuery is an option for integration testing that sets up a BigQuery database test container.
type IntegrationTestWithBigQuery struct {
	Version  string
	DataPath string
}

func (o IntegrationTestWithBigQuery) setup(it *IntegrationTest) {
	container, err := setupBigqueryEmulator(context.Background(), o.Version, o.DataPath)
	if err != nil {
		it.T.Fatalf("database setup error: %v", err)
	}

	it.BqContainer = container
}

func (o IntegrationTestWithBigQuery) tearDown(it *IntegrationTest) {
	err := it.BqContainer.Terminate(context.Background())
	if err != nil {
		it.T.Logf("error detected during container termination: %v", err)
	}
}
