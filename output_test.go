package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/jedib0t/go-pretty/v6/table"
	cli "github.com/urfave/cli/v3"
)

func TestNewAppDefaultOutputIsTable(t *testing.T) {
	t.Parallel()

	app, err := newApp()
	if err != nil {
		t.Fatalf("newApp returned error: %v", err)
	}

	for _, flag := range app.Flags {
		stringFlag, ok := flag.(*cli.StringFlag)
		if !ok || stringFlag.Name != "output" {
			continue
		}
		if stringFlag.Value != defaultOutputFormat {
			t.Fatalf("output flag default = %q, want %q", stringFlag.Value, defaultOutputFormat)
		}
		return
	}

	t.Fatal("output flag not found")
}

func TestTabulariseProjectPreset(t *testing.T) {
	t.Parallel()

	rows, headers := tabularise(map[string]any{
		"id":   42,
		"name": "Pine",
		"slug": "pine",
		"owner": map[string]any{
			"id":                7,
			"full_name_display": "Jane Doe",
			"username":          "jdoe",
		},
		"description": "ignored in table preset",
	}, renderOptions{View: "projects"})

	wantHeaders := []string{"ID", "Name", "Slug", "Owner"}
	if !reflect.DeepEqual(headers, wantHeaders) {
		t.Fatalf("headers = %#v, want %#v", headers, wantHeaders)
	}
	if len(rows) != 1 {
		t.Fatalf("row count = %d, want 1", len(rows))
	}

	wantRow := table.Row{"42", "Pine", "pine", "7 / Jane Doe / jdoe"}
	if !reflect.DeepEqual(rows[0], wantRow) {
		t.Fatalf("row = %#v, want %#v", rows[0], wantRow)
	}
}

func TestTabulariseUserStoryPreset(t *testing.T) {
	t.Parallel()

	rows, headers := tabularise([]map[string]any{
		{
			"id":      11,
			"subject": "Improve CLI output",
			"assigned_to_extra_info": map[string]any{
				"id":                3,
				"full_name_display": "Assigned User",
				"username":          "assigned",
			},
			"assigned_users": []any{float64(3), float64(5)},
			"owner_extra_info": map[string]any{
				"id":                1,
				"full_name_display": "Owner User",
				"username":          "owner",
			},
			"status_extra_info": map[string]any{
				"name": "In Progress",
			},
			"project_extra_info": map[string]any{
				"name": "CLI",
				"slug": "cli",
			},
			"description": "ignored in table preset",
		},
	}, renderOptions{View: "user-stories"})

	wantHeaders := []string{"ID", "Subject", "Assigned To", "Assigned Users", "Owner", "Status", "Project", "Slug"}
	if !reflect.DeepEqual(headers, wantHeaders) {
		t.Fatalf("headers = %#v, want %#v", headers, wantHeaders)
	}
	if len(rows) != 1 {
		t.Fatalf("row count = %d, want 1", len(rows))
	}

	wantRow := table.Row{
		"11",
		"Improve CLI output",
		"3 / Assigned User / assigned",
		"3, 5",
		"1 / Owner User / owner",
		"In Progress",
		"CLI",
		"cli",
	}
	if !reflect.DeepEqual(rows[0], wantRow) {
		t.Fatalf("row = %#v, want %#v", rows[0], wantRow)
	}
}

func TestGenericListTableUsesPrioritizedLimitedHeaders(t *testing.T) {
	t.Parallel()

	rows, headers := tabularise([]map[string]any{
		{
			"id":            9,
			"slug":          "pine",
			"subject":       "Default table output",
			"status":        "ready",
			"project":       "Pine",
			"owner":         "Kristof",
			"created_date":  "2026-04-05",
			"modified_date": "2026-04-06",
			"description":   "not selected",
			"extra":         "not selected",
		},
	}, renderOptions{})

	wantHeaders := []string{"ID", "Slug", "Subject", "Status", "Project", "Owner", "Created Date", "Modified Date"}
	if !reflect.DeepEqual(headers, wantHeaders) {
		t.Fatalf("headers = %#v, want %#v", headers, wantHeaders)
	}
	if len(rows) != 1 {
		t.Fatalf("row count = %d, want 1", len(rows))
	}
	if len(headers) != maxGenericTableColumns {
		t.Fatalf("header count = %d, want %d", len(headers), maxGenericTableColumns)
	}
}

func TestRenderTableSummaryIncludesPaginationAndExtra(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := renderTable(&out, PaginationEnvelope{
		Items: []map[string]any{
			{
				"id":               1,
				"name":             "Sprint 1",
				"slug":             "sprint-1",
				"closed":           false,
				"estimated_start":  "2026-04-01",
				"estimated_finish": "2026-04-14",
			},
		},
		Pagination: PaginationView{
			Paginated: true,
			Page:      2,
			PageSize:  5,
			Count:     10,
		},
		Extra: map[string]any{
			"opened_milestones": "1",
			"closed_milestones": "2",
		},
	}, renderOptions{View: "milestones"})
	if err != nil {
		t.Fatalf("renderTable returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Pagination enabled (page 2, count 10, page size 5)") {
		t.Fatalf("expected pagination summary in output: %s", output)
	}
	if !strings.Contains(output, "Opened Milestones: 1") {
		t.Fatalf("expected opened milestones summary in output: %s", output)
	}
	if !strings.Contains(output, "Closed Milestones: 2") {
		t.Fatalf("expected closed milestones summary in output: %s", output)
	}
}
