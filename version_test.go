package main

import "testing"

func TestCurrentVersionInfoUsesRuntimeGoVersionWhenBuildValueMissing(t *testing.T) {
	originalVersion := buildVersion
	originalCommit := buildCommit
	originalGoVersion := buildGoVersion
	t.Cleanup(func() {
		buildVersion = originalVersion
		buildCommit = originalCommit
		buildGoVersion = originalGoVersion
	})

	buildVersion = "v1.2.3"
	buildCommit = "abcdef0"
	buildGoVersion = ""

	info := currentVersionInfo()
	if info.Name != appName {
		t.Fatalf("expected app name %q, got %q", appName, info.Name)
	}
	if info.Version != "v1.2.3" {
		t.Fatalf("expected build version to be used, got %q", info.Version)
	}
	if info.Commit != "abcdef0" {
		t.Fatalf("expected build commit to be used, got %q", info.Commit)
	}
	if info.GoVersion == "" {
		t.Fatal("expected a Go version")
	}
}

func TestCurrentVersionInfoUsesInjectedGoVersion(t *testing.T) {
	originalGoVersion := buildGoVersion
	t.Cleanup(func() {
		buildGoVersion = originalGoVersion
	})

	buildGoVersion = "go1.25.2"

	info := currentVersionInfo()
	if info.GoVersion != "go1.25.2" {
		t.Fatalf("expected injected Go version, got %q", info.GoVersion)
	}
}
