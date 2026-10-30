# Implementation Plan: Implementer Resolution Rule

**Branch**: `010-implementer-resolution` | **Date**: 2026-10-30 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/010-implementer-resolution/spec.md`

## Summary

Change how an issue's Implementer is derived. Today `resolveImplementer` returns the assignee at the first move to "In Review." and falls back to the current assignee, and it loses any assignee that was set at creation and never changed (about 15 of 35 issues in ST26.20 come back empty). The new rule is: the assignee just before the first move to "In Progress"; if there was none, the assignee just before the first move to "In Review."; if still none, the current assignee, then empty. The creation-time assignee is recovered from the `fromString` of the first assignee change in the existing changelog, so no extra Jira calls are needed. The change is confined to one function in `internal/jiraservice` plus its tests; the dashboard and MCP server pick it up automatically because both go through `sprintview.Build`.

## Technical Context

**Language/Version**: Go 1.24.0 (module `go-word-create`)

**Primary Dependencies**: No new dependencies. Uses `github.com/andygrunwald/go-jira` types already in use (`jira.Issue`, `jira.Changelog`, `jira.ChangelogItems`).

**Storage**: None.

**Testing**: `go test ./internal/jiraservice/...` (table-driven `TestResolveImplementer`); full gate is `make fmt lint test build`. No network or credentials needed.

**Target Platform**: macOS / Linux, same binaries as today (`web-sprint-labels-report`, `mcp-jira-reports`).

**Project Type**: Go CLI/HTTP/MCP project; this feature touches the shared Jira service package only.

**Performance Goals**: No extra Jira requests; the rule is one pass over each issue's changelog already fetched with `expand=changelog` (SC-003: load time within 10% of today).

**Constraints**: No new environment variables, flags, endpoints or response fields; JSON shape of `implementer` unchanged; Word, CSV and label report unaffected.

**Scale/Scope**: Up to a few hundred issues per sprint, each with a short changelog.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Checked against `.specify/memory/constitution.md` v1.2.0.

| Principle | Status | Notes |
|-----------|--------|-------|
| I. CLI-First | PASS | No new user-facing functionality or binaries; existing surfaces show the new value. |
| II. Environment-Driven | PARTIAL | The two status names ("In Progress", "In Review.") stay hardcoded as named constants, as "In Review." already is. Making them configurable is out of scope (spec Assumptions); see Complexity Tracking. |
| III. Package Separation | PASS | Logic stays in `internal/jiraservice`; `sprintview`, `dashboard` and `mcpserver` are unchanged and keep delegating. |
| IV. Consistent Document Output | N/A | No Word output changes. |
| V. Observability | PASS | No new failure paths: a missing changelog degrades to the current assignee, exactly as today. No new logging needed. |
| Tech Stack | PASS | No dependency changes. |
| Build Tooling | PASS | No new binary or `make` target. Gates: `make fmt`, `make lint`, `make test`, `make build`. |

**Post-design re-check**: unchanged. The design adds no packages, imports or configuration.

## Project Structure

### Documentation (this feature)

```text
specs/010-implementer-resolution/
├── plan.md                        # This file
├── research.md                    # Phase 0 output
├── data-model.md                  # Phase 1 output (resolution algorithm)
├── quickstart.md                  # Phase 1 output (how to verify)
├── contracts/
│   └── implementer-field.md       # Phase 1 output (meaning of the `implementer` field)
├── checklists/requirements.md
└── tasks.md                       # Created later by /speckit-tasks
```

### Source Code (repository root)

```text
internal/
├── jiraservice/
│   ├── jira.go             ← CHANGE: rewrite resolveImplementer(), add status-name constants
│   └── jira_test.go        ← CHANGE: update 2 cases whose expected value flips, add cases for US1-US3 and edge cases
├── mcpserver/
│   └── server.go           ← CHANGE (docs only): get_sprint_issues description states the implementer rule
├── sprintview/             ← NO CHANGES (passes Issue.Implementer through)
└── dashboard/              ← NO CHANGES (renders implementer || "—")
README.md                   ← CHANGE (docs only): mention the Implementer column and rule in the dashboard section
specs/006-implementer-column/spec.md ← CHANGE (docs only): note that the rule is superseded by 010
```

**Structure Decision**: Single-function change in the existing Jira service package. `GetIssuesFromSprintWithChangelog` already calls `resolveImplementer` and is the only producer of `Issue.Implementer`, so FR-009 (same rule on every surface) holds without touching other packages.

## Design Notes

### Resolution rule (see data-model.md for the full algorithm)

1. `current` = the issue's current assignee display name (empty if none).
2. No changelog: return `current` (FR-008).
3. `assignee` starts as the creation-time assignee: the `fromString` of the first `assignee` item in the whole changelog, or `current` when the changelog never changed the assignee (FR-004).
4. Walk the histories in order. Within one history entry apply its `assignee` items first, then look at its `status` items (FR-005, and the same order the current code uses).
5. First move to "In Progress": return `assignee` if non-empty; otherwise remember that the primary moment passed and later "In Progress" moves are ignored (FR-001, FR-002).
6. First move to "In Review.": remember `assignee` as the fallback (FR-002).
7. After the walk: fallback if non-empty, else `current` (FR-003).
8. Status names compare case-insensitively and exactly (FR-007); an assignee change to nobody sets `assignee` to empty (FR-006).

### Tests

- Update the two existing cases whose result changes by design: `in_review_no_assignee_change_in_history_fields_assignee_set` and `full_flow_assigned_at_creation_no_history_assignee_change` (both now return the current assignee, per FR-004).
- Add a test helper that sets `FromString` so reassignment history (`from` -> `to`) can be expressed.
- Add cases for every acceptance scenario and edge case in the spec: reassigned after start, reassigned before start, first of several "In Progress" moves, empty at "In Progress" then fallback, skipped "In Progress", same-entry assignment, assignee removed before the move, never reached either status, nil and empty changelog.

### Risks

- **Changelog completeness**: the rule trusts the changelog returned by the search call. If Jira truncates very long histories, the result for such an issue could be wrong; this limitation exists today and is not addressed here.
- **History order**: the code relies on Jira returning histories oldest first, as it does today.
- **Existing sprints change**: values shown for past sprints will change after deployment, because the rule changes. This is intended.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Principle II: status names hardcoded as constants | The rule is defined by the two workflow status names, and the existing code already hardcodes "In Review." | Two new environment variables would add configuration and documentation for names that are fixed in this workflow; the spec lists the names as an assumption. Revisit if another team's workflow differs. |
