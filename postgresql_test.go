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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPostgresServer_BrokenInitScriptFailsWithPsqlOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("(skipped)")
	}

	script := filepath.Join(t.TempDir(), "broken.sql")
	if err := os.WriteFile(script, []byte("CREATE TABLE ok (id INT);\nSELECT * FROM missing_table;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	server := &postgresServer{}
	t.Cleanup(server.close)

	_, err := server.createDatabase(context.Background(), script)
	if err == nil {
		t.Fatal("expected an error for a broken init script, got nil")
	}

	if !strings.Contains(err.Error(), "missing_table") {
		t.Errorf("expected the psql error in the message, got: %v", err)
	}
}
