// Package excel provides small helpers for exporting tabular data with
// excelize.
package excel

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// Export creates an Excel workbook with one sheet. Header values are written to
// the first row; each row is written below it. Rows longer than headers are
// rejected.
func Export(sheet string, headers []string, rows [][]any) (*excelize.File, error) {
	if sheet == "" {
		sheet = "Sheet1"
	}
	f := excelize.NewFile()
	if sheet != "Sheet1" {
		if _, err := f.NewSheet(sheet); err != nil {
			return nil, err
		}
	}
	for column, header := range headers {
		cell, err := excelize.CoordinatesToCellName(column+1, 1)
		if err != nil {
			return nil, err
		}
		if err := f.SetCellValue(sheet, cell, header); err != nil {
			return nil, err
		}
	}
	for rowIndex, row := range rows {
		if len(row) > len(headers) {
			return nil, fmt.Errorf("excel: row %d has %d cells, headers have %d", rowIndex+1, len(row), len(headers))
		}
		for column, value := range row {
			cell, err := excelize.CoordinatesToCellName(column+1, rowIndex+2)
			if err != nil {
				return nil, err
			}
			if err := f.SetCellValue(sheet, cell, value); err != nil {
				return nil, err
			}
		}
	}
	if sheet != "Sheet1" {
		if index, err := f.GetSheetIndex(sheet); err == nil {
			f.SetActiveSheet(index)
		}
		_ = f.DeleteSheet("Sheet1")
	}
	return f, nil
}

// Save writes an exported workbook to path.
func Save(path, sheet string, headers []string, rows [][]any) error {
	f, err := Export(sheet, headers, rows)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.SaveAs(path)
}
