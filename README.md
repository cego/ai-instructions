# ai-instructions

A package-manager-style CLI tool for managing company-wide AI coding instruction files (`.md`) across project repositories. Think **npm/Composer but for AI agent instructions**.

It syncs instruction stacks from a central registry into project repos and manages `CLAUDE.md`, `AGENTS.md`, and `.cursorrules` files automatically.

## Architecture

```
┌──────────────────────────────────────────────────┐
│              REGISTRY (source of truth)           │
│                                                   │
│  Private GitLab repo or webserver                 │
│  Serves: registry.json + stack folders with .md   │
└──────────────────────┬───────────────────────────┘
                       │ HTTPS
                       ▼
┌──────────────────────────────────────────────────┐
│           ai-instructions CLI (Go binary)         │
│                                                   │
│  Developer: init, add, remove, sync              │
│  CI:        verify (exit 0 or exit 1)            │
└──────────────────────┬───────────────────────────┘
                       │ reads/writes
                       ▼
┌──────────────────────────────────────────────────┐
│                 PROJECT REPO                      │
│                                                   │
│  ai-instructions-settings.json  ← lockfile        │
│  ai-instructions/               ← .md files       │
│  CLAUDE.md                      ← managed block   │
│  AGENTS.md                      ← managed block   │
│  .cursorrules                   ← managed block   │
└──────────────────────────────────────────────────┘
```

## Installation

### From source

```bash
make build
# Binary is at ./bin/ai-instructions
```

### Docker

```bash
docker build -t ai-instructions .
```

## Quick start

```bash
# 1. Initialize a project (interactive wizard)
ai-instructions init --registry https://ai-ctx.yourcompany.com

# 2. Check what's installed
ai-instructions list

# 3. Add more stacks
ai-instructions add docker terraform

# 4. Sync updates from registry
ai-instructions sync

# 5. Verify everything is correct (for CI)
ai-instructions verify
```

## Commands

| Command | Description |
|---------|-------------|
| `init` | Interactive setup wizard — select mode, vibe mode, and stacks |
| `add <stack...>` | Add stacks to an existing project |
| `remove <stack...>` | Remove stacks, with orphan dependency detection |
| `sync` | Download latest files from registry, update managed blocks |
| `verify` | CI command — check freshness, integrity, and managed blocks |
| `list` | List installed stacks (offline) |
| `outdated` | Show stacks with available updates |
| `search <term>` | Search available stacks in the registry |
| `doctor` | Run diagnostic checks |

## How it works

### Registry-driven

The CLI never hardcodes stack names or file lists. Everything comes from the registry. Adding a new stack (e.g. `rust`) means adding it to the registry repo — the CLI picks it up automatically.

### Dependency resolution

Stacks can declare dependencies. Selecting `laravel` automatically pulls in `php`. Removing `nuxt-ui` detects that `nuxt` and `vue` are now orphaned and offers to remove them.

```
ai-instructions init
# Select: laravel, nuxt-ui
# Resolved: php → laravel, vue → nuxt → nuxt-ui
```

### Marker-based injection

Managed content is injected between markers in `CLAUDE.md`, `AGENTS.md`, and `.cursorrules`. Content outside the markers is never touched.

```markdown
<!-- AI-INSTRUCTIONS:START — managed by ai-instructions, do not edit -->
# Company AI Instructions

This project uses the following instruction stacks: php, laravel

Read and follow ALL instruction files in the `ai-instructions/` folder:
- ai-instructions/php/coding-standards.md
- ai-instructions/php/testing.md
- ai-instructions/laravel/conventions.md
- ai-instructions/laravel/eloquent.md

These are mandatory company standards. Follow them strictly.
<!-- AI-INSTRUCTIONS:END -->

(your own project-specific instructions below are preserved)
```

### Lockfile

`ai-instructions-settings.json` tracks explicit stacks, resolved dependencies, versions, and SHA256 hashes. Commit this file to your repo.

### Vibe mode

Optional lighter rule set that excludes strict testing patterns and code review rules for prototyping/vibe-coding projects.

## CI usage

### Generate instructions locally with `gitlab-ci-local`

Use the `ai-instructions-generate` job to generate instruction files locally via `gitlab-ci-local`:

```yaml
# @Description Generates AI instructions file
ai-instructions-generate:
  tags: [shared-docker-executor]
  image:
    name: ai-instructions:latest
    entrypoint: [""]
  needs: []
  rules:
    - { if: $GITLAB_CI == "false", when: manual }
  script:
    - ai-instructions init go docker
  artifacts:
    paths: [.github/copilot-instructions.md, AGENTS.md]
```

Run it with:

```bash
gitlab-ci-local ai-instructions-generate
```

To list all available stacks (and see which are already installed):

```yaml
# @Description Lists all available AI instruction stacks
ai-instructions-stacks:
  tags: [shared-docker-executor]
  image:
    name: ai-instructions:latest
    entrypoint: [""]
  needs: []
  rules:
    - { if: $GITLAB_CI == "false", when: manual }
  script:
    - ai-instructions stacks
```

```bash
gitlab-ci-local ai-instructions-stacks
```

Both jobs require the `ai-instructions` Docker image to be built locally first (`docker build -t ai-instructions .`).

### GitLab CI

```yaml
ai-instructions:verify:
  stage: validate
  image: registry.yourcompany.com/tools/ai-instructions:latest
  variables:
    AI_INSTRUCTIONS_REGISTRY: "https://ai-ctx.yourcompany.com"
  script:
    - ai-instructions verify
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
  allow_failure: false
```

### Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Verification failed (outdated, tampered, or missing blocks) |
| 2 | Configuration error (missing settings file) |
| 3 | Network error (registry unreachable) |
| 4 | Usage error (bad arguments) |

The `--strict` flag on `verify` makes registry-unreachable a hard failure (exit 3) instead of a warning.

## Environment variables

| Variable | Description |
|----------|-------------|
| `AI_INSTRUCTIONS_REGISTRY` | Registry URL |
| `AI_INSTRUCTIONS_TOKEN` | Auth token for registry |
| `AI_INSTRUCTIONS_CI` | Force non-interactive mode |
| `AI_INSTRUCTIONS_NO_COLOR` | Disable colored output |
| `AI_INSTRUCTIONS_DEBUG` | Enable debug logging |

All are overridable via CLI flags (`--registry`, `--token`, `--debug`).

## Development

```bash
# Build
make build

# Run tests
make test

# Clean
make clean
```

### Project structure

```
cmd/ai-instructions/     CLI entrypoint
internal/
  cli/                   Command implementations (init, add, sync, ...)
  config/                Settings file read/write/validate
  registry/              HTTP client, cache, GitLab URL builder
  resolver/              Dependency resolution (topological sort)
  filemanager/           Download, hash, verify, cleanup
  injector/              Marker-based CLAUDE.md/AGENTS.md/.cursorrules injection
  ui/                    Prompts (charmbracelet/huh), styled output, spinner
  exitcodes/             Exit code constants
testdata/registry/       Sample registry for tests
```
