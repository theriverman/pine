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
				"id":   29,
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
		"29 / CLI",
		"cli",
	}
	if !reflect.DeepEqual(rows[0], wantRow) {
		t.Fatalf("row = %#v, want %#v", rows[0], wantRow)
	}
}

func TestTabulariseTaskPresetIncludesIDsInRelatedColumns(t *testing.T) {
	t.Parallel()

	rows, headers := tabularise([]map[string]any{
		{
			"id":      8868813,
			"subject": "Holdanya",
			"owner_extra_info": map[string]any{
				"id":                836830,
				"full_name_display": "Kristof",
				"username":          "theriverman67",
			},
			"status_extra_info": map[string]any{
				"name": "New",
			},
			"user_story_extra_info": map[string]any{
				"id":      112233,
				"subject": "Vashegyek",
			},
			"project_extra_info": map[string]any{
				"id":   1711749,
				"name": "Thy Catafalque",
				"slug": "theriverman67-x",
			},
		},
	}, renderOptions{View: "tasks"})

	wantHeaders := []string{"ID", "Subject", "Assigned To", "Owner", "Status", "User Story", "Project", "Slug"}
	if !reflect.DeepEqual(headers, wantHeaders) {
		t.Fatalf("headers = %#v, want %#v", headers, wantHeaders)
	}
	if len(rows) != 1 {
		t.Fatalf("row count = %d, want 1", len(rows))
	}

	wantRow := table.Row{
		"8868813",
		"Holdanya",
		"",
		"836830 / Kristof / theriverman67",
		"New",
		"112233 / Vashegyek",
		"1711749 / Thy Catafalque",
		"theriverman67-x",
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

func TestRenderTableContextViewRendersReadableSummary(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := renderTable(&out, ContextView{
		CurrentInstance: "tree",
		Instance: &Instance{
			Alias:       "tree",
			FrontendURL: "https://tree.taiga.io",
			APIURL:      "https://api.taiga.io/api/v1",
			AuthType:    "normal",
			Username:    "theriverman67",
			DefaultProject: &SavedProject{
				ID:   1711749,
				Slug: "theriverman67-x",
				Name: "X",
			},
			SavedProjects: []*SavedProject{
				{
					ID:   1711749,
					Slug: "theriverman67-x",
					Name: "X",
				},
				{
					ID:   1711750,
					Slug: "theriverman67-y",
					Name: "Y",
				},
			},
		},
		DefaultProject: &SavedProject{
			ID:   1711749,
			Slug: "theriverman67-x",
			Name: "X",
		},
	}, renderOptions{View: "context"})
	if err != nil {
		t.Fatalf("renderTable returned error: %v", err)
	}

	output := out.String()
	for _, snippet := range []string{
		"Current context",
		"Instance:         tree",
		"Frontend URL:     https://tree.taiga.io",
		"API URL:          https://api.taiga.io/api/v1",
		"Auth Type:        normal",
		"Username:         theriverman67",
		"Default Project:  X (theriverman67-x, ID 1711749)",
		"Saved projects",
		"| yes     | 1711749 | theriverman67-x | X    |",
		"|         | 1711750 | theriverman67-y | Y    |",
	} {
		if !strings.Contains(output, snippet) {
			t.Fatalf("expected %q in output:\n%s", snippet, output)
		}
	}
	if strings.Contains(output, "| FIELD ") || strings.Contains(output, "| VALUE ") {
		t.Fatalf("expected context view to avoid generic field/value table:\n%s", output)
	}
}

func TestRenderTableContextViewWithoutSelection(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := renderTable(&out, ContextView{}, renderOptions{View: "context"})
	if err != nil {
		t.Fatalf("renderTable returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Instance:  none selected") {
		t.Fatalf("expected empty selection message in output:\n%s", output)
	}
	if strings.Contains(output, "Saved projects") {
		t.Fatalf("did not expect saved projects section in output:\n%s", output)
	}
}
