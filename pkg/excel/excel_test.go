package excel

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestExport(t *testing.T) {
	f, err := Export("users", []string{"ID", "Name"}, [][]any{
		{1, "alice"},
		{2, "bob"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	value, err := f.GetCellValue("users", "B2")
	if err != nil || value != "alice" {
		t.Fatalf("B2 = %q, %v", value, err)
	}
	if index, _ := f.GetSheetIndex("Sheet1"); index != -1 {
		t.Fatal("default sheet should be removed")
	}
}

func TestSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.xlsx")
	if err := Save(path, "data", []string{"A"}, [][]any{{1}}); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	value, err := f.GetCellValue("data", "A2")
	if err != nil || value != "1" {
		t.Fatalf("A2 = %q, %v", value, err)
	}
}

func TestRowTooLong(t *testing.T) {
	if _, err := Export("s", []string{"A"}, [][]any{{1, 2}}); err == nil {
		t.Fatal("expected row length error")
	}
}
