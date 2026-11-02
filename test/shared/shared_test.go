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

package shared

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/ingka-group/echoprobe"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var containerIDs sync.Map

func TestMain(m *testing.M) {
	code := echoprobe.RunWithPostgres(m, echoprobe.PostgresOptions{InitSQLScript: "init.sql"})

	var count int
	containerIDs.Range(func(_, _ any) bool {
		count++
		return true
	})
	if code == 0 && count > 1 {
		fmt.Fprintf(os.Stderr, "expected one shared Postgres container, got %d\n", count)
		code = 1
	}

	os.Exit(code)
}

func newTest(t *testing.T, opts ...echoprobe.IntegrationTestOption) (*echoprobe.IntegrationTest, *gorm.DB) {
	t.Helper()

	if testing.Short() {
		t.Skip("(skipped)")
	}

	if len(opts) == 0 {
		opts = []echoprobe.IntegrationTestOption{echoprobe.IntegrationTestWithPostgres{}}
	}

	it := echoprobe.NewIntegrationTest(t, opts...)
	containerIDs.Store(it.Container.GetContainerID(), struct{}{})

	db, ok := it.Db.(*gorm.DB)
	if !ok {
		t.Fatal("failed to assert it.Db to *gorm.DB")
	}

	return it, db
}

func insertItem(t *testing.T, db *gorm.DB, key string) {
	t.Helper()

	if err := db.Exec("INSERT INTO items (key, value) VALUES (?, ?)", key, t.Name()).Error; err != nil {
		t.Fatalf("insert %q: %v", key, err)
	}
}

func countItems(t *testing.T, db *gorm.DB) int64 {
	t.Helper()

	var count int64
	if err := db.Raw("SELECT count(*) FROM items").Scan(&count).Error; err != nil {
		t.Fatalf("count items: %v", err)
	}

	return count
}

func TestTemplate_InitScriptIsApplied(t *testing.T) {
	_, db := newTest(t)

	var value string
	if err := db.Raw("SELECT value FROM items WHERE key = 'seeded'").Scan(&value).Error; err != nil {
		t.Fatalf("query seeded row: %v", err)
	}

	if value != "from-init-script" {
		t.Errorf("expected seeded value %q, got %q", "from-init-script", value)
	}
}

// The two isolation tests insert the same primary key. If they shared a
// database, whichever runs second would fail with a duplicate key error.
func TestIsolation_FirstTestWritesRow(t *testing.T) {
	_, db := newTest(t)

	insertItem(t, db, "written-by-a-test")

	if got := countItems(t, db); got != 2 {
		t.Errorf("expected the seeded row and our own row, got %d rows", got)
	}
}

func TestIsolation_SecondTestDoesNotSeeIt(t *testing.T) {
	_, db := newTest(t)

	insertItem(t, db, "written-by-a-test")

	if got := countItems(t, db); got != 2 {
		t.Errorf("expected the seeded row and our own row, got %d rows", got)
	}
}

func TestIsolation_ParallelTests(t *testing.T) {
	for i := range 10 {
		t.Run(fmt.Sprintf("worker-%d", i), func(t *testing.T) {
			t.Parallel()

			_, db := newTest(t)

			insertItem(t, db, "written-by-a-test")

			if got := countItems(t, db); got != 2 {
				t.Errorf("expected the seeded row and our own row, got %d rows", got)
			}
		})
	}
}

func TestDatabase_IsDroppedAfterTest(t *testing.T) {
	var testDB string
	t.Run("inner", func(t *testing.T) {
		it, _ := newTest(t)
		testDB = it.Container.DBName
	})

	_, db := newTest(t)

	var exists bool
	if err := db.Raw("SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = ?)", testDB).Scan(&exists).Error; err != nil {
		t.Fatalf("query pg_database: %v", err)
	}

	if exists {
		t.Errorf("expected database %q to be dropped after its test finished", testDB)
	}
}

func TestDatabase_GormConfigIsApplied(t *testing.T) {
	_, db := newTest(t, echoprobe.IntegrationTestWithPostgres{
		Config: &gorm.Config{NamingStrategy: schema.NamingStrategy{TablePrefix: "app_"}},
	})

	if got := db.NamingStrategy.TableName("Item"); got != "app_items" {
		t.Errorf("expected table name %q from the custom naming strategy, got %q", "app_items", got)
	}
}
