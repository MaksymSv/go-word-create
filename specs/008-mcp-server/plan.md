# Implementation Plan: MCP Server for Jira Sprint & Label Reports

**Branch**: none (built directly on `dev`) | **Date**: 2026-10-29 (initial), 2026-10-30 (`sprint_issues` + CSV) | **Spec**: [spec.md](./spec.md)

**Status**: Implemented (retroactive plan written 2026-10-30 from the code)

**Input**: Feature specification from `specs/008-mcp-server/spec.md`

## Summary

A third surface, `cmd/mcp-jira-reports`, serves the existing Jira reporting logic to LLM clients over the Model Context Protocol (stdio). Seven typed tools wrap shared `internal/` code (`sprintview`, `labelreport`, `issuecsv`, `jiraservice`); only `set_issue_label` writes to Jira and it requires an explicit `confirm=true`. The initial six tools landed in `4ea2ec1` (2026-10-29); `sprint_issues` with CSV text output landed in `cd0dfcd` (2026-10-30). A project-scoped `.mcp.json` and a companion skill make it usable from Claude Code.

## Technical Context

**Language/Version**: Go 1.24.0 (module `go-word-create`)

**Primary Dependencies**: `github.com/modelcontextprotocol/go-sdk` v1.4.0 (new); existing `go-jira`, `godotenv`. Tool input/output JSON Schemas are generated from the Go structs and their `jsonschema` tags.

**Storage**: None. Every call queries Jira live.

**Testing**: `go test ./internal/mcpserver/...` using a `fakeJira` that implements `mcpserver.Jira`, plus in-memory client/server protocol tests (`TestServerEndToEnd`, `TestSprintIssuesOverProtocol`). `internal/sprintview` has its own tests. No test touches the network.

**Target Platform**: macOS / Linux, launched as a child process by an MCP client.

**Project Type**: Single Go module with multiple binaries; this adds one stdio server binary.

**Performance Goals**: None specified; latency is dominated by Jira API calls.

**Constraints**: Standard output carries only protocol frames; logging goes to standard error. The server never writes files. Handler errors are tool errors, not protocol errors.

**Scale/Scope**: Internal tool; a single client per process; a handful of teams and sprints of up to a few hundred issues.

## Constitution Check

*Checked against `.specify/memory/constitution.md` v1.1.0 (which already names the MCP server as a third surface).*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. CLI-First | ⚠️ PARTIAL | The binary has flags (`-workdir`, `-debug`) and the constitution explicitly allows MCP as a third surface that "MUST reuse `internal/` logic, not duplicate it". It reuses `sprintview`, `labelreport`, `issuecsv`, `jiraservice` and `ParseSprintNames`. Some logic lives only in `internal/mcpserver` and is not shared with the CLIs or dashboard: status filtering, per-label summarising, component scoping for `sprint_issues`, and the Closed/Open month split (`report.go`, `tools.go`). The dashboard computes the same summaries again in JavaScript. |
| II. Environment-Driven | ⚠️ PARTIAL | Uses `config.Load()`, so required variables are validated at startup with a clear error. `.env.example` still omits `REPORT_LABELS` and `ISSUE_TYPES`, which the tools expose through `list_teams`. The server version (`1.0.0`) is a source constant. |
| III. Package Separation | ✅ PASS | `cmd/mcp-jira-reports/main.go` is wiring only (flags, `chdir`, config, Jira client, run). Logic is in `internal/mcpserver`, which depends on a narrow `Jira` interface rather than the concrete client. `sprintview` is a new shared package used by both the dashboard and the server. No circular imports. |
| IV. Consistent Document Output | ✅ N/A | No Word documents. CSV text comes from `internal/issuecsv`. |
| V. Observability | ⚠️ PARTIAL | `-debug` only adds file/line to log output on standard error; it does not print structured issue data, because standard output is reserved for protocol frames. There is no per-tool-call request log (method, tool, duration). Errors do carry context (team, sprint, month, issue key). Startup logs team and label counts to standard error. |
| Tech Stack | ✅ PASS | The MCP SDK is the dependency the constitution names; no second MCP library. |
| Build Tooling | ✅ PASS | `make build-mcp` builds `bin/mcp-jira-reports`, is part of `make build` and appears in `make help`. There is no `run-*` target, which is reasonable for a stdio server. |

Deviations are documented rather than blocking; they are candidates for follow-up (per-call logging, `.env.example`, moving shared aggregation into `internal/`).

## Project Structure

### Documentation (this feature)

```text
specs/008-mcp-server/
├── spec.md              ← retroactive feature spec
└── plan.md              ← this file (as-built)
```

No `research.md`, `data-model.md`, `contracts/` or `tasks.md` exist for this feature.

### Source Code (repository root)

```text
cmd/mcp-jira-reports/main.go        ← flags, chdir, config, Jira client, stdio Run
internal/mcpserver/
├── server.go                       ← instructions text, Jira interface, tool registration, annotations
├── tools.go                        ← tool input/output types and handlers
├── report.go                       ← status filter, label summary, label-report shaping
└── server_test.go                  ← fakeJira, handler tests, protocol tests
internal/sprintview/                ← shared sprint view (also used by internal/dashboard)
internal/issuecsv/                  ← CSV writer + file naming (used by sprint_issues)
internal/jiraservice/sprintnames.go ← ParseSprintNames (shared with the CLIs)
.mcp.json                           ← project-scoped client registration
.claude/skills/jira-sprint-reports/ ← SKILL.md + references/cli-output.md
Makefile                            ← MCP_CMD, build-mcp (included in build)
```

**Structure Decision**: The server is a thin adapter. Handlers call a narrow `Jira` interface (sprint view source, sprints for a board, sprint issues by name, month issues, label update) so tests substitute a fake.

---

## As-Built Design

### Process and flags

| Flag | Default | Effect |
|------|---------|--------|
| `-workdir` | `.` | The process changes into this directory before `config.Load()` so `.env` resolves when the client starts it elsewhere. |
| `-debug` | off | Adds file/line to log output (standard error). |

Startup order: set log output to standard error, parse flags, `chdir`, `config.Load()`, create the Jira client, then run the server on `mcp.StdioTransport` until interrupted. Failures call `log.Fatalf` (standard error). The server name is `jira-reports`, title "Jira sprint & label reports", version `1.0.0`.

### Registration

`.mcp.json` registers `jira-reports` with command `./bin/mcp-jira-reports` and args `-workdir .`. It contains no secrets (credentials stay in `.env`). The README documents `claude mcp add` and the Claude Desktop / Codemie `mcpServers` JSON. `.claude/settings.local.json` enables the `jira-reports` server for this checkout.

### Tools

All tools are annotated read-only and open-world except `set_issue_label` (not read-only, non-destructive, idempotent, open-world). The server also sends an instructions string describing the typical flow and the write protocol.

| Tool | Input | Output (structured) | Notes |
|------|-------|---------------------|-------|
| `list_teams` | none | `teams[{componentName, boardName}]`, `configuredLabels`, `issueTypes` | From config only; no Jira call. |
| `list_sprints` | `team`, `limit?` | `team`, `board`, `sprints[{id, name, state, startDate?, endDate?}]` | Default limit 5; newest first; unknown team errors with the configured list. |
| `get_sprint_issues` | `sprint_id`, `statuses?` | `sprintId`, `configuredLabels`, `availableStatuses`, `appliedStatuses?`, `unmatchedStatuses?`, `totals`, `unlabeled`, `labelSummary[{label,count,storyPoints}]`, `issues[]` | Uses `sprintview.Build`; no issue-type filter. Summaries cover the filtered set. |
| `sprint_issues` | `sprints[]`, `team?`, `statuses?`, `format?` (`json`\|`csv`) | `sprints[{sprint, count, storyPoints, availableStatuses, unmatchedStatuses?, issues[{type,key,summary,epic,storyPoints,status,URL}], fileName?, csv?}]` | Names may also be comma-joined inside one entry. Boards from config (or the team's board); team also scopes by component after fetch. Uses configured issue types. CSV built in memory with `issuecsv.Write`; name from `issuecsv.FileName(sprint, now)`. |
| `sprint_label_report` | `sprints[]`, `team?` | `configuredLabels`, `reports[{sprint, components[{component, issueCount, storyPoints, labelGroups[], unlabeledIssues[], labeledCountPct, labeledStoryPointsPct}]}]` | Wraps `labelreport.Aggregate`; percentages rounded to 1 decimal; `issueCount` is the unique issue count. A team restricts boards and the returned component. |
| `month_issues` | `month` (`YYYY.MM`), `team?` | `month`, `teams[{component, closedCount, openCount, closed[], open[]}]` | Closed means `Status == "Closed"`; everything else is Open. |
| `set_issue_label` | `issue_key`, `label`, `action` (`add`\|`remove`), `confirm?` | `applied`, `issueKey`, `label`, `action`, `message` | See the write protocol below. |

### Write protocol (`set_issue_label`)

1. Validate the key against `^[A-Za-z][A-Za-z0-9_]*-[0-9]+$`, lower-case the action and require `add`/`remove`, and require the label to be in `ReportLabels`. Any failure is a tool error and Jira is not called.
2. If `confirm` is false (the default): return `applied=false` with "PREVIEW ONLY, nothing changed. Would … Ask the user to approve, then call again with confirm=true." No Jira call.
3. If `confirm` is true: call `UpdateIssueLabel(key, label, action == "add")` once. On error return a tool error prefixed with the issue key; on success return `applied=true`.

The companion skill adds the human-side rules: gather the list of changes, preview each, get explicit approval for those exact changes, then apply one at a time and report per issue.

### Standard output rule

Standard output is the protocol channel. `main` calls `log.SetOutput(os.Stderr)` before anything else; no handler prints. CSV is returned as a string field and the server never opens files.

### Companion skill

`.claude/skills/jira-sprint-reports/SKILL.md` contains a tool picker, workflows, metric semantics (non-exclusive labels, percentage denominators, "No Component" bucket, 0 SP meaning unestimated), the write protocol, a CLI fallback (`make run-sprint-label-report`, `make run-month … LOGONLY=1`) and troubleshooting. `references/cli-output.md` documents the CLI table layouts. The skill forbids reading or printing `.env`.

### Shared code introduced or extracted

- `internal/sprintview` (new in `4ea2ec1`): `Build` loads epics (failure tolerated and logged), loads sprint issues with changelog, marks which configured labels each issue carries. The dashboard handler was changed to use it.
- `internal/jiraservice/sprintnames.go`: comma-separated sprint-name parsing shared with the CLIs.
- `internal/issuecsv` (new in `cd0dfcd`): CSV writer and `<sprint>_<YYYY-MM-DD>.csv` naming, also used by `get-sprint-issues -format csv`.

### Error Handling

Handler errors are returned as Go errors and the SDK converts them to tool error results (asserted in `TestServerEndToEnd`). Messages carry context: team, board, sprint name or ID, month, issue key. Unknown teams list the configured names and note case-sensitivity.

### Tests

`internal/mcpserver/server_test.go` covers: status filter and summary, unlabeled counts, bad sprint ID, tolerated epic failure, label-report mapping and team scoping, error cases, month split, default sprint limit, `set_issue_label` (preview makes no Jira call, confirm applies, invalid input makes no call, Jira failure surfaces with the key), multi-sprint/status/CSV output for `sprint_issues`, team filter and error cases, and two in-memory protocol tests (tool registration and annotations; `sprint_issues` over the protocol).

---

## Implementation Notes

### Files Changed (Summary)

| File | Change |
|------|--------|
| `cmd/mcp-jira-reports/main.go` | New binary |
| `internal/mcpserver/*` | New package (server, tools, report, tests) |
| `internal/sprintview/*` | New shared package; dashboard now delegates to it |
| `internal/issuecsv/*` | New package (CSV, with `sprint_issues`) |
| `internal/jiraservice/sprintnames.go` | New shared parser |
| `.mcp.json`, `.claude/skills/jira-sprint-reports/*` | Client registration and skill |
| `Makefile`, `README.md`, `go.mod`, `go.sum` | `build-mcp` target, MCP docs, SDK dependency |
| `.specify/memory/constitution.md` | v1.1.0: MCP as third surface, SDK as core dependency |

### Known Gaps and Inconsistencies (as of 2026-10-30)

- No per-tool-call logging, so Principle V's request-visibility intent is only met by startup and error messages.
- `get_sprint_issues` ignores the configured issue types while `sprint_issues`, `sprint_label_report` and `month_issues` apply them.
- `sprint_issues` splits comma-joined names in a single entry; `sprint_label_report` does not.
- Summary/filter logic is duplicated across the server (Go) and the dashboard (JavaScript).
- `.env.example` is missing `REPORT_LABELS` and `ISSUE_TYPES`.
- The server version string is hard-coded.
