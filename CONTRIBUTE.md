# Contributing To Pine

This document is for people developing, testing, or releasing `pine`.

For end-user setup and CLI usage, see [`README.md`](./README.md).

## Toolchain

- Go version is defined in [`go.mod`](./go.mod)
- the repository currently pins `go 1.25.0` with `toolchain go1.25.9`
- `make` is used for local and release builds
- Docker is required if you want to run the live integration suite locally

Install module dependencies in the usual Go way:

```bash
go mod download
```

## Repository layout

- `main.go`, `app.go`: CLI entrypoint and command wiring
- `resources.go`, `clone.go`, `output.go`: resource actions, clone helpers, output rendering
- `config.go`, `client.go`, `secret_*.go`: configuration loading, Taiga client setup, secret storage
- `scripts/smoke-cli.sh`: smoke test for a built binary
- `scripts/ci/prepare_integration_env`: prepares CI integration-test state
- `.github/workflows/ci.yml`: CI, smoke, integration, and release workflow

## Local build

Build the current host binary:

```bash
make build
```

Show available build targets and derived build metadata:

```bash
make
```

Build all release targets locally:

```bash
make darwin
make linux
make windows
```

Artifacts are written to `dist/`.

`make darwin` must run on a macOS host so the release binaries keep native Keychain support through `cgo`.

`make linux` and `make windows` remain pure-Go cross-builds with `CGO_ENABLED=0`.

## Local checks

Run the core local checks before opening a PR:

```bash
gofmt -w .
go test ./...
go vet ./...
make smoke
```

CI also verifies:

- `go mod tidy`
- `deadcode`
- `govulncheck`
- `typos`
- `golangci-lint`
- race detection and coverage on Linux
- native smoke tests on Linux, macOS, and Windows
- release builds for every target, with macOS built natively on macOS and Linux/Windows built via generic cross-compilation
- Markdown-only branch pushes and PR updates are skipped; semver tags pointing at Markdown-only commits are also gated off before the expensive CI and release jobs start

If you want to mirror CI more closely, the commands are defined in [`.github/workflows/ci.yml`](./.github/workflows/ci.yml), plus [`.golangci.yml`](./.golangci.yml), [`.typos.toml`](./.typos.toml), and [`.goreleaser.yml`](./.goreleaser.yml).

## Smoke testing

`make smoke` builds the host binary and validates:

- `pine --help`
- `pine --version`
- `pine completion bash`

The smoke script lives in [`scripts/smoke-cli.sh`](./scripts/smoke-cli.sh).

## Live integration tests

The repository includes live integration tests against a real Taiga stack in [`integration_test.go`](./integration_test.go).

CI runs them by:

1. checking out a pinned `taigaio/taiga-docker` revision
2. launching the stack locally with Docker Compose
3. seeding the admin account
4. preparing a test project with `scripts/ci/prepare_integration_env`
5. running `go test -count=1 -run '^TestIntegration' ./...`

To run them locally, you need a reachable Taiga frontend and must set:

- `PINE_RUN_INTEGRATION=1`

Optional integration overrides:

- `PINE_INTEGRATION_FRONTEND`
- `PINE_INTEGRATION_USERNAME`
- `PINE_INTEGRATION_PASSWORD`
- `PINE_INTEGRATION_PROJECT_ID`
- `PINE_INTEGRATION_PROJECT_SLUG`

Default local assumptions in the tests are:

- frontend: `http://localhost:9000`
- username: `admin`
- password: `123123`
- project slug: `demo`
- project ID: `1`

Example:

```bash
PINE_RUN_INTEGRATION=1 go test -count=1 -run '^TestIntegration' ./...
```

## Release flow

Local release artifacts are produced through the `Makefile`, while tagged GitHub releases are published by GoReleaser.

Tagged release publishing runs on a macOS runner so the darwin artifacts are built with `CGO_ENABLED=1`; Linux and Windows artifacts keep the existing `CGO_ENABLED=0` cross-build path.

Release publishing in CI happens only for semver tags matching `v*` after all quality, test, smoke, release-target, and integration jobs pass.

The archive configuration lives in [`.goreleaser.yml`](./.goreleaser.yml).

## Documentation split

Documentation in this repository is intentionally split by audience:

- [`README.md`](./README.md): what `pine` is, how to install it, how to configure it, and how to use it
- [`CONTRIBUTE.md`](./CONTRIBUTE.md): how to build, test, validate, and release changes
