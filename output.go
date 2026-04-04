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

const maxGenericTableColumns = 8

type renderOptions struct {
	View string
}

type tableColumn struct {
	Header  string
	Extract func(map[string]any) string
}

type tablePreset struct {
	Columns []tableColumn
}

var genericColumnPriority = []string{
	"id",
	"ref",
	"name",
	"slug",
	"subject",
	"username",
	"full_name_display",
	"status",
	"project",
	"owner",
	"assigned_to",
	"closed",
	"is_closed",
	"due_date",
	"estimated_start",
	"estimated_finish",
	"created_date",
	"modified_date",
}

var tablePresets = map[string]tablePreset{
	"projects": {
		Columns: []tableColumn{
			pathColumn("ID", "id"),
			pathColumn("Name", "name"),
			pathColumn("Slug", "slug"),
			personColumn("Owner", "owner"),
		},
	},
	"user-stories": {
		Columns: []tableColumn{
			pathColumn("ID", "id"),
			pathColumn("Subject", "subject"),
			personColumn("Assigned To", "assigned_to_extra_info", "assigned_to"),
			pathColumn("Assigned Users", "assigned_users"),
			personColumn("Owner", "owner_extra_info", "owner"),
			pathColumn("Status", "status_extra_info.name", "status"),
			pathColumn("Project", "project_extra_info.name", "project"),
			pathColumn("Slug", "project_extra_info.slug"),
		},
	},
	"epics": {
		Columns: []tableColumn{
			pathColumn("ID", "id"),
			pathColumn("Subject", "subject"),
			personColumn("Assigned To", "assigned_to_extra_info", "assigned_to"),
			personColumn("Owner", "owner_extra_info", "owner"),
			pathColumn("Status", "status_extra_info.name", "status"),
			pathColumn("Project", "project_extra_info.name", "project"),
			pathColumn("Slug", "project_extra_info.slug"),
			pathColumn("User Stories", "user_stories_counts.total"),
		},
	},
	"tasks": {
		Columns: []tableColumn{
			pathColumn("ID", "id"),
			pathColumn("Subject", "subject"),
			personColumn("Assigned To", "assigned_to_extra_info", "assigned_to"),
			personColumn("Owner", "owner_extra_info", "owner"),
			pathColumn("Status", "status_extra_info.name", "status"),
			pathColumn("User Story", "user_story_extra_info.subject", "user_story"),
			pathColumn("Project", "project_extra_info.name", "project"),
			pathColumn("Slug", "project_extra_info.slug"),
		},
	},
	"milestones": {
		Columns: []tableColumn{
			pathColumn("ID", "id"),
			pathColumn("Name", "name"),
			pathColumn("Slug", "slug"),
			pathColumn("Project", "project_extra_info.name", "project"),
			pathColumn("Closed", "closed"),
			pathColumn("Estimated Start", "estimated_start"),
			pathColumn("Estimated Finish", "estimated_finish"),
		},
	},
}

func render(output string, value any) error {
	return renderWithOptions(output, value, renderOptions{})
}

func renderView(output string, value any, view string) error {
	return renderWithOptions(output, value, renderOptions{View: view})
}

func renderWithOptions(output string, value any, options renderOptions) error {
	output = strings.TrimSpace(output)
	if output == "" {
		output = defaultOutputFormat
	}

	switch output {
	case "json":
		return renderJSON(os.Stdout, value)
	case "yaml":
		return renderYAML(os.Stdout, value)
	case "table":
		return renderTable(os.Stdout, value, options)
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

func renderTable(w io.Writer, value any, options renderOptions) error {
	switch typed := value.(type) {
	case PaginationEnvelope:
		rendered, err := renderTableRows(w, typed.Items, options)
		if err != nil {
			return err
		}
		if !rendered {
			return renderJSON(w, value)
		}
		return renderTableSummary(w, typed.Pagination, typed.Extra)
	default:
		rendered, err := renderTableRows(w, value, options)
		if err != nil {
			return err
		}
		if !rendered {
			return renderJSON(w, value)
		}
		return nil
	}
}

func renderTableRows(w io.Writer, value any, options renderOptions) (bool, error) {
	rows, headers := tabularise(value, options)
	if len(headers) == 0 {
		return false, nil
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
	return true, nil
}

func renderTableSummary(w io.Writer, pagination PaginationView, extra map[string]any) error {
	wroteSummary := false
	if pagination.Paginated {
		message := "\nPagination enabled"
		if pagination.Page > 0 {
			message += fmt.Sprintf(" (page %d", pagination.Page)
			if pagination.Count > 0 {
				message += fmt.Sprintf(", count %d", pagination.Count)
			}
			if pagination.PageSize > 0 {
				message += fmt.Sprintf(", page size %d", pagination.PageSize)
			}
			message += ")"
		} else if pagination.PageSize > 0 {
			message += fmt.Sprintf(" (page size %d)", pagination.PageSize)
		}
		message += "\n"
		if _, err := fmt.Fprint(w, message); err != nil {
			return err
		}
		wroteSummary = true
	}
	if len(extra) == 0 {
		return nil
	}
	if !wroteSummary {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	keys := make([]string, 0, len(extra))
	for key := range extra {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, err := fmt.Fprintf(w, "%s: %s\n", humanizeHeader(key), fmtScalar(extra[key])); err != nil {
			return err
		}
	}
	return nil
}

func tabularise(value any, options renderOptions) ([]table.Row, []string) {
	if preset, ok := tablePresets[options.View]; ok {
		if rows, headers, ok := rowsFromPreset(value, preset); ok {
			return rows, headers
		}
	}

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

func rowsFromPreset(value any, preset tablePreset) ([]table.Row, []string, bool) {
	if item, ok := asJSONMap(value); ok {
		return []table.Row{rowFromPreset(item, preset)}, presetHeaders(preset), true
	}
	if items, ok := asJSONMaps(value); ok {
		rows := make([]table.Row, 0, len(items))
		for _, item := range items {
			rows = append(rows, rowFromPreset(item, preset))
		}
		return rows, presetHeaders(preset), true
	}
	return nil, nil, false
}

func rowFromPreset(item map[string]any, preset tablePreset) table.Row {
	row := make(table.Row, 0, len(preset.Columns))
	for _, column := range preset.Columns {
		row = append(row, column.Extract(item))
	}
	return row
}

func presetHeaders(preset tablePreset) []string {
	headers := make([]string, 0, len(preset.Columns))
	for _, column := range preset.Columns {
		headers = append(headers, column.Header)
	}
	return headers
}

func rowsFromMaps(items []map[string]any) ([]table.Row, []string) {
	keys := genericHeaders(items)
	if len(keys) == 0 {
		return nil, nil
	}

	rows := make([]table.Row, 0, len(items))
	for _, item := range items {
		row := make(table.Row, 0, len(keys))
		for _, key := range keys {
			row = append(row, fmtScalar(item[key]))
		}
		rows = append(rows, row)
	}

	headers := make([]string, 0, len(keys))
	for _, key := range keys {
		headers = append(headers, humanizeHeader(key))
	}
	return rows, headers
}

func genericHeaders(items []map[string]any) []string {
	keys := map[string]struct{}{}
	for _, item := range items {
		for key, value := range item {
			if isColumnValue(value) {
				keys[key] = struct{}{}
			}
		}
	}
	if len(keys) == 0 {
		return nil
	}

	headers := make([]string, 0, len(keys))
	for _, priority := range genericColumnPriority {
		if _, ok := keys[priority]; ok {
			headers = append(headers, priority)
			delete(keys, priority)
		}
	}

	remaining := make([]string, 0, len(keys))
	for key := range keys {
		remaining = append(remaining, key)
	}
	sort.Strings(remaining)
	headers = append(headers, remaining...)
	if len(headers) > maxGenericTableColumns {
		headers = headers[:maxGenericTableColumns]
	}
	return headers
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

func pathColumn(header string, paths ...string) tableColumn {
	return tableColumn{
		Header: header,
		Extract: func(item map[string]any) string {
			for _, path := range paths {
				if value, ok := valueAtPath(item, path); ok {
					if formatted := fmtScalar(value); formatted != "" {
						return formatted
					}
				}
			}
			return ""
		},
	}
}

func personColumn(header string, paths ...string) tableColumn {
	return tableColumn{
		Header: header,
		Extract: func(item map[string]any) string {
			for _, path := range paths {
				if value, ok := valueAtPath(item, path); ok {
					if formatted := formatIdentity(value); formatted != "" {
						return formatted
					}
				}
			}
			return ""
		},
	}
}

func formatIdentity(value any) string {
	info, ok := value.(map[string]any)
	if !ok {
		return fmtScalar(value)
	}

	parts := []string{}
	for _, key := range []string{"id", "full_name_display", "username"} {
		if value, ok := info[key]; ok {
			if formatted := fmtScalar(value); formatted != "" {
				parts = append(parts, formatted)
			}
		}
	}
	return strings.Join(parts, " / ")
}

func valueAtPath(item map[string]any, path string) (any, bool) {
	if path == "" {
		return nil, false
	}
	current := any(item)
	for _, part := range strings.Split(path, ".") {
		asMap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := asMap[part]
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func asJSONMap(value any) (map[string]any, bool) {
	if typed, ok := value.(map[string]any); ok {
		return typed, true
	}
	payload, err := mustJSONMap(value)
	if err != nil {
		return nil, false
	}
	return payload, true
}

func asJSONMaps(value any) ([]map[string]any, bool) {
	if typed, ok := value.([]map[string]any); ok {
		return typed, true
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	var items []map[string]any
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, false
	}
	return items, true
}

func isColumnValue(value any) bool {
	if isScalar(value) {
		return true
	}
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if !isScalar(item) {
				return false
			}
		}
		return true
	case []string, []int, []int64, []float64:
		return true
	default:
		return false
	}
}

func humanizeHeader(key string) string {
	parts := strings.Fields(strings.NewReplacer("_", " ", "-", " ").Replace(key))
	for i, part := range parts {
		switch strings.ToLower(part) {
		case "id":
			parts[i] = "ID"
		case "api":
			parts[i] = "API"
		case "url":
			parts[i] = "URL"
		default:
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, " ")
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
	case int:
		return fmt.Sprintf("%d", typed)
	case int64:
		return fmt.Sprintf("%d", typed)
	case float64:
		if float64(int64(typed)) == typed {
			return fmt.Sprintf("%d", int64(typed))
		}
		return fmt.Sprintf("%v", typed)
	case []any:
		return joinScalarSlice(typed)
	case []string:
		return strings.Join(typed, ", ")
	case []int:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			items = append(items, fmt.Sprintf("%d", item))
		}
		return strings.Join(items, ", ")
	case []int64:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			items = append(items, fmt.Sprintf("%d", item))
		}
		return strings.Join(items, ", ")
	case []float64:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			items = append(items, fmtScalar(item))
		}
		return strings.Join(items, ", ")
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprint(value)
		}
		return strings.TrimSpace(string(data))
	}
}

func joinScalarSlice(items []any) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		formatted := fmtScalar(item)
		if formatted == "" {
			continue
		}
		parts = append(parts, formatted)
	}
	return strings.Join(parts, ", ")
}
