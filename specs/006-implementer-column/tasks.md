# Tasks: Implementer Column

**Input**: Design documents from `specs/006-implementer-column/`

**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/api.md ✓

**Organization**: Single user story; tasks grouped by layer (data → service → API → frontend).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to

---

## Phase 1: Setup

**Purpose**: No project initialization required — this feature modifies an existing Go module. Only confirm the build baseline passes before changes.

- [x] T001 Verify baseline: run `go build ./...` from repo root to confirm clean build before changes

---

## Phase 2: Foundational (Blocking Prerequisite)

**Purpose**: Add `Implementer` field to `jiraservice.Issue` — required by all downstream tasks.

**⚠️ CRITICAL**: T003 and T004 cannot be completed until this phase is done.

- [x] T002 Add `Implementer string` field to the `Issue` struct in `internal/jiraservice/jira.go` (after the `Components` field)

**Checkpoint**: `go build ./...` still passes; existing callers compile unchanged (empty string zero value).

---

## Phase 3: User Story 1 - Implementer in Web Dashboard (Priority: P1) 🎯 MVP

**Goal**: Display an "Implementer" column in the web dashboard's sprint issues table, showing who was assigned when the issue first moved to "In Progress" (with current-assignee fallback).

**Independent Test**: Start `make run-web-dashboard`, select a team and sprint, and confirm:
1. The "Implementer" column appears between "Epic" and "SP" headers.
2. Issues with "In Progress" history show the correct assignee name.
3. Issues never moved to "In Progress" show the current assignee (or "—").

### Implementation

- [x] T003 [P] [US1] Add `resolveImplementer(issue jira.Issue) string` private helper to `internal/jiraservice/jira.go` — walks `issue.Changelog.Histories` in order; within each entry applies assignee items first then checks for first "in progress" status match (case-insensitive); falls back to `issue.Fields.Assignee.DisplayName`; returns `""` if neither found
- [x] T004 [P] [US1] Add `Implementer string \`json:"implementer"\`` field to `DashboardIssue` struct in `internal/dashboard/api.go` (after the `Epic` field, before `StoryPoints`)
- [x] T005 [US1] Add `GetIssuesFromSprintWithChangelog(sprintID int, epicNames map[string]string, typeFilter map[string]struct{}) ([]Issue, error)` method to `internal/jiraservice/jira.go` — fetches via JQL `sprint = {sprintID} ORDER BY created ASC` with `SearchOptions{MaxResults: 1000, Expand: "changelog"}`, maps each issue including `Implementer: resolveImplementer(issue)` (depends on T002, T003)
- [x] T006 [US1] Update `getSprintIssues` handler in `internal/dashboard/handler.go` — replace `h.jira.LoadIssuesFromSprint(sprintID, epicNames, nil)` with `h.jira.GetIssuesFromSprintWithChangelog(sprintID, epicNames, nil)` and add `Implementer: issue.Implementer` to the `DashboardIssue` mapping (depends on T004, T005)
- [x] T007 [P] [US1] Add "Implementer" column to `renderTable` in `internal/dashboard/web/app.js` — add `'Implementer'` to the headers array after `'Epic'`; add `tr.insertCell().textContent = issue.implementer \|\| '—'` after the Epic cell

**Checkpoint**: `go build ./...` passes. Load a sprint in the dashboard and confirm the Implementer column is populated correctly.

---

## Phase 4: Polish & Cross-Cutting Concerns

- [x] T008 Run `go build ./...` and confirm clean build with all changes in place
- [x] T009 [P] Run `make fmt` to ensure all modified Go files are correctly formatted
- [x] T010 [P] Run `make lint` to confirm zero lint errors in modified files

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — BLOCKS T005, T006
- **User Story 1 (Phase 3)**: T003 and T004 can start after T002; T005 needs T002+T003; T006 needs T004+T005; T007 is independent (different file)
- **Polish (Phase 4)**: Depends on all Phase 3 tasks complete

### Within User Story 1

```
T002 (Issue.Implementer field)
  ├── T003 (resolveImplementer helper)   [P with T004, T007]
  │     └── T005 (GetIssuesFromSprintWithChangelog)
  │           └── T006 (handler update)
  │                 └── T008/T009/T010 (polish)
  └── T004 (DashboardIssue.Implementer) [P with T003, T007]
        └── T006 (handler update)

T007 (app.js column)  — parallel to all Go tasks, depends only on contract knowledge
```

### Parallel Opportunities

- T003 and T004 can run in parallel (different files: `jira.go` vs `api.go`)
- T007 (app.js) can run in parallel with all Go tasks
- T009 and T010 (fmt + lint) can run in parallel in the Polish phase

---

## Parallel Example: User Story 1

```
# After T002 completes, start these in parallel:
Task T003: Add resolveImplementer helper in internal/jiraservice/jira.go
Task T004: Add Implementer field to DashboardIssue in internal/dashboard/api.go
Task T007: Add Implementer column to renderTable in internal/dashboard/web/app.js

# After T003 completes:
Task T005: Add GetIssuesFromSprintWithChangelog in internal/jiraservice/jira.go

# After T004 + T005 complete:
Task T006: Update getSprintIssues handler in internal/dashboard/handler.go
```

---

## Implementation Strategy

### MVP (User Story 1 Only)

1. Complete Phase 1: confirm clean baseline build (T001)
2. Complete Phase 2: add `Implementer` field to `Issue` struct (T002)
3. Complete Phase 3: implement all US1 tasks (T003–T007) — can be done in ~30 min
4. **STOP and VALIDATE**: load the dashboard, confirm Implementer column shows correctly
5. Complete Phase 4: fmt + lint + build check

### Total Scope

4 source files, 6 implementation tasks, ~50 lines of Go + 2 lines of JS.

---

## Notes

- `resolveImplementer` must handle `nil` Changelog gracefully (issues with no history)
- Two-pass within each history entry: apply assignee changes before checking status — handles "assigned + moved to In Progress" in the same action
- `GetIssuesFromSprintWithChangelog` uses JQL instead of `Sprint.GetIssuesForSprint` because the Sprint API wrapper does not support `Expand` options
- `LoadIssuesFromSprint` is left unchanged — other code paths (Word reports, label report) do not need changelog data
