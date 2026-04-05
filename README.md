<p align="center">
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="assets/banner-dark.png">
  <source media="(prefers-color-scheme: light)" srcset="assets/banner-light.png">
  <img alt="Pine Banner." src="assets/banner-dark.png">
</picture>
</p>

[![CI](https://github.com/theriverman/pine/actions/workflows/ci.yml/badge.svg)](https://github.com/theriverman/pine/actions/workflows/ci.yml)

# Pine

`pine` is a Go CLI for Taiga, built on [`github.com/theriverman/taigo`](https://github.com/theriverman/taigo) and [`urfave/cli/v3`](https://cli.urfave.org/).

It is aimed at people who want a local CLI for common Taiga workflows with saved context, structured output, and shell completion.

## Current scope

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

## Install

Download a prebuilt binary from [GitHub Releases](https://github.com/theriverman/pine/releases), or build from source if you prefer.

To build the current host binary locally:

```bash
make build
```

The binary is written to `dist/`.

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
- Linux: Secret Service (`gnome-keyring`) or KWallet when a desktop session exposes them; otherwise use environment variables

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

## Quick start

Add a Taiga instance interactively:

```bash
pine ctx instance add
```

`pine` fetches `<frontend>/conf.json`, requires an `api` key, and derives the Taiga client base URL and API version from it.

List saved instances:

```bash
pine ctx instance list
```

Select the active instance:

```bash
pine ctx instance use local
```

Add a project from the active instance:

```bash
pine ctx project add
```

Select the default project:

```bash
pine ctx project use demo
```

Show the current context:

```bash
pine ctx show
```

Remove saved context when needed:

```bash
pine ctx instance remove local
pine ctx project remove demo
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

Table is the default. Use `--output json` when you want the full raw payload.

Tables are rendered with `github.com/jedib0t/go-pretty` and prefer a curated subset of the most useful fields for common resource types such as projects, epics, user stories, tasks, and milestones.

`pine ctx show` uses a dedicated table view that summarises the active instance, authentication settings, default project, and saved projects. Project and task tables also include related IDs alongside names where that makes cross-referencing easier.

## Common command patterns

List resources:

```bash
pine epics list
pine epics list --project=1 --status-is-closed=false
pine projects list --page=1 --page-size=10
pine projects list --mine
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
- `pine projects list --mine` filters projects to the current authenticated user and cannot be combined with `--member` or `--members`
- create and edit accept `--from-json`; CLI flags override values from the JSON file
- clone commands accept `--subject` or `--subject-prefix`; by default the cloned subject is prefixed with `Copy of `
- `pine us clone --with-subtasks` clones the user story and recreates its related tasks under the cloned user story
- `pine epics clone --with-related-user-stories` clones the epic, clones each related user story, and links the cloned user stories to the cloned epic
- `pine tasks clone --user-story=<id>` clones a task directly into a different user story

## Scope and limitations

- this milestone intentionally focuses on curated endpoints rather than the entire Taiga API surface
- `resolver`, attachments, import/export, and other admin-heavy or specialised endpoints are not included yet
- Linux native secret storage requires a running Secret Service or KWallet session; otherwise provide credentials through environment variables
- `edit` uses merged-update behaviour; explicit field clearing is handled through `--clear`

## Contributing

Contributor and development workflow documentation lives in [`CONTRIBUTE.md`](./CONTRIBUTE.md).
