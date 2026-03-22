package main

import (
	"testing"

	"pine/internal/taigainstance"
)

func TestSplitAPIURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		apiURL     string
		wantBase   string
		wantAPIver string
	}{
		{
			name:       "root api path",
			apiURL:     "http://localhost:9000/api/v1/",
			wantBase:   "http://localhost:9000",
			wantAPIver: "v1",
		},
		{
			name:       "prefixed api path",
			apiURL:     "https://example.com/taiga/api/v2",
			wantBase:   "https://example.com/taiga",
			wantAPIver: "v2",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			baseURL, apiVersion, err := taigainstance.SplitAPIURL(test.apiURL)
			if err != nil {
				t.Fatalf("splitAPIURL returned error: %v", err)
			}
			if baseURL != test.wantBase {
				t.Fatalf("baseURL = %q, want %q", baseURL, test.wantBase)
			}
			if apiVersion != test.wantAPIver {
				t.Fatalf("apiVersion = %q, want %q", apiVersion, test.wantAPIver)
			}
		})
	}
}

func TestDeriveChangedFields(t *testing.T) {
	t.Parallel()

	current := map[string]any{
		"subject":          "Before",
		"description":      "Old",
		"team_requirement": false,
		"watchers":         []any{float64(1), float64(2)},
	}
	desired := map[string]any{
		"subject":          "Before",
		"description":      "",
		"team_requirement": true,
		"watchers":         []int{},
	}

	changed := deriveChangedFields(current, desired)
	if got, ok := changed["description"].(string); !ok || got != "" {
		t.Fatalf("description not tracked as a cleared field: %#v", changed["description"])
	}
	if got, ok := changed["team_requirement"].(bool); !ok || !got {
		t.Fatalf("team_requirement not tracked as a changed bool: %#v", changed["team_requirement"])
	}
	if got, ok := changed["watchers"].([]int); !ok || len(got) != 0 {
		t.Fatalf("watchers not tracked as a cleared slice: %#v", changed["watchers"])
	}
}
