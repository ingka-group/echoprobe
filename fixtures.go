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
	"fmt"
	"log"
	"os"
)

// Fixtures is a helper for reading fixtures.
type Fixtures struct{}

// ReadResponse reads the response from a file.
func (f Fixtures) ReadResponse(s string) string {
	return f.ReadFixture(s+".json", "responses")
}

// ReadRequestBody reads the request body from a file.
func (f Fixtures) ReadRequestBody(s string) string {
	return f.ReadFixture(s+".json", "requests")
}

// ReadExcelFile reads an excel file with xlsx extension.
func (f Fixtures) ReadExcelFile(s string) map[string][][]string {
	content := f.ReadFixture(s+".xlsx", "excel")
	buf := []byte(content)
	file, err := f.ExcelToMap(buf)
	if err != nil {
		log.Fatalf("could not load excel file '%s': %v", s, err)
	}
	return file
}

// ReadExcelFile reads an excel file with xlsx extension.
func (f Fixtures) ReadCsvFile(s string) string {
	executionPath, err := testpath()
	if err != nil {
		log.Fatalf("could not file test path: %v", err)
	}

	path := fmt.Sprintf("%s/fixtures/csv/%s.csv", executionPath, s)
	buf, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("could not read file '%s': %v", path, err)
	}

	return string(buf)
}

// ReadFixture reads a fixture from a file.
func (f Fixtures) ReadFixture(filename, dir string) string {
	executionPath, err := testpath()
	if err != nil {
		log.Fatalf("could not file test path: %v", err)
	}

	path := fmt.Sprintf("%s/fixtures/%s/%s", executionPath, dir, filename)
	buf, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("could not read file '%s': %v", path, err)
	}

	return string(buf)
}

func (f Fixtures) ExcelToMap(content []byte) (map[string][][]string, error) {
	file, err := bytesToExcel(content)
	if err != nil {
		return nil, err
	}

	return excelToMap(file)
}
