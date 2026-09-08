package tui

import (
	"encoding/csv"
	"os"
	"reflect"
	"testing"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
)

func TestExportBatchCSV_IncludesUnionOfProperties(t *testing.T) {
	path := t.TempDir() + "/properties.csv"
	rows := []repoRow{
		{owner: "acme", repo: "one", properties: []ghclient.PropertyValue{
			{Name: "team", Value: "platform"},
			{Name: "tags", Value: []any{"go", "cli"}},
			{Name: "owner", Value: "team@example.com"},
		}},
		{owner: "acme", repo: "two", properties: []ghclient.PropertyValue{
			{Name: "team", Value: nil},
		}},
		{owner: "other", repo: "failed", err: os.ErrNotExist},
	}
	schemas := map[string][]ghclient.PropertyDefinition{
		"acme":  {{Name: "lifecycle"}, {Name: "team"}},
		"other": {{Name: "owner"}},
	}

	if err := exportBatchCSV(path, rows, schemas); err != nil {
		t.Fatalf("exportBatchCSV: %v", err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open export: %v", err)
	}
	defer func() { _ = file.Close() }()
	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	want := [][]string{
		{"owner", "repo", "repo_url", "lifecycle", "property_owner", "tags", "team"},
		{"acme", "one", "https://github.com/acme/one", "", "team@example.com", "go, cli", "platform"},
		{"acme", "two", "https://github.com/acme/two", "", "", "", ""},
		{"other", "failed", "https://github.com/other/failed", "", "", "", ""},
	}
	if !reflect.DeepEqual(records, want) {
		t.Errorf("CSV records = %#v, want %#v", records, want)
	}
}

func TestBatchModel_ExportCommandWritesTimestampedCSV(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PWD", workingDir)
	// The command writes relative to the process working directory. Use a
	// temporary process directory by changing it only for this test.
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(workingDir) })

	m := newTableModel(t, &fakeAPI{}, []repoRow{{owner: "acme", repo: "one"}}, nil)
	next, _ := m.Update(runeKey('x'))
	m = settleBatch(next.(*batchModel))
	if m.screen != bScreenExporting {
		t.Fatalf("after export command: screen=%v, want exporting", m.screen)
	}
	msg := m.exportCmd()()
	next, _ = m.Update(msg)
	m = next.(*batchModel)
	if m.screen != bScreenExportResult || m.exportErr != nil {
		t.Fatalf("export result: screen=%v err=%v", m.screen, m.exportErr)
	}
	if _, err := os.Stat(m.exportPath); err != nil {
		t.Fatalf("export file %q: %v", m.exportPath, err)
	}
}
