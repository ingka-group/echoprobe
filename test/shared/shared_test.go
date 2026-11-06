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

func TestDatabase_StartsFromTheTemplateForItsOptions(t *testing.T) {
	tests := []struct {
		name        string
		opts        echoprobe.IntegrationTestWithPostgres
		wantVersion string
		wantSeeded  string // empty means no items table
	}{
		{
			name:        "init script is applied",
			opts:        withInitScript(),
			wantVersion: "17.",
			wantSeeded:  "from-init-script",
		},
		{
			name:        "each init script has its own template",
			opts:        echoprobe.IntegrationTestWithPostgres{InitSQLScript: "other.sql"},
			wantVersion: "17.",
			wantSeeded:  "from-other-script",
		},
		{
			name:        "no init script gives an empty database",
			opts:        echoprobe.IntegrationTestWithPostgres{},
			wantVersion: "17.",
		},
		{
			name:        "each version has its own container",
			opts:        echoprobe.IntegrationTestWithPostgres{Version: "16-alpine", InitSQLScript: "init.sql"},
			wantVersion: "16.",
			wantSeeded:  "from-init-script",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newTest(t, tt.opts)

			var version string
			if err := db.Raw("SHOW server_version").Scan(&version).Error; err != nil {
				t.Fatalf("query server version: %v", err)
			}
			if !strings.HasPrefix(version, tt.wantVersion) {
				t.Errorf("expected Postgres %s*, got %q", tt.wantVersion, version)
			}

			if tt.wantSeeded == "" {
				if db.Migrator().HasTable("items") {
					t.Error("expected no items table without an init script")
				}
				return
			}

			var seeded string
			if err := db.Raw("SELECT value FROM items WHERE key = 'seeded'").Scan(&seeded).Error; err != nil {
				t.Fatalf("query seeded row: %v", err)
			}
			if seeded != tt.wantSeeded {
				t.Errorf("expected seeded value %q, got %q", tt.wantSeeded, seeded)
			}
		})
	}
}

// Every subtest inserts the same primary key. If two subtests shared a
// database, the later one would fail with a duplicate key error.
func TestIsolation(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		parallel bool
	}{
		{name: "sequential", count: 2},
		{name: "parallel", count: 10, parallel: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := range tt.count {
				t.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
					if tt.parallel {
						t.Parallel()
					}

					db := newTest(t, withInitScript())

					if err := db.Exec("INSERT INTO items (key, value) VALUES ('written-by-a-test', ?)", t.Name()).Error; err != nil {
						t.Fatalf("insert: %v", err)
					}

					var count int64
					if err := db.Raw("SELECT count(*) FROM items").Scan(&count).Error; err != nil {
						t.Fatalf("count items: %v", err)
					}
					if count != 2 {
						t.Errorf("expected the seeded row and our own row, got %d rows", count)
					}
				})
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
