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
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/labstack/echo/v4"
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
type IntegrationTestWithPostgres struct {
	InitSQLScript string
	Config        *gorm.Config
}

func (o IntegrationTestWithPostgres) setup(it *IntegrationTest) {
	// sanity check
	if o.Config == nil {
		o.Config = &gorm.Config{}
	}

	dbContainer, err := setupPostgresDB(context.Background(), o.InitSQLScript)
	if err != nil {
		it.T.Fatalf("database setup error: %v", err)
	}

	it.Container = dbContainer

	dsn := dbURL(dbContainer.DBHost, nat.Port(fmt.Sprintf("%d/tcp", dbContainer.DBPort)))
	db, err := gorm.Open(postgres.Open(dsn), o.Config)
	if err != nil {
		it.T.Fatalf("database connection error: %v", err)
	}

	it.Db = db
}

func (o IntegrationTestWithPostgres) tearDown(it *IntegrationTest) {
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
	DataPath string
}

func (o IntegrationTestWithBigQuery) setup(it *IntegrationTest) {
	container, err := setupBigqueryEmulator(context.Background(), o.DataPath)
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
