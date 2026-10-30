# Tasks: Implementer Resolution Rule

**Input**: Design documents from `specs/010-implementer-resolution/`

**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/implementer-field.md ✓, quickstart.md ✓

**Tests**: Included. The existing table-driven `TestResolveImplementer` must change anyway (two cases flip by design), so each story extends it.

**Organization**: One function (`resolveImplementer`) serves all three stories, so stories build on each other in priority order. Each phase ends with a checkpoint that can be run on its own.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: User story the task belongs to (US1, US2, US3)

## Path Conventions

Single Go module at the repository root. Code lives in `internal/jiraservice/`; tests sit next to it.

---

## Phase 1: Setup

**Purpose**: No new project structure is needed. Confirm the baseline is green before changing anything.

- [x] T001 Verify baseline: run `go build ./... && go test ./internal/jiraservice/...` from the repo root and confirm both pass before changes

---

## Phase 2: Foundational

**Purpose**: Shared pieces every story needs: the status-name constants and test helpers that can express reassignment history.

**⚠️ CRITICAL**: US1 to US3 cannot start until this phase is done.

- [x] T002 [P] Add two unexported constants to `internal/jiraservice/jira.go`: `implementerPrimaryStatus = "In Progress"` and `implementerFallbackStatus = "In Review."` (with the trailing period), placed just above `resolveImplementer`
- [x] T003 [P] Add test helpers to `internal/jiraservice/jira_test.go`: `assignChange(from, to string) jira.ChangelogItems` (field `assignee`, sets both `FromString` and `ToString`) and `statusChange(to string) jira.ChangelogItems` (field `status`, sets `ToString`); keep the existing `items`, `history` and `assignee` helpers

**Checkpoint**: `go build ./... && go vet ./internal/jiraservice/...` pass (unused constants are allowed at package level).

---

## Phase 3: User Story 1 - Implementer Is Whoever Held the Issue When Work Started (Priority: P1) 🎯 MVP

**Goal**: The Implementer is the assignee just before the first move to "In Progress", including an assignee set at creation and never changed, and an assignee set in the same change as the move.

**Independent Test**: `go test ./internal/jiraservice/ -run TestResolveImplementer` passes with the US1 cases below.

### Tests for User Story 1

- [x] T004 [US1] In `internal/jiraservice/jira_test.go`, update the two existing cases whose result changes by design and fix their comments: `in_review_no_assignee_change_in_history_fields_assignee_set` now expects `"Alice"`, and `full_flow_assigned_at_creation_no_history_assignee_change` now expects `"Alice"` (the creation-time assignee counts, FR-004)
- [x] T005 [US1] In `internal/jiraservice/jira_test.go`, add US1 cases using the new helpers: assigned A then moved to In Progress then reassigned B returns A; A reassigned to B before the move returns B; assigned at creation and reassigned only after the move returns the original assignee via the first change's `FromString`; assignee set in the same history entry as the move counts; several moves to In Progress use the first; status name matched case-insensitively and "In Progress (Dev)" does not match

### Implementation for User Story 1

- [x] T006 [US1] Rewrite `resolveImplementer` in `internal/jiraservice/jira.go` per `data-model.md`: read the current assignee; return it when `issue.Changelog` is nil; start `assignee` from the `FromString` of the first `assignee` item in all histories (else the current assignee); walk histories in order applying a history's `assignee` items before its `status` items; on the first move to `implementerPrimaryStatus` mark it seen and return `assignee` if non-empty (later moves are ignored); otherwise return the current assignee. Update the doc comment to describe the new rule

**Checkpoint**: `go test ./internal/jiraservice/ -run TestResolveImplementer -v` passes. The MVP is shippable here: issues that passed through "In Progress" show the right implementer.

---

## Phase 4: User Story 2 - Fall Back to the Assignee Before "In Review." (Priority: P2)

**Goal**: When there was no assignee at the first move to "In Progress", or the issue never went through it, the Implementer is the assignee just before the first move to "In Review.".

**Independent Test**: The US2 cases below pass, and US1 cases still pass.

### Tests for User Story 2

- [x] T007 [US2] In `internal/jiraservice/jira_test.go`, add US2 cases: unassigned at In Progress and assigned C before In Review. returns C; skipped In Progress and assigned C before In Review. returns C; an assignee at In Progress wins over a different assignee at In Review.; assignee removed (`assignChange("A", "")`) before the move to In Progress falls back to the In Review. assignee; first In Progress move empty and a later In Progress move with an assignee still falls back to In Review.; first In Review. move wins over later ones; "In Review" without the trailing period does not match

### Implementation for User Story 2

- [x] T008 [US2] Extend `resolveImplementer` in `internal/jiraservice/jira.go`: on the first move to `implementerFallbackStatus` record the then-current `assignee` as the fallback; after the walk return the fallback when non-empty, before returning the current assignee

**Checkpoint**: `go test ./internal/jiraservice/ -run TestResolveImplementer -v` passes with US1 and US2 cases.

---

## Phase 5: User Story 3 - Sensible Value When Neither Moment Applies (Priority: P3)

**Goal**: With no usable assignee at either moment the result is the current assignee, then empty, and nothing fails.

**Independent Test**: The US3 cases below pass.

### Tests for User Story 3

- [x] T009 [US3] In `internal/jiraservice/jira_test.go`, review and complete the last-resort cases: never reached either status with a current assignee returns it; no assignee anywhere returns `""`; nil changelog with and without a current assignee; empty changelog with and without a current assignee; history containing only other statuses (for example "Closed") returns the current assignee; keep the existing `no_in_review_*` cases consistent with the new rule

### Implementation for User Story 3

- [x] T010 [US3] Run `go test ./internal/jiraservice/...`; if any US3 case fails, fix the final-return order or the nil-changelog path in `resolveImplementer` in `internal/jiraservice/jira.go` (no new code is expected, the paths are written in T006 and T008)

**Checkpoint**: All `TestResolveImplementer` cases pass.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Docs that describe the rule, the constitution gates, and a check against real data.

- [x] T011 [P] Update the `get_sprint_issues` tool description in `internal/mcpserver/server.go` to state how `implementer` is derived (assignee before the first "In Progress", else before the first "In Review.", else current assignee); confirm `internal/mcpserver/server_test.go` does not assert the description text
- [x] T012 [P] Update the "Web Sprint Labels Dashboard" features list in `README.md` to include the Implementer column and its rule
- [x] T013 [P] Add a note under the Status line of `specs/006-implementer-column/spec.md` that the resolution rule is superseded by `specs/010-implementer-resolution/spec.md`
- [x] T014 Run the constitution gates from the repo root: `make fmt`, `make lint`, `make test`, `make build`; all must pass
- [ ] T015 Validate against real data per `specs/010-implementer-resolution/quickstart.md`: rebuild, reconnect the MCP server, call `get_sprint_issues` for sprint 950218 (ST26.20) and confirm that only issues that never had an assignee remain empty (baseline: 15 of 35 empty), then compare at least 10 varied issues with their Jira history. This step needs Jira access and is manual

---

## Dependencies & Execution Order

```text
T001 (baseline)
  ├── T002 [P] constants (jira.go)
  └── T003 [P] test helpers (jira_test.go)
        └── US1: T004, T005 (tests) -> T006 (rewrite)      🎯 MVP
              └── US2: T007 (tests) -> T008 (fallback)
                    └── US3: T009 (tests) -> T010 (verify)
                          └── Polish: T011 [P], T012 [P], T013 [P] -> T014 -> T015
```

- US2 and US3 extend the same function and the same test file as US1, so they follow it in order; they are not independent edits.
- T002 and T003 touch different files and can run together. T011, T012 and T013 touch different files and can run together.
- Within each story the tests come first and should fail (or, for T004, change) before the implementation task.

### Parallel Example

```text
Task T002: add status constants in internal/jiraservice/jira.go
Task T003: add test helpers in internal/jiraservice/jira_test.go

Task T011: update MCP tool description in internal/mcpserver/server.go
Task T012: update dashboard section in README.md
Task T013: add superseded note in specs/006-implementer-column/spec.md
```

---

## Implementation Strategy

1. **MVP (US1)**: T001 to T006. After this the main case (assignee before "In Progress", including the creation-time assignee) is correct and can be merged.
2. **Add US2**: T007 and T008 for issues that skipped "In Progress" or were unassigned at that moment.
3. **Add US3**: T009 and T010 lock in the last-resort behavior.
4. **Polish**: T011 to T015, then use the result on ST26.20 to confirm the empty implementers are gone.

## Notes

- Do not change `internal/sprintview`, `internal/dashboard` or any JSON shape; the new rule reaches both surfaces through `Issue.Implementer`.
- Do not add configuration for the status names (see Complexity Tracking in plan.md).
- Past sprints will show different implementers after the change; that is intended.
