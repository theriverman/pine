package main

import (
	"os"
	"strconv"
	"testing"
)

type noopSecretStore struct{}

func (noopSecretStore) Get(alias string) (*Secret, error)      { return nil, os.ErrNotExist }
func (noopSecretStore) Set(alias string, secret *Secret) error { return nil }
func (noopSecretStore) Delete(alias string) error              { return nil }
func (noopSecretStore) Supported() bool                        { return false }

func TestIntegrationUserStoryCRUD(t *testing.T) {
	if os.Getenv("PINE_RUN_INTEGRATION") != "1" {
		t.Skip("set PINE_RUN_INTEGRATION=1 to run live Taiga integration tests")
	}

	runtime := integrationRuntime(t)
	session, err := runtime.openSession("local")
	if err != nil {
		t.Fatalf("openSession failed: %v", err)
	}
	defer session.close()

	createPayload := map[string]any{
		"project": session.activeProjectID(),
		"subject": "Integration test US",
	}
	created := map[string]any{}
	if _, err := session.Client.Request.Post(session.Client.MakeURL("userstories"), createPayload, &created); err != nil {
		t.Fatalf("create user story failed: %v", err)
	}

	id := intFromAny(created["id"])
	version := intFromAny(created["version"])
	if id == 0 || version == 0 {
		t.Fatalf("unexpected created user story payload: %#v", created)
	}

	patchPayload := map[string]any{
		"version":     version,
		"description": "Updated by integration test",
	}
	updated := map[string]any{}
	if _, err := session.Client.Request.Patch(session.Client.MakeURL("userstories", strconv.Itoa(id)), patchPayload, &updated); err != nil {
		t.Fatalf("edit user story failed: %v", err)
	}
	if updated["description"] != "Updated by integration test" {
		t.Fatalf("unexpected updated description: %#v", updated["description"])
	}

	if _, err := session.Client.Request.Delete(session.Client.MakeURL("userstories", strconv.Itoa(id))); err != nil {
		t.Fatalf("delete user story failed: %v", err)
	}
}

func TestIntegrationEpicBulkCreate(t *testing.T) {
	if os.Getenv("PINE_RUN_INTEGRATION") != "1" {
		t.Skip("set PINE_RUN_INTEGRATION=1 to run live Taiga integration tests")
	}

	runtime := integrationRuntime(t)
	session, err := runtime.openSession("local")
	if err != nil {
		t.Fatalf("openSession failed: %v", err)
	}
	defer session.close()

	created, err := postEpicBulkCreate(session, session.activeProjectID(), []string{"Integration bulk epic 1", "Integration bulk epic 2"})
	if err != nil {
		t.Fatalf("bulk create epics failed: %v", err)
	}
	if len(created) != 2 {
		t.Fatalf("bulk create returned %d epics, want 2", len(created))
	}

	for _, epic := range created {
		id := intFromAny(epic["id"])
		if id == 0 {
			t.Fatalf("unexpected epic payload: %#v", epic)
		}
		if _, err := session.Client.Request.Delete(session.Client.MakeURL("epics", strconv.Itoa(id))); err != nil {
			t.Fatalf("delete epic %d failed: %v", id, err)
		}
	}
}

func TestIntegrationCloneUserStoryWithSubtasks(t *testing.T) {
	if os.Getenv("PINE_RUN_INTEGRATION") != "1" {
		t.Skip("set PINE_RUN_INTEGRATION=1 to run live Taiga integration tests")
	}

	runtime := integrationRuntime(t)
	session, err := runtime.openSession("local")
	if err != nil {
		t.Fatalf("openSession failed: %v", err)
	}
	defer session.close()

	sourceUS := map[string]any{}
	if _, err := session.Client.Request.Post(session.Client.MakeURL("userstories"), map[string]any{
		"project": session.activeProjectID(),
		"subject": "Clone source user story",
	}, &sourceUS); err != nil {
		t.Fatalf("create source user story failed: %v", err)
	}
	sourceUSID := intFromAny(sourceUS["id"])

	sourceTask := map[string]any{}
	if _, err := session.Client.Request.Post(session.Client.MakeURL("tasks"), map[string]any{
		"project":    session.activeProjectID(),
		"user_story": sourceUSID,
		"subject":    "Clone source task",
	}, &sourceTask); err != nil {
		t.Fatalf("create source task failed: %v", err)
	}

	result, err := cloneUserStory(session, sourceUSID, cloneOptions{
		SubjectPrefix: defaultCloneSubjectPrefix,
		WithSubtasks:  true,
	})
	if err != nil {
		t.Fatalf("cloneUserStory failed: %v", err)
	}

	cloneUSID := intFromAny(result["clone"].(map[string]any)["id"])
	clonedTasks := result["cloned_subtasks"].([]map[string]any)
	if cloneUSID == 0 || len(clonedTasks) != 1 {
		t.Fatalf("unexpected clone result: %#v", result)
	}

	clonedTaskID := intFromAny(clonedTasks[0]["id"])
	if clonedTaskID == 0 {
		t.Fatalf("unexpected cloned task summary: %#v", clonedTasks[0])
	}

	if _, err := session.Client.Request.Delete(session.Client.MakeURL("tasks", strconv.Itoa(clonedTaskID))); err != nil {
		t.Fatalf("delete cloned task failed: %v", err)
	}
	if _, err := session.Client.Request.Delete(session.Client.MakeURL("userstories", strconv.Itoa(cloneUSID))); err != nil {
		t.Fatalf("delete cloned user story failed: %v", err)
	}
	if _, err := session.Client.Request.Delete(session.Client.MakeURL("tasks", strconv.Itoa(intFromAny(sourceTask["id"])))); err != nil {
		t.Fatalf("delete source task failed: %v", err)
	}
	if _, err := session.Client.Request.Delete(session.Client.MakeURL("userstories", strconv.Itoa(sourceUSID))); err != nil {
		t.Fatalf("delete source user story failed: %v", err)
	}
}

func TestIntegrationCloneEpicWithRelatedUserStories(t *testing.T) {
	if os.Getenv("PINE_RUN_INTEGRATION") != "1" {
		t.Skip("set PINE_RUN_INTEGRATION=1 to run live Taiga integration tests")
	}

	runtime := integrationRuntime(t)
	session, err := runtime.openSession("local")
	if err != nil {
		t.Fatalf("openSession failed: %v", err)
	}
	defer session.close()

	sourceEpic := map[string]any{}
	if _, err := session.Client.Request.Post(session.Client.MakeURL("epics"), map[string]any{
		"project": session.activeProjectID(),
		"subject": "Clone source epic",
	}, &sourceEpic); err != nil {
		t.Fatalf("create source epic failed: %v", err)
	}
	sourceEpicID := intFromAny(sourceEpic["id"])

	sourceUS := map[string]any{}
	if _, err := session.Client.Request.Post(session.Client.MakeURL("userstories"), map[string]any{
		"project": session.activeProjectID(),
		"subject": "Related source user story",
	}, &sourceUS); err != nil {
		t.Fatalf("create source user story failed: %v", err)
	}
	sourceUSID := intFromAny(sourceUS["id"])

	if _, err := session.Client.Epic.CreateRelatedUserStory(sourceEpicID, sourceUSID); err != nil {
		t.Fatalf("link related user story failed: %v", err)
	}

	result, err := cloneEpic(session, sourceEpicID, cloneOptions{
		SubjectPrefix:          defaultCloneSubjectPrefix,
		WithRelatedUserStories: true,
	})
	if err != nil {
		t.Fatalf("cloneEpic failed: %v", err)
	}

	cloneEpicID := intFromAny(result["clone"].(map[string]any)["id"])
	clonedUserStories := result["cloned_related_user_stories"].([]map[string]any)
	if cloneEpicID == 0 || len(clonedUserStories) != 1 {
		t.Fatalf("unexpected clone result: %#v", result)
	}

	clonedUSID := intFromAny(clonedUserStories[0]["id"])
	if clonedUSID == 0 {
		t.Fatalf("unexpected cloned user story summary: %#v", clonedUserStories[0])
	}

	if _, err := session.Client.Request.Delete(session.Client.MakeURL("userstories", strconv.Itoa(clonedUSID))); err != nil {
		t.Fatalf("delete cloned user story failed: %v", err)
	}
	if _, err := session.Client.Request.Delete(session.Client.MakeURL("epics", strconv.Itoa(cloneEpicID))); err != nil {
		t.Fatalf("delete cloned epic failed: %v", err)
	}
	if _, err := session.Client.Request.Delete(session.Client.MakeURL("userstories", strconv.Itoa(sourceUSID))); err != nil {
		t.Fatalf("delete source user story failed: %v", err)
	}
	if _, err := session.Client.Request.Delete(session.Client.MakeURL("epics", strconv.Itoa(sourceEpicID))); err != nil {
		t.Fatalf("delete source epic failed: %v", err)
	}
}

func integrationRuntime(t *testing.T) *Runtime {
	t.Helper()

	frontendURL := os.Getenv("PINE_INTEGRATION_FRONTEND")
	if frontendURL == "" {
		frontendURL = "http://localhost:9000"
	}
	username := os.Getenv("PINE_INTEGRATION_USERNAME")
	if username == "" {
		username = "admin"
	}
	password := os.Getenv("PINE_INTEGRATION_PASSWORD")
	if password == "" {
		password = "123123"
	}
	projectID := 1
	if raw := os.Getenv("PINE_INTEGRATION_PROJECT_ID"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			t.Fatalf("invalid PINE_INTEGRATION_PROJECT_ID: %v", err)
		}
		projectID = parsed
	}
	projectSlug := os.Getenv("PINE_INTEGRATION_PROJECT_SLUG")
	if projectSlug == "" {
		projectSlug = "demo"
	}

	apiURL, baseURL, apiVersion, err := discoverInstance(frontendURL)
	if err != nil {
		t.Fatalf("discoverInstance failed: %v", err)
	}

	t.Setenv(envPassword, password)

	return &Runtime{
		Config: &Config{
			CurrentInstance: "local",
			Instances: map[string]*Instance{
				"local": {
					Alias:       "local",
					FrontendURL: frontendURL,
					APIURL:      apiURL,
					BaseURL:     baseURL,
					APIVersion:  apiVersion,
					AuthType:    "normal",
					Username:    username,
					DefaultProject: &SavedProject{
						ID:   projectID,
						Slug: projectSlug,
						Name: projectSlug,
					},
				},
			},
		},
		Secrets: noopSecretStore{},
	}
}

func intFromAny(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}
