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
	"bytes"

	"github.com/xuri/excelize/v2"
)

// excelToMap unloads data from Excel spreadsheet as a map with sheet name as a key and rows as values.
func excelToMap(excelFile *excelize.File) (map[string][][]string, error) {
	data := make(map[string][][]string)
	sheets := excelFile.GetSheetList()

	for _, sheet := range sheets {
		rows, err := excelFile.GetRows(sheet)
		if err != nil {
			return nil, err
		}

		data[sheet] = rows
	}

	return data, nil
}

// bytesToExcel parses a byte array into an excel file.
func bytesToExcel(content []byte) (*excelize.File, error) {
	reader := bytes.NewReader(content)

	file, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, err
	}
	return file, nil
}
