package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"gopkg.in/yaml.v3"
)

func render(output string, value any) error {
	switch output {
	case "", "json":
		return renderJSON(os.Stdout, value)
	case "yaml":
		return renderYAML(os.Stdout, value)
	case "table":
		return renderTable(os.Stdout, value)
	default:
		return fmt.Errorf("unsupported output format %q", output)
	}
}

func renderJSON(w io.Writer, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

func renderYAML(w io.Writer, value any) error {
	data, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, string(data))
	return err
}

func renderTable(w io.Writer, value any) error {
	switch typed := value.(type) {
	case PaginationEnvelope:
		if err := renderTableRows(w, typed.Items); err != nil {
			return err
		}
		if typed.Pagination.Paginated {
			message := "\nPagination enabled"
			if typed.Pagination.Page > 0 {
				message += fmt.Sprintf(" (page %d", typed.Pagination.Page)
				if typed.Pagination.Count > 0 {
					message += fmt.Sprintf(", count %d", typed.Pagination.Count)
				}
				if typed.Pagination.PageSize > 0 {
					message += fmt.Sprintf(", page size %d", typed.Pagination.PageSize)
				}
				message += ")"
			} else if typed.Pagination.PageSize > 0 {
				message += fmt.Sprintf(" (page size %d)", typed.Pagination.PageSize)
			}
			message += "\n"
			_, err := fmt.Fprint(w, message)
			return err
		}
		return nil
	default:
		return renderTableRows(w, value)
	}
}

func renderTableRows(w io.Writer, value any) error {
	rows, headers := tabularise(value)
	if len(headers) == 0 {
		return renderJSON(w, value)
	}

	t := table.NewWriter()
	t.SetOutputMirror(w)
	headerRow := make(table.Row, 0, len(headers))
	for _, header := range headers {
		headerRow = append(headerRow, header)
	}
	t.AppendHeader(headerRow)
	for _, row := range rows {
		t.AppendRow(row)
	}
	t.Render()
	return nil
}

func tabularise(value any) ([]table.Row, []string) {
	switch typed := value.(type) {
	case []map[string]any:
		return rowsFromMaps(typed)
	case map[string]any:
		return rowsFromKeyValueMap(typed)
	default:
		payload, err := mustJSONMap(value)
		if err == nil {
			return rowsFromKeyValueMap(payload)
		}
		data, err := json.Marshal(value)
		if err != nil {
			return nil, nil
		}
		var slice []map[string]any
		if err := json.Unmarshal(data, &slice); err == nil {
			return rowsFromMaps(slice)
		}
		var single map[string]any
		if err := json.Unmarshal(data, &single); err == nil {
			return rowsFromKeyValueMap(single)
		}
	}
	return nil, nil
}

func rowsFromMaps(items []map[string]any) ([]table.Row, []string) {
	keys := map[string]struct{}{}
	for _, item := range items {
		for key, value := range item {
			if isScalar(value) {
				keys[key] = struct{}{}
			}
		}
	}
	headers := make([]string, 0, len(keys))
	for key := range keys {
		headers = append(headers, key)
	}
	sort.Strings(headers)

	rows := make([]table.Row, 0, len(items))
	for _, item := range items {
		row := make(table.Row, 0, len(headers))
		for _, header := range headers {
			row = append(row, fmtScalar(item[header]))
		}
		rows = append(rows, row)
	}
	return rows, headers
}

func rowsFromKeyValueMap(item map[string]any) ([]table.Row, []string) {
	rows := make([]table.Row, 0, len(item))
	keys := make([]string, 0, len(item))
	for key := range item {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		rows = append(rows, table.Row{key, fmtScalar(item[key])})
	}
	return rows, []string{"field", "value"}
}

func fmtScalar(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case float64:
		if float64(int64(typed)) == typed {
			return fmt.Sprintf("%d", int64(typed))
		}
		return fmt.Sprintf("%v", typed)
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprint(value)
		}
		return strings.TrimSpace(string(data))
	}
}
