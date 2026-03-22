APP := pine
DIST_DIR := dist
RELEASE_TARGETS := darwin linux windows
RELEASE_ARCHES := amd64 arm64
VERSION ?= $(shell sh -c 'tag=$$(git tag --points-at HEAD --sort=-v:refname | head -n 1); if [ -z "$$tag" ]; then tag=$$(git tag --sort=-v:refname | head -n 1); fi; if [ -n "$$tag" ]; then printf "%s" "$$tag"; else printf "dev"; fi')
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
GO_VERSION ?= $(shell go env GOVERSION 2>/dev/null || go version | awk '{print $$3}')
HOST_GOOS ?= $(shell go env GOHOSTOS 2>/dev/null || printf "unknown")
HOST_GOARCH ?= $(shell go env GOHOSTARCH 2>/dev/null || printf "unknown")
BUILD_EXT := $(if $(filter windows,$(HOST_GOOS)),.exe,)
LDFLAGS := -s -w -X 'main.buildVersion=$(VERSION)' -X 'main.buildCommit=$(COMMIT)' -X 'main.buildGoVersion=$(GO_VERSION)'
MAKEFILE_PATH := $(firstword $(MAKEFILE_LIST))
HOST_BINARY := $(DIST_DIR)/$(APP)-$(HOST_GOOS)-$(HOST_GOARCH)$(BUILD_EXT)

.DEFAULT_GOAL := help

.PHONY: help build smoke release-builds darwin linux windows clean

define build_binary
GOOS=$(1) GOARCH=$(2) go build -ldflags "$(LDFLAGS)" -o "$(DIST_DIR)/$(APP)-$(1)-$(2)$(if $(filter windows,$(1)),.exe,)" .
endef

define build_release_target
mkdir -p $(DIST_DIR)
	@set -e; \
	for arch in $(RELEASE_ARCHES); do \
		printf "building %s/%s\n" "$(1)" "$$arch"; \
		ext=""; \
		if [ "$(1)" = "windows" ]; then ext=".exe"; fi; \
		GOOS=$(1) GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o "$(DIST_DIR)/$(APP)-$(1)-$$arch$$ext" .; \
	done
endef

help: ## Show available build targets and derived build metadata
	@printf "Available targets for %s\n\n" "$(APP)"
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_.-]+:.*## / {printf "  %-10s %s\n", $$1, $$2}' $(MAKEFILE_PATH)
	@printf "\nBuild metadata\n"
	@printf "  version    %s\n" "$(VERSION)"
	@printf "  commit     %s\n" "$(COMMIT)"
	@printf "  go         %s\n" "$(GO_VERSION)"
	@printf "  host       %s/%s\n" "$(HOST_GOOS)" "$(HOST_GOARCH)"

build: ## Build a binary for the current host OS and architecture into dist/
	mkdir -p $(DIST_DIR)
	$(call build_binary,$(HOST_GOOS),$(HOST_GOARCH))

smoke: build ## Build and smoke test the current host binary
	./scripts/smoke-cli.sh "$(HOST_BINARY)"

release-builds: $(RELEASE_TARGETS) ## Build all cross-platform release binaries into dist/

darwin: ## Build macOS binaries for amd64 and arm64 into dist/
	$(call build_release_target,darwin)

linux: ## Build Linux binaries for amd64 and arm64 into dist/
	$(call build_release_target,linux)

windows: ## Build Windows binaries for amd64 and arm64 into dist/
	$(call build_release_target,windows)

clean: ## Remove generated release binaries
	rm -rf $(DIST_DIR)
