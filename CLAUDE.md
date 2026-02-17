<!-- AI-INSTRUCTIONS:START — managed by ai-instructions, do not edit -->
# Company AI Instructions

If any instruction file is missing or inaccessible, stop and ask for it before proceeding.

This project uses the following instruction stacks: php, laravel, vue, nuxt

Read and follow ALL instruction files in the `ai-instructions/company-instructions/` folder:
- ai-instructions/company-instructions/php/coding-standards.md
- ai-instructions/company-instructions/php/testing.md
- ai-instructions/company-instructions/laravel/conventions.md
- ai-instructions/company-instructions/laravel/coding-standards.md
- ai-instructions/company-instructions/laravel/form-requests.md
- ai-instructions/company-instructions/laravel/testing.md
- ai-instructions/company-instructions/nuxt/conventions.md
- ai-instructions/company-instructions/nuxt/vue-components.md
- ai-instructions/company-instructions/nuxt/typescript.md
- ai-instructions/company-instructions/nuxt/scss.md
- ai-instructions/company-instructions/nuxt/composables.md
- ai-instructions/company-instructions/nuxt/api-integration.md
- ai-instructions/company-instructions/nuxt/module-development.md
- ai-instructions/company-instructions/nuxt/testing.md
- ai-instructions/company-instructions/nuxt/branding.md

These are mandatory company standards. Follow them strictly.
<!-- AI-INSTRUCTIONS:END -->

## 1. Project Overview
* **Name:** ai-instructions
* **Purpose:** In house developer tooling for code generation and AI interactions in projects and workflows. Can be extended with agents skills, hooks, custom code generation templates and more.
* **Architecture:** Hexagonal / Clean Architecture (Standard Go Layout).

## 3. Project Structure
We follow the **Standard Go Project Layout**:

* `cmd/`: Entry points (main applications).
* `internal/`: Private application and business logic.

To add dynamic variables to the helm chart, you can add them to `values.env.yaml` which is envsubsted before applying the helm chart. The env to be substed is defined in .gitlab files. For dynamic env depending on environment, look in .gitlab-ci-env-tiers.yml. For deployment specific env, look in .gitlab-ci-{deploymenTarget}.yaml.

## 4. Coding Standards (Strict)

### General Style
* **Formatting:** Always run `gofmt` (or `goimports`).
* **Linting:** Code must pass `golangci-lint` with default settings. The linter can be run using `gitlab-ci-local golangci-lint-fmt`
* **Naming:**
    * Use `CamelCase` for exported identifiers.
    * Use `camelCase` for unexported identifiers.
    * Acronyms should be consistent (e.g., `ServeHTTP`, not `ServeHttp`).
    * Variable names should be short (1-2 chars) for small scopes, descriptive for larger scopes.
* **Data validation:** Always ensure data integrity by validating inputs at the boundaries (e.g., API and kafka consumers). Use `go-playground/validator`. Where needed database constraints should also be used.

### Error Handling
* **Wrap Errors:** Never return a raw error from a sub-call if you can add context. Use `fmt.Errorf("doing action: %w", err)`.
* **Check Errors:** Never ignore errors using `_`. Handle them or bubble them up.
* **Sentinel Errors:** Define static errors in the domain layer (e.g., `var ErrNotFound = errors.New("not found")`).

### Concurrency & Context
* **Context Propagation:** Every I/O bound function (DB, HTTP, External API) must accept `ctx context.Context` as the first argument.
* **Cancellation:** Always respect context cancellation.
* **Goroutines:** Never spawn a goroutine without a mechanism to stop it (waitgroups or context).

### Modern Go Features
* **Slices/Maps:** Use the `github.com/samber/lo` library for functional programming helpers (e.g., `lo.Map`, `lo.Filter`).
* **Generics:** Use Generics to reduce code duplication, but do not overuse them where interfaces suffice.
* **Any:** Use `any` instead of `interface{}`.

### Production considerations
* **Running production** The project is live in production. When you make changes, you should consider if it is breaking for the running functionality, and if the deployment can be done without downtime. This is especially important for database migrations, helm chart changes and API breaking changes.
* **Database Migrations:** Always use a non-blocking variant if possible. Production tables may contain billions of rows, and migrations should not cause downtime. If a heavy migration must be performed, make sure you prompt for confirmation and notify it in the MR description.

## 5. Testing Guidelines
* **Pattern:** Use Table-Driven Tests for all logic.
* **Packages:** Use `testing` package.
* **Location:** Test files go next to the code they test (`service_test.go`).
* **Mocks:** Use interface-based mocking. Generate mocks using `mockery` or `gomock`.
* **Coverage:** Focus on business logic (Services/Domain) over boilerplate.

## 6. "Do Nots" (Guardrails)
* ❌ **No Panic:** Never use `panic()` in application code. Return errors.
* ❌ **No Global State:** Avoid global variables. Use Dependency Injection (passing structs).
* ❌ **No `dot` imports:** (e.g., `import . "fmt"`).
* ❌ **No Circular Dependencies:** Architect your packages to avoid cycles.
* ❌ **Do not use comments to describe what commit messages should describe.** Code should be self-explanatory.
## 7. Tools
* Use `gitlab-ci-local` to run CI jobs locally for linting and testing. For example `gitlab-ci-local sqlc`.

Never run sqlc, templ, oapi-codegen normally, always run it with `gitlab-ci-local sqlc` etc to ensure consistent code generation.
Use `gitlab-ci-local --list` to see all available jobs.

You can use the go cli for running tests, formatting and building, but for code generation, always use `gitlab-ci-local` to ensure proper isolation and versions.

If in doubt of usage of any go package, always use the godoc mcp server for up-to-date documentation.

If in doubt of usage or best practices of postgresql and timescaledb, always use pg-aiguide mcp server for up-to-date documentation.

## 8. Branching & MRs
## Branching

Never commit directly on master. If on master, create a new branch first.

Branch naming convention: `{initials}/{descriptive-name-with-dashes}`

Get initials from the local part of `git config user.email` (the part before @).

Example: if email is `hello@cego.dk`, branch name could be `hello/add-claude-context`.

## Merge Requests

This project is hosted on GitLab. Use `glab` CLI (not `gh`) for Merge Requests.

### Creating or updating an MR

1. Run `/change-management` to generate the MR description (CIATF assessment)
2. Check if an MR already exists for the current branch: `glab mr list --source-branch=$(git branch --show-current)`
3. If MR exists, update it: `glab mr update <mr-number> --description "..."`
4. If no MR exists, create one: `glab mr create --title "Title" --description "..." --target-branch master`

Use the output from `/change-management` as the description.

## 9. Code Reviews
When reviewing code, focus on:
* Correctness: Does the code do what it is supposed to do? If you are in doubt of the desired behavior, check the MR description. If it is at all not clear, make sure to add a comment asking for clarification.
* Readability: Is the code easy to understand? If not, suggest improvements.
* Best Practices: Does the code follow the coding standards outlined in this document? If not, suggest improvements.
* Tests: Are there sufficient tests? Do they cover edge cases? If not, suggest improvements.

Leave comments on the MR for any issues found. Leave a comment per issue, not a single comment with multiple issues.
