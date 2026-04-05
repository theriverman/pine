package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	taigo "github.com/theriverman/taigo/v2"
	cli "github.com/urfave/cli/v3"
)

func TestProjectListCommandIncludesMineFlag(t *testing.T) {
	t.Parallel()

	spec := mustResourceSpec(t, "projects")
	listCmd := resourceCommand(spec).Command("list")
	if listCmd == nil {
		t.Fatal("projects list command not found")
	}

	for _, flag := range listCmd.Flags {
		boolFlag, ok := flag.(*cli.BoolFlag)
		if !ok {
			continue
		}
		if boolFlag.Name == "mine" {
			return
		}
	}

	t.Fatal("projects list command missing --mine flag")
}

func TestPrepareProjectListQueryMineUsesCurrentUserMembership(t *testing.T) {
	t.Parallel()

	spec := mustResourceSpec(t, "projects")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users/me" {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"id":       42,
			"username": "theriverman67",
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	session := newTestSession(t, server)
	values, err := runProjectListQueryHook(context.Background(), spec, session, []string{"list", "--mine"})
	if err != nil {
		t.Fatalf("runProjectListQueryHook returned error: %v", err)
	}
	if got := values.Get("member"); got != "42" {
		t.Fatalf("member filter = %q, want %q", got, "42")
	}
	if got := values.Get("members"); got != "" {
		t.Fatalf("members filter = %q, want empty", got)
	}
}

func TestPrepareProjectListQueryMineRejectsExplicitMemberFilters(t *testing.T) {
	t.Parallel()

	spec := mustResourceSpec(t, "projects")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request path: %s", r.URL.Path)
	}))
	defer server.Close()

	session := newTestSession(t, server)
	_, err := runProjectListQueryHook(context.Background(), spec, session, []string{"list", "--mine", "--member", "7"})
	if err == nil {
		t.Fatal("expected an error for conflicting member filters")
	}
	if got := err.Error(); got != "--mine cannot be combined with --member or --members" {
		t.Fatalf("error = %q", got)
	}
}

func runProjectListQueryHook(ctx context.Context, spec resourceSpec, session *Session, args []string) (url.Values, error) {
	values := url.Values{}
	cmd := resourceCommand(spec).Command("list")
	if cmd == nil {
		return nil, http.ErrMissingFile
	}
	cmd.Action = func(_ context.Context, cmd *cli.Command) error {
		queryValues, err := collectQueryValues(cmd, spec.QueryFields)
		if err != nil {
			return err
		}
		if spec.PrepareListQuery != nil {
			if err := spec.PrepareListQuery(session, cmd, queryValues); err != nil {
				return err
			}
		}
		values = queryValues
		return nil
	}

	if err := cmd.Run(ctx, args); err != nil {
		return nil, err
	}
	return values, nil
}

func newTestSession(t *testing.T, server *httptest.Server) *Session {
	t.Helper()

	client := &taigo.Client{
		BaseURL:    server.URL,
		APIversion: "v1",
		HTTPClient: server.Client(),
	}
	if err := client.Initialise(); err != nil {
		t.Fatalf("Initialise failed: %v", err)
	}
	client.SetAuthTokens(taigo.TokenBearer, "test-token", "")
	t.Cleanup(client.Close)

	return &Session{Client: client}
}

func mustResourceSpec(t *testing.T, name string) resourceSpec {
	t.Helper()

	for _, spec := range resourceSpecs() {
		if spec.Name == name {
			return spec
		}
	}

	t.Fatalf("resource spec %q not found", name)
	return resourceSpec{}
}
