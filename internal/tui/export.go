package tui

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
)

// exportBatchCSV writes every batch row and the union of all known property
// fields. Schema fields are included even when a repo has not set a value.
// Rows whose fetch failed are still exported with blank property values.
func exportBatchCSV(path string, rows []repoRow, schemaByOrg map[string][]ghclient.PropertyDefinition) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	columns := map[string]bool{}
	for _, row := range rows {
		for _, property := range row.properties {
			columns[property.Name] = true
		}
		for _, definition := range schemaByOrg[row.owner] {
			columns[definition.Name] = true
		}
	}
	propertyNames := make([]string, 0, len(columns))
	for name := range columns {
		propertyNames = append(propertyNames, name)
	}
	sort.Strings(propertyNames)
	propertyHeaders := uniquePropertyHeaders(propertyNames)

	record := make([]string, 3+len(propertyHeaders))
	record[0], record[1], record[2] = "owner", "repo", "repo_url"
	copy(record[3:], propertyHeaders)
	writer := csv.NewWriter(file)
	if err := writer.Write(record); err != nil {
		return err
	}

	for _, row := range rows {
		record = make([]string, 3+len(propertyNames))
		record[0], record[1] = row.owner, row.repo
		record[2] = "https://github.com/" + row.owner + "/" + row.repo
		values := make(map[string]any, len(row.properties))
		for _, property := range row.properties {
			values[property.Name] = property.Value
		}
		for i, name := range propertyNames {
			record[i+3] = csvValue(values[name])
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}
	return nil
}

func uniquePropertyHeaders(propertyNames []string) []string {
	reserved := map[string]bool{"owner": true, "repo": true, "repo_url": true}
	headers := make([]string, len(propertyNames))
	used := make(map[string]bool, len(propertyNames)+len(reserved))
	for name := range reserved {
		used[name] = true
	}
	for i, name := range propertyNames {
		header := name
		if reserved[name] {
			header = "property_" + name
		}
		for used[header] {
			header = "property_" + header
		}
		headers[i] = header
		used[header] = true
	}
	return headers
}

func csvValue(value any) string {
	switch value := value.(type) {
	case nil:
		return ""
	case string:
		return value
	case []string:
		return strings.Join(value, ", ")
	case []any:
		parts := make([]string, len(value))
		for i, item := range value {
			parts[i] = csvValue(item)
		}
		return strings.Join(parts, ", ")
	case bool:
		return strconv.FormatBool(value)
	default:
		return fmt.Sprint(value)
	}
}
