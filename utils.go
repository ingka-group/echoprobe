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
	"errors"
	"path/filepath"
	"runtime"
	"strings"
)

// testpath returns a full path for the directory of a test file that called this function,
// so it can be used to build a path to binary files like fixtures next to the test files,
// which gives us an option to store fixtures in the same package with test.
func testpath() (string, error) {
	for i := 0; i < 32; i++ {
		_, caller, _, ok := runtime.Caller(i)
		if ok && strings.HasSuffix(caller, "_test.go") {
			return filepath.Dir(caller), nil
		}
	}

	return "", errors.New("cannot determine filesystem path for current test file")
}
