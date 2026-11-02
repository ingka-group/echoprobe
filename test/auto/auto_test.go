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

// Package auto has no TestMain, so container sharing depends on whether Ryuk is enabled.
package auto

import (
	"testing"

	"github.com/ingka-group/echoprobe"
	"github.com/testcontainers/testcontainers-go"
)

func TestSharing_FollowsRyukSetting(t *testing.T) {
	if testing.Short() {
		t.Skip("(skipped)")
	}

	first := echoprobe.NewIntegrationTest(t, echoprobe.IntegrationTestWithPostgres{})
	defer first.TearDown()
	second := echoprobe.NewIntegrationTest(t, echoprobe.IntegrationTestWithPostgres{})
	defer second.TearDown()

	sameContainer := first.Container.GetContainerID() == second.Container.GetContainerID()

	if testcontainers.ReadConfig().Config.RyukDisabled {
		if sameContainer {
			t.Error("expected a container per test when Ryuk is disabled")
		}
		return
	}

	if !sameContainer {
		t.Error("expected one shared container when Ryuk is enabled")
	}
	if first.Container.DBName == second.Container.DBName {
		t.Errorf("expected a database per test, both got %q", first.Container.DBName)
	}
}
