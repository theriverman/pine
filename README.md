# Pine

`pine` is a Go CLI for Taiga, built on [`github.com/theriverman/taigo/v2`](https://github.com/theriverman/taigo/tree/v2) and [`urfave/cli/v3`](https://cli.urfave.org/).

The current scope is the first milestone:

- saved Taiga instance management
- saved project management per instance
- persistent default project selection
- shell completion for `bash`, `zsh`, `fish`, and `powershell`
- CRUD for the curated resource set:
  - `projects`
  - `epics`
  - `user-stories` / `us`
  - `tasks`
  - `issues`
  - `milestones`
  - `wiki`
  - `users`
  - `me`
  - `search`
- clone helpers for:
  - `epics`
  - `user-stories` / `us`
  - `tasks`

## Build

`pine` depends on `github.com/theriverman/taigo/v2` as a normal Go module dependency.

```bash
go build ./...
```

Cross-platform release builds are handled by the included `Makefile`:

```bash
make
make build
make darwin
make linux
make windows
```

Running `make` with no target prints the available targets and the derived build metadata.
`make build` creates a binary for the current host OS and architecture.
The generated binaries are written to `dist/`.
Tagged GitHub releases are published from semver tags by GoReleaser after the full CI workflow passes.

During each build, `make` collects:

- the latest git tag as the app version
- the latest commit hash
- the Go version used for the build

These values are injected into the binary and exposed through `pine --version`.
If the repository has no git tags yet, the version falls back to `dev`.

## Version information

Print build metadata with:

```bash
pine --version
```

The output includes:

- app name
- app version
- app commit
- build Go version

## Configuration and secrets

`pine` stores configuration in the platform config directory:

- Linux: `~/.config/pine/config.yaml`
- macOS: `~/Library/Application Support/pine/config.yaml`
- Windows: `%AppData%\pine\config.yaml`

The YAML file stores metadata only:

- saved instance aliases
- frontend URL
- discovered API URL
- auth type
- username
- saved projects
- default project

Secrets are stored separately:

- macOS: native Keychain
- Windows: native Credential Manager
- Linux: secret persistence is not supported; use environment variables instead

For development and tests, `PINE_CONFIG_DIR` can override the config directory.

## Environment variables

Runtime precedence is:

1. explicit CLI flags
2. environment variables
3. native secret storage
4. saved config

Supported environment variables:

- `PINE_INSTANCE`
- `PINE_AUTH_TYPE`
- `PINE_USERNAME`
- `PINE_PASSWORD`
- `PINE_TOKEN`
- `PINE_CONFIG_DIR`

## Instance workflow

Add an instance interactively:

```bash
pine ctx instance add
```

`pine` fetches `<frontend>/conf.json`, requires an `api` key, and derives the Taigo client `BaseURL` and API version from it.

List instances:

```bash
pine ctx instance list
```

Switch the active instance:

```bash
pine ctx instance use
pine ctx instance use local
```

Remove an instance:

```bash
pine ctx instance remove
pine ctx instance remove local
```

## Project workflow

Add a project from the active Taiga instance:

```bash
pine ctx project add
```

List saved projects:

```bash
pine ctx project list
```

Select the default project:

```bash
pine ctx project use
pine ctx project use demo
```

Remove a saved project:

```bash
pine ctx project remove
```

Show the current context:

```bash
pine ctx show
```

## Shell completion

Generate completion scripts with the built-in `completion` command:

```bash
pine completion bash
pine completion zsh
pine completion fish
pine completion powershell
```

Examples:

```bash
source <(pine completion bash)
source <(pine completion zsh)
pine completion fish > ~/.config/fish/completions/pine.fish
```

## Output formats

`pine` supports:

- `--output json`
- `--output yaml`
- `--output table`

JSON is the default. Tables are rendered with `github.com/jedib0t/go-pretty`.

For deeply nested resources, table output is best-effort and may still be wide.

## Common command patterns

List resources:

```bash
pine epics list
pine epics list --project=1 --status-is-closed=false
pine projects list --page=1 --page-size=10
```

Fetch a resource:

```bash
pine epics get 7
pine projects get demo
pine wiki get home
```

Create a resource:

```bash
pine us create --subject="My first user story"
pine issues create --subject="Bug report" --priority=2 --severity=3
pine milestones create --name="Sprint 1" --estimated-start=2026-03-13 --estimated-finish=2026-03-27
```

Edit a resource:

```bash
pine us edit 42 --description="Updated description"
pine us edit 42 --from-json=/path/to/us.json --subject="CLI flags still win"
pine us edit 42 --clear description
```

Clone a resource:

```bash
pine epics clone 7
pine epics clone 7 --with-related-user-stories
pine us clone 42
pine us clone 42 --with-subtasks
pine tasks clone 99
pine tasks clone 99 --user-story=42
```

Delete a resource:

```bash
pine us delete 42
pine us delete 42 --yes
```

Bulk-create epics from JSON:

```bash
pine epics bulk-creation --json=/path/to/epics.json
```

`bulk-creation` accepts a JSON array of strings or objects with a `subject` key.

## Flag conventions

- command names use kebab case
- query and payload flags expose kebab-case names
- where useful, Taiga-style aliases are also accepted, for example `--status__is_closed`
- project-scoped create commands fall back to the saved default project when `--project` is omitted
- create and edit accept `--from-json`; CLI flags override values from the JSON file
- clone commands accept `--subject` or `--subject-prefix`; by default the cloned subject is prefixed with `Copy of `
- `pine us clone --with-subtasks` clones the user story and recreates its related tasks under the cloned user story
- `pine epics clone --with-related-user-stories` clones the epic, clones each related user story, and links the cloned user stories to the cloned epic
- `pine tasks clone --user-story=<id>` clones a task directly into a different user story

## Current scope and limitations

- This milestone intentionally focuses on curated endpoints rather than the entire Taiga API surface.
- `resolver`, attachments, import/export, and other admin-heavy or specialised endpoints are not included yet.
- Linux users must provide credentials through environment variables instead of native secret storage.
- `edit` is user-friendly merged update behaviour; explicit field clearing is handled through `--clear`.

## Testing

Run the unit suite:

```bash
go test ./...
```

Run the live integration suite against a local Taiga instance:

```bash
PINE_RUN_INTEGRATION=1 go test ./...
```

GitHub Actions boots a pinned [`taigaio/taiga-docker`](https://github.com/taigaio/taiga-docker) sandbox on `http://localhost:9000`, creates an admin user with Django's non-interactive `createsuperuser` flow, and creates an ephemeral integration project before running the same live suite.

Optional integration environment variables:

- `PINE_INTEGRATION_FRONTEND`
- `PINE_INTEGRATION_USERNAME`
- `PINE_INTEGRATION_PASSWORD`
- `PINE_INTEGRATION_PROJECT_ID`
- `PINE_INTEGRATION_PROJECT_SLUG`

The integration tests default to:

- frontend: `http://localhost:9000`
- username: `admin`
- password: `123123`
- project: `demo`

## Notes

The CLI has been exercised against a live Docker Taiga sandbox at `http://localhost:9000`, including:

- instance discovery through `conf.json`
- saved instance and project flows
- paginated list commands
- `us` create/edit/clear/delete
- `epics`, `us`, and `tasks` clone flows
- `epics` bulk creation
- completion generation
