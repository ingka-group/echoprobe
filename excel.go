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

// readFile reads an excel from a file.
func readFile(filePath string) (*excelize.File, error) {
	file, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	return file, nil
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
