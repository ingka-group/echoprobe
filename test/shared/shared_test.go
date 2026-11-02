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
	"strings"
	"sync"
	"testing"

	"github.com/ingka-group/echoprobe"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// containerIDs maps each Postgres version to the IDs of the containers that served it.
var (
	containerIDsMu sync.Mutex
	containerIDs   = map[string]map[string]struct{}{}
)

func TestMain(m *testing.M) {
	code := echoprobe.RunWithPostgres(m)

	for version, ids := range containerIDs {
		if code == 0 && len(ids) > 1 {
			fmt.Fprintf(os.Stderr, "expected one shared container for version %q, got %d\n", version, len(ids))
			code = 1
		}
	}

	os.Exit(code)
}

func newTest(t *testing.T, opts echoprobe.IntegrationTestWithPostgres) *gorm.DB {
	t.Helper()

	if testing.Short() {
		t.Skip("(skipped)")
	}

	it := echoprobe.NewIntegrationTest(t, opts)

	containerIDsMu.Lock()
	if containerIDs[opts.Version] == nil {
		containerIDs[opts.Version] = map[string]struct{}{}
	}
	containerIDs[opts.Version][it.Container.GetContainerID()] = struct{}{}
	containerIDsMu.Unlock()

	db, ok := it.Db.(*gorm.DB)
	if !ok {
		t.Fatal("failed to assert it.Db to *gorm.DB")
	}

	return db
}

func withInitScript() echoprobe.IntegrationTestWithPostgres {
	return echoprobe.IntegrationTestWithPostgres{InitSQLScript: "init.sql"}
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

func seededValue(t *testing.T, db *gorm.DB) string {
	t.Helper()

	var value string
	if err := db.Raw("SELECT value FROM items WHERE key = 'seeded'").Scan(&value).Error; err != nil {
		t.Fatalf("query seeded row: %v", err)
	}

	return value
}

func TestTemplate_InitScriptIsApplied(t *testing.T) {
	db := newTest(t, withInitScript())

	if got := seededValue(t, db); got != "from-init-script" {
		t.Errorf("expected seeded value %q, got %q", "from-init-script", got)
	}
}

func TestTemplate_EachInitScriptHasItsOwnTemplate(t *testing.T) {
	db := newTest(t, echoprobe.IntegrationTestWithPostgres{InitSQLScript: "other.sql"})

	if got := seededValue(t, db); got != "from-other-script" {
		t.Errorf("expected seeded value %q, got %q", "from-other-script", got)
	}
}

func TestTemplate_NoInitScriptGivesAnEmptyDatabase(t *testing.T) {
	db := newTest(t, echoprobe.IntegrationTestWithPostgres{})

	if db.Migrator().HasTable("items") {
		t.Error("expected no items table without an init script")
	}
}

// The two isolation tests insert the same primary key. If they shared a
// database, whichever runs second would fail with a duplicate key error.
func TestIsolation_FirstTestWritesRow(t *testing.T) {
	db := newTest(t, withInitScript())

	insertItem(t, db, "written-by-a-test")

	if got := countItems(t, db); got != 2 {
		t.Errorf("expected the seeded row and our own row, got %d rows", got)
	}
}

func TestIsolation_SecondTestDoesNotSeeIt(t *testing.T) {
	db := newTest(t, withInitScript())

	insertItem(t, db, "written-by-a-test")

	if got := countItems(t, db); got != 2 {
		t.Errorf("expected the seeded row and our own row, got %d rows", got)
	}
}

func TestIsolation_ParallelTests(t *testing.T) {
	for i := range 10 {
		t.Run(fmt.Sprintf("worker-%d", i), func(t *testing.T) {
			t.Parallel()

			db := newTest(t, withInitScript())

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
		db := newTest(t, withInitScript())
		if err := db.Raw("SELECT current_database()").Scan(&testDB).Error; err != nil {
			t.Fatalf("query current database: %v", err)
		}
	})

	db := newTest(t, withInitScript())

	var exists bool
	if err := db.Raw("SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = ?)", testDB).Scan(&exists).Error; err != nil {
		t.Fatalf("query pg_database: %v", err)
	}

	if exists {
		t.Errorf("expected database %q to be dropped after its test finished", testDB)
	}
}

func TestDatabase_GormConfigIsApplied(t *testing.T) {
	db := newTest(t, echoprobe.IntegrationTestWithPostgres{
		Config: &gorm.Config{NamingStrategy: schema.NamingStrategy{TablePrefix: "app_"}},
	})

	if got := db.NamingStrategy.TableName("Item"); got != "app_items" {
		t.Errorf("expected table name %q from the custom naming strategy, got %q", "app_items", got)
	}
}

func TestVersion_EachVersionHasItsOwnContainer(t *testing.T) {
	db := newTest(t, echoprobe.IntegrationTestWithPostgres{Version: "16-alpine", InitSQLScript: "init.sql"})

	var version string
	if err := db.Raw("SHOW server_version").Scan(&version).Error; err != nil {
		t.Fatalf("query server version: %v", err)
	}

	if !strings.HasPrefix(version, "16.") {
		t.Errorf("expected Postgres 16, got %q", version)
	}

	if got := seededValue(t, db); got != "from-init-script" {
		t.Errorf("expected seeded value %q, got %q", "from-init-script", got)
	}
}
