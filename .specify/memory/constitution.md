<!--
SYNC IMPACT REPORT
==================
Version change: (template / unversioned) → 1.0.0
Bump rationale: MAJOR — initial constitution creation; all principles and governance
defined for the first time from project README and codebase analysis.

Modified principles:
  (none — first version)

Added sections:
  - Core Principles (I–V)
  - Technical Stack Constraints
  - Development Workflow
  - Governance

Removed sections:
  (none — first version)

Templates reviewed and alignment status:
  ✅ .specify/templates/plan-template.md — Constitution Check section present; aligns
  ✅ .specify/templates/spec-template.md — FR/SC structure aligns with principles
  ✅ .specify/templates/tasks-template.md — Phase structure and observability tasks align
  ✅ No .specify/templates/commands/ directory — skip

Follow-up TODOs:
  (none — all fields resolved from README and codebase)
-->

# Go Word Create Constitution

## Core Principles

### I. CLI-First Design

All user-facing functionality MUST be accessible via command-line flags and arguments.
Entry points live exclusively under `cmd/`; each binary MUST document its flags in the
project README. The HTTP server is a secondary surface and MUST NOT be the only way to
invoke core document-generation logic.

- Binary flags use kebab-case or single-letter short forms (`-month`, `-output`, `-debug`).
- Standard I/O contract: results to a file or stdout; errors to stderr.
- Every command MUST support a debug/dry-run mode that produces human-readable console
  output instead of a binary artifact (via `-debug` flag or `LOGONLY=1` env var).

### II. Environment-Driven Configuration

All runtime configuration MUST be supplied through environment variables (loaded via
`.env` for local development). No credentials, URLs, or tunable parameters MAY be
hardcoded in source files.

- Required variables MUST be validated at process startup; the process MUST exit with a
  clear, actionable error message if any required variable is absent.
- `.env.example` MUST stay in sync with the full set of recognized variables; every
  variable MUST include a comment explaining its purpose and an example value.
- Sensitive values (API tokens, passwords) MUST never be committed to the repository.

### III. Package Separation

Business logic MUST live in `internal/` sub-packages; `cmd/` packages MUST contain only
wiring (flag parsing, env loading, calling `internal/` services). Cross-cutting concerns
(config, logging) belong in their own `internal/` packages.

- `internal/jiraservice` — Jira API client; MUST have no dependency on Word generation.
- `internal/word` — document generation; MUST have no dependency on Jira or HTTP.
- `internal/server` — HTTP handler; MUST delegate to service packages and MUST NOT
  contain business logic.
- Circular imports between `internal/` packages are forbidden.

### IV. Consistent Document Output

Every generated Word document MUST conform to the project's formatting standard so that
output is predictable and visually uniform across all commands and teams.

- **Font**: Aptos Narrow, 8 pt.
- **Borders**: single black borders on all table cells.
- **Header row**: blue background (`#365F91`), white bold text.
- **Cell margins**: 0.2 cm on all sides.
- **Section structure** for month reports: one "Closed Issues" section and one "Open
  Issues" section per team, with headings that include the month and team name.
- Changes to the formatting standard MUST be applied consistently across all
  document-generating code paths in the same PR.

### V. Observability

All commands and the HTTP server MUST provide enough runtime visibility to diagnose
issues without requiring a debugger.

- Every `cmd/` binary MUST support a `-debug` flag (or `LOGONLY` env var) that prints
  structured issue data to stdout instead of writing a file.
- The HTTP server MUST log each request (method, path, status, duration).
- Errors from Jira or document-generation MUST include context (team name, sprint/month,
  Jira issue key where applicable) — bare "failed" messages are forbidden.
- Log output MUST go to stderr; document output MUST go to stdout or the specified file.

## Technical Stack Constraints

- **Language**: Go 1.24.0 or higher. No language version downgrade without a PR
  that explains the rationale.
- **Core dependencies**:
  - `github.com/andygrunwald/go-jira` — Jira API client; do not re-implement Jira auth.
  - `github.com/carmel/gooxml` — Word document generation; do not introduce a second
    OOXML library.
  - `github.com/joho/godotenv` — `.env` loading (supports Environment principle).
- **Build tooling**: `make` targets are the canonical way to build, test, lint, and run
  commands. New binaries MUST have corresponding `make` targets and appear in `make help`.
- **Binary output**: compiled artifacts MUST go to `bin/` and MUST NOT be committed.
- **External services**: Jira Cloud only; on-premise Jira support requires an explicit
  feature specification before implementation.

## Development Workflow

- **Feature work**: create a feature branch, write a spec, get it reviewed, then implement.
- **Makefile gates** that MUST pass before any PR is merged:
  - `make fmt` — code formatted with `gofmt`.
  - `make lint` — zero lint errors (golangci-lint auto-installed if absent).
  - `make test` — all tests pass.
  - `make build` — all binaries compile without warnings.
- **PR scope**: one logical change per PR. Document-formatting changes, Jira client
  changes, and server changes SHOULD be in separate PRs unless tightly coupled.
- **Commit messages**: imperative present tense, ≤72 chars in the subject line.
  Reference the feature branch number or Jira issue key when applicable.
- **Secrets hygiene**: confirm `.env` is not staged before every commit.

## Governance

This constitution supersedes all informal conventions and undocumented practices.
Any conflict between this document and a README section, comment, or verbal agreement
is resolved in favour of the constitution.

**Amendment procedure**:
1. Open a PR that modifies this file with the proposed change.
2. Describe the motivation and impact in the PR description.
3. Increment `CONSTITUTION_VERSION` following semantic versioning:
   - MAJOR: principle removed, renamed, or its non-negotiable rules changed incompatibly.
   - MINOR: new principle or section added, or existing guidance materially expanded.
   - PATCH: clarification, wording improvement, or typo fix.
4. Update `LAST_AMENDED_DATE` to the merge date (ISO 8601: YYYY-MM-DD).
5. Propagate changes to affected templates in `.specify/templates/`.

**Compliance review**: each feature plan (`plan.md`) MUST include a Constitution Check
section that explicitly confirms or flags violations before Phase 0 research begins.
Violations MUST be justified in the Complexity Tracking table of the plan.

**Version**: 1.0.0 | **Ratified**: 2026-05-29 | **Last Amended**: 2026-05-29
