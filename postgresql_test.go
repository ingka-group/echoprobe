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
	"strings"
	"testing"
)

func TestPostgresOptions_Conflict(t *testing.T) {
	shared := PostgresOptions{InitSQLScript: "init.sql"}

	tests := []struct {
		name      string
		option    IntegrationTestWithPostgres
		wantError []string
	}{
		{
			name:   "ok: empty options inherit the package setup",
			option: IntegrationTestWithPostgres{},
		},
		{
			name:   "ok: explicit default version matches an empty package version",
			option: IntegrationTestWithPostgres{Version: defaultPostgresVersion},
		},
		{
			name:   "ok: same init script",
			option: IntegrationTestWithPostgres{InitSQLScript: "init.sql"},
		},
		{
			name:      "error: different version",
			option:    IntegrationTestWithPostgres{Version: "16-alpine"},
			wantError: []string{`"16-alpine"`, `"17-alpine"`},
		},
		{
			name:      "error: different init script",
			option:    IntegrationTestWithPostgres{InitSQLScript: "other.sql"},
			wantError: []string{`"other.sql"`, `"init.sql"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := shared.conflict(tt.option)

			if tt.wantError == nil {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			for _, want := range tt.wantError {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("expected error to contain %s, got: %v", want, err)
				}
			}
		})
	}
}
