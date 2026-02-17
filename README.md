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
│  Commands: init, list, sync, verify, version      │
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
# 1. Initialize a project with stacks
ai-instructions init php laravel --registry https://ai-ctx.yourcompany.com

# 2. List all available stacks (marks installed ones)
ai-instructions list

# 3. Sync updates from registry
ai-instructions sync

# 4. Verify everything is correct (for CI)
ai-instructions verify
```

## Commands

| Command | Description |
|---------|-------------|
| `init <stack> [stack...]` | Initialize project with given stacks, resolve dependencies, download files |
| `list` | List all registry stacks grouped by category, mark installed ones |
| `sync` | Download latest files from registry, update managed blocks |
| `verify [--strict]` | CI gate — check freshness, integrity, and managed blocks |
| `version` | Print version information |

## How it works

### Registry-driven

The CLI never hardcodes stack names or file lists. Everything comes from the registry. Adding a new stack (e.g. `rust`) means adding it to the registry repo — the CLI picks it up automatically.

### Dependency resolution

Stacks can declare dependencies. Selecting `laravel` automatically pulls in `php`.

```bash
ai-instructions init laravel nuxt
# Resolved: php → laravel, vue → nuxt
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

## CI usage

### `gitlab-ci-local` jobs

Each command has a corresponding CI job with a `# @Description` comment for `gitlab-ci-local --list`. All jobs require the `ai-instructions` Docker image to be built locally first (`docker build -t ai-instructions .`).

```yaml
# @Description Initializes AI instructions with given stacks
ai-instructions-init:
  tags: [shared-docker-executor]
  image:
    name: ai-instructions:latest
    entrypoint: [""]
  needs: []
  rules:
    - { if: $GITLAB_CI == "false", when: manual }
  script:
    - ai-instructions init go docker

# @Description Lists all available AI instruction stacks
ai-instructions-list:
  tags: [shared-docker-executor]
  image:
    name: ai-instructions:latest
    entrypoint: [""]
  needs: []
  rules:
    - { if: $GITLAB_CI == "false", when: manual }
  script:
    - ai-instructions list

# @Description Syncs AI instruction files from registry
ai-instructions-sync:
  tags: [shared-docker-executor]
  image:
    name: ai-instructions:latest
    entrypoint: [""]
  needs: []
  rules:
    - { if: $GITLAB_CI == "false", when: manual }
  script:
    - ai-instructions sync

# @Description Verifies AI instruction files are up to date
ai-instructions-verify:
  tags: [shared-docker-executor]
  image:
    name: ai-instructions:latest
    entrypoint: [""]
  needs: []
  script:
    - ai-instructions verify

# @Description Prints AI instructions CLI version
ai-instructions-version:
  tags: [shared-docker-executor]
  image:
    name: ai-instructions:latest
    entrypoint: [""]
  needs: []
  rules:
    - { if: $GITLAB_CI == "false", when: manual }
  script:
    - ai-instructions version
```

Run any job with:

```bash
gitlab-ci-local ai-instructions-init
gitlab-ci-local ai-instructions-list
gitlab-ci-local ai-instructions-sync
gitlab-ci-local ai-instructions-verify
gitlab-ci-local ai-instructions-version
```

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
| `AI_INSTRUCTIONS_BRANCH` | Registry branch (default: master) |
| `AI_INSTRUCTIONS_TOKEN` | Auth token for registry |
| `AI_INSTRUCTIONS_NO_COLOR` | Disable colored output |
| `AI_INSTRUCTIONS_DEBUG` | Enable debug logging |

All are overridable via CLI flags (`--registry`, `--branch`, `--token`, `--debug`).

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
  cli/                   Command implementations (init, list, sync, verify)
  config/                Settings file read/write/validate
  registry/              HTTP client, cache, GitLab URL builder
  resolver/              Dependency resolution (topological sort)
  filemanager/           Download, hash, verify, cleanup
  injector/              Marker-based CLAUDE.md/AGENTS.md/.cursorrules injection
  ui/                    Styled terminal output
  exitcodes/             Exit code constants
testdata/registry/       Sample registry for tests
```
