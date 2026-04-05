# AGENTS.md

Guidance for coding agents working in this repository.

## Project Summary

- `pine` is a single-binary Go CLI for Taiga.
- Primary libraries:
  - `github.com/urfave/cli/v3` for command wiring
  - `github.com/theriverman/taigo/v2` for Taiga API access
  - `github.com/jedib0t/go-pretty/v6` for table output
  - `gopkg.in/yaml.v3` for config serialization
- Toolchain is pinned in `go.mod`:
  - `go 1.25.0`
  - `toolchain go1.25.8`

## Repository Map

- `main.go`: thin entrypoint; builds the app and prints errors to stderr.
- `app.go`: root command, global flags, `ctx`/`me` commands, and top-level command assembly.
- `resources.go`: data-driven CRUD command specs, field mappings, query/body flag definitions, and generic resource handlers.
- `clone.go`: clone flows for epics, user stories, and tasks.
- `output.go`: output selection and table rendering. Default output is `table`.
- `config.go`: runtime loading, config persistence, saved instance/project management.
- `client.go`: Taiga discovery and authenticated session creation.
- `secret_common.go`, `secret_supported.go`, `secret_unsupported.go`: secret storage integration.
- `internal/taigainstance/`: frontend discovery helpers for Taiga instance metadata.
- `integration_test.go`: opt-in live tests against a real Taiga stack.
- `scripts/smoke-cli.sh`: smoke checks for `--help`, `--version`, and shell completion.
- `README.md`: end-user install/config/usage docs.
- `CONTRIBUTE.md`: contributor build/test/release workflow.

## Architecture Notes

- Keep `main.go` thin. Command handlers should return errors; let the entrypoint print and exit.
- Prefer extending the existing data-driven resource model in `resources.go` over adding one-off command implementations.
- Match the existing flag convention:
  - CLI flags use kebab-case.
  - API fields usually map to snake_case aliases through `fieldSpec`.
- Project resolution already has a clear precedence path:
  - explicit CLI flags
  - environment variables
  - saved config / secret store
- Output behavior is intentional:
  - JSON and YAML should stay close to raw payloads.
  - Table output is curated and view-specific.
  - If you change user-visible table output, update `output.go` and `output_test.go` together.

## Editing Guidance

- Check `git status` before editing. Assume the worktree may already contain unrelated user changes.
- Do not revert or overwrite changes you did not make unless the user explicitly asks for that.
- Keep patches focused. This repo favors small, direct changes over broad refactors.
- When patching code, update the relevant documentation in the same change. Do not leave code and docs out of sync.
- Follow the existing style:
  - simple functions
  - standard library first
  - table-driven tests where useful
  - `t.Parallel()` in unit tests when safe
- When adding a new resource behavior, look first at `resourceSpecs()`, `resourceCommand(...)`, and related tests before inventing new patterns.
- When changing config/session behavior, verify both saved-config and environment-variable paths.
- When changing release or secret-storage behavior, preserve the platform split between darwin and linux/windows builds.

## Validation

Run the smallest meaningful set of checks for the change, then state what you ran.

- Whenever an AI agent works on the project, review `go.mod` dependencies for newer available versions and call out any upgrades made or intentionally deferred.

Common local loop:

```bash
gofmt -w .
go test ./...
go vet ./...
make smoke
```

CI-relevant checks:

```bash
go test -count=1 -shuffle=on ./...
go test -count=1 -shuffle=on -race -covermode=atomic -coverprofile=coverage.out ./...
go mod tidy
```

Additional CI gates to keep in mind:

- `gofmt` must leave no diff.
- `go mod tidy` must not change `go.mod` or `go.sum`.
- `golangci-lint` uses `.golangci.yml`.
- typo checks use `.typos.toml`.
- release builds are validated via `Makefile` targets and `.goreleaser.yml`.

## Test Placement

- Put unit tests next to the code they cover.
- Output-format and table-preset behavior belongs in `output_test.go`.
- Clone behavior belongs in `clone_test.go`.
- Shared helpers and parsing behavior usually belong in `util_test.go` or the nearest focused `*_test.go`.
- Live Taiga coverage belongs in `integration_test.go` and must remain opt-in.

## Environment And Manual Testing

- Use `PINE_CONFIG_DIR` for local manual testing so you do not touch real user config.
- Supported runtime environment variables:
  - `PINE_INSTANCE`
  - `PINE_AUTH_TYPE`
  - `PINE_USERNAME`
  - `PINE_PASSWORD`
  - `PINE_TOKEN`
  - `PINE_CONFIG_DIR`
- Live integration tests require `PINE_RUN_INTEGRATION=1`.
- Integration defaults target:
  - frontend: `http://localhost:9000`
  - username: `admin`
  - password: `123123`
  - project slug: `demo`
  - project ID: `1`

## Build And Release Constraints

- `make build` writes the host binary to `dist/`.
- `make smoke` builds the host binary and runs `scripts/smoke-cli.sh`.
- `make darwin` must run on macOS because release darwin binaries keep Keychain support through `CGO_ENABLED=1`.
- Linux and Windows release builds stay on the pure-Go cross-build path with `CGO_ENABLED=0`.
- The project should aim to support the latest released versions of Go. When Go-version-related changes are made, keep `go.mod`, CI, build scripts, and release configuration aligned.
- If you change build metadata, release artifacts, or target names, keep `Makefile`, `.github/workflows/ci.yml`, and `.goreleaser.yml` aligned.

## Commit Messages

- Whenever creating a new commit, follow the Conventional Commits 1.0.0 specification: [conventionalcommits.org](https://www.conventionalcommits.org/en/v1.0.0/).
- The full specification is also available as raw Markdown at [conventionalcommits.org source](https://raw.githubusercontent.com/conventional-commits/conventionalcommits.org/refs/heads/master/content/v1.0.0/index.md).
- Commit subjects must use the required Conventional Commits structure: `<type>[optional scope][optional !]: <description>`.
- Use `feat` for new features, `fix` for bug fixes, and mark breaking changes with `!` and/or a `BREAKING CHANGE:` footer where appropriate.
- Do not create ad-hoc commit-message formats when committing from this repository.

## Documentation Expectations

- Documentation and in-line comments must follow British English spelling and usage.
- Update `README.md` when user-facing CLI behavior, flags, examples, or config semantics change.
- Update `CONTRIBUTE.md` when build, test, CI, or release workflow changes.
- Do not let docs drift from the actual command names or environment variables.
