---

description: "Task list for Sprint Label Aggregation Report"
---

# Tasks: Sprint Label Aggregation Report

**Input**: Design documents from `/specs/001-sprint-label-report/`

**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/cli.md ✅

**Tests**: No test tasks — not requested in the feature specification.

**Organization**: Tasks are grouped by user story to enable independent implementation
and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- Entry point: `cmd/get-sprint-label-report/`
- Business logic: `internal/labelreport/`
- Shared changes: `internal/jiraservice/`, `internal/config/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the skeleton for the new command and the new internal package.

- [x] T001 Create `cmd/get-sprint-label-report/main.go` with package declaration and empty `main()` function
- [x] T002 [P] Create `internal/labelreport/aggregator.go` with package declaration stub

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Extend shared data structures and implement the core aggregation logic that
all user stories depend on. No user story work can begin until this phase is complete.

**⚠️ CRITICAL**: Phases 3, 4, and 5 all depend on this phase completing first.

- [x] T003 Add `Labels []string` and `Components []string` fields to the `Issue` struct in `internal/jiraservice/jira.go`
- [x] T004 Populate `Issue.Labels` from `issue.Fields.Labels` and `Issue.Components` from `issue.Fields.Components` (extracting `c.Name` for each non-nil entry) inside `LoadIssuesFromSprint` in `internal/jiraservice/jira.go` (depends on T003)
- [x] T005 [P] Add `ReportLabels []string` to the `Config` struct in `internal/config/config.go`; parse from `REPORT_LABELS` env var (comma-separated); default to `["ai-assisted","ai-assisted-ba","ai-assisted-dev","ai-assisted-qa"]` when the variable is absent or empty
- [x] T006 Define `LabelGroup` struct (fields: `LabelName string`, `Issues []Issue`, `Count int`, `TotalSP float64`) in `internal/labelreport/aggregator.go`
- [x] T007 [P] Define `ComponentReport` struct (fields: `ComponentName string`, `LabelGroups []LabelGroup`, `UnlabeledIssues []Issue`) in `internal/labelreport/aggregator.go`
- [x] T008 Implement `Aggregate(issues []jiraservice.Issue, orderedLabels []string) []ComponentReport` in `internal/labelreport/aggregator.go` (depends on T006, T007): for each distinct component in issues, build one `ComponentReport`; each `LabelGroup` entry contains all issues in that component carrying the label; `UnlabeledIssues` contains issues in that component with no configured label; issues with no component go into a `"No Component"` report; sort reports alphabetically by `ComponentName` with `"No Component"` last

**Checkpoint**: Foundation ready — all user story phases can now proceed.

---

## Phase 3: User Story 1 — Sprint Label Summary Report (Priority: P1) 🎯 MVP

**Goal**: Generate a short-format Word document (or console output) showing one label
aggregation row per configured label per component, plus the full wiring of the command.

**Independent Test**: Run `./bin/get-sprint-label-report -sprint="<sprint>" -debug` and
confirm one label-count-SP summary block per component with correct totals.

### Implementation for User Story 1

- [x] T009 [US1] Implement `renderShortFormatDoc(doc *word.Doc, reports []ComponentReport)` in `cmd/get-sprint-label-report/main.go`: for each `ComponentReport` add a heading with the component name, then a Word table (columns: `Label | Count | Total SP`) with one row per `LabelGroup`; reuse `word.NewTable`, `AddHeaderRow`, `AddDataRow`
- [x] T010 [US1] Implement `printShortFormatConsole(reports []ComponentReport)` in `cmd/get-sprint-label-report/main.go`: print each component's label summary to stdout in plain-text tabular format (component name header, then `label | count | SP` per group)
- [x] T011 [US1] Wire `main()` in `cmd/get-sprint-label-report/main.go`: parse flags (`-sprint` required, `-output`, `-format` default `"short"`, `-debug`); load config; create `jiraservice.JiraService`; call `GetSprintIssues`; call `labelreport.Aggregate`; route to `renderShortFormatDoc` (writes docx) or `printShortFormatConsole` (debug mode) based on `-debug` and `-format` flags; print actionable errors to stderr and exit 1 on failure
- [x] T012 [P] [US1] Add `build-sprint-label-report` target to `Makefile`: `go build -o bin/get-sprint-label-report ./cmd/get-sprint-label-report`
- [x] T013 [P] [US1] Add `run-sprint-label-report` target to `Makefile` with `SPRINT`, `FORMAT` (default `short`), `LOGONLY` variables; also add entry to `make help`

**Checkpoint**: At this point US1 is fully functional and testable independently —
short-format label summary report per component, both doc and console modes.

---

## Phase 4: User Story 2 — Detailed Per-Issue Breakdown (Priority: P2)

**Goal**: Add a full-format rendering mode that shows one row per issue within each label
group, repeating the label name, group count, and total SP on every row.

**Independent Test**: Run `./bin/get-sprint-label-report -sprint="<sprint>" -format=full -debug`
and confirm each label group shows individual issue rows with all six columns.

### Implementation for User Story 2

- [x] T014 [US2] Implement `renderFullFormatDoc(doc *word.Doc, reports []ComponentReport)` in `cmd/get-sprint-label-report/main.go`: for each `ComponentReport` add a component heading, then a Word table (columns: `Label | Count | Total SP | Key | Summary | SP`); for each `LabelGroup` emit one row per issue in the group with label name, group Count, group TotalSP, issue Key, issue Summary, issue StoryPoints repeated; reuse `word.NewTable`, `AddHeaderRow`, `AddDataRow`
- [x] T015 [US2] Implement `printFullFormatConsole(reports []ComponentReport)` in `cmd/get-sprint-label-report/main.go`: print full per-issue breakdown to stdout; each label group header followed by one issue line per issue
- [x] T016 [US2] Wire `-format=full` routing in `main()` in `cmd/get-sprint-label-report/main.go`: when `-format=full` and not `-debug`, call `renderFullFormatDoc`; when `-format=full` and `-debug`, call `printFullFormatConsole`; add validation that `-format` is one of `short` or `full` (exit 1 with error message otherwise)

**Checkpoint**: At this point US1 (short) and US2 (full) both work independently.

---

## Phase 5: User Story 3 — Unlabeled Issues Table (Priority: P2)

**Goal**: Append a separate unlabeled issues table below the label aggregation table in
every component section, for both short and full formats.

**Independent Test**: Run the report against a sprint containing issues with no configured
labels; confirm a separate table with columns `Key | Summary | SP` appears below the label
summary in each component section that has unlabeled issues.

### Implementation for User Story 3

- [x] T017 [US3] Add unlabeled issues Word table to `renderShortFormatDoc` and `renderFullFormatDoc` in `cmd/get-sprint-label-report/main.go`: after the label aggregation table for each `ComponentReport`, if `UnlabeledIssues` is non-empty add a sub-heading "Unlabeled Issues" and a Word table (columns: `Key | Summary | SP`) with one row per unlabeled issue
- [x] T018 [US3] Add unlabeled issues section to `printShortFormatConsole` and `printFullFormatConsole` in `cmd/get-sprint-label-report/main.go`: after the label summary for each component, if unlabeled issues exist print a "Unlabeled Issues:" header followed by one line per issue (`key | summary | SP`)

**Checkpoint**: All three user stories are fully functional. Both formats include the
unlabeled issues table. The complete feature is ready for validation.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Ensure the new binary integrates cleanly with the existing project.

- [x] T019 [P] Verify `make build` compiles all binaries (including `bin/get-sprint-label-report`) without errors or warnings
- [x] T020 [P] Run `make fmt` and `make lint`; fix any issues reported in the new files
- [ ] T021 Run the quickstart.md validation checklist (`specs/001-sprint-label-report/quickstart.md`) end-to-end against a real sprint

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — **BLOCKS all user stories**
- **US1 (Phase 3)**: Depends on Phase 2 completion
- **US2 (Phase 4)**: Depends on Phase 3 completion (reuses renderers and wiring from US1)
- **US3 (Phase 5)**: Can start after Phase 2; touches same renderer functions as US1/US2 so run after Phase 4 to avoid conflicts
- **Polish (Phase 6)**: Depends on all user story phases completing

### Within Each Phase

- Tasks marked [P] within the same phase can run in parallel (they touch different files)
- T003 → T004 (sequential — same file, T004 depends on T003's new fields)
- T006 → T008, T007 → T008 (T008 depends on both struct definitions)
- T009 → T011, T010 → T011 (wiring depends on renderers being defined)

### Parallel Opportunities

```bash
# Phase 1 — run together:
Task T001  # cmd/get-sprint-label-report/main.go stub
Task T002  # internal/labelreport/aggregator.go stub

# Phase 2 — T003/T004 sequential (jira.go); T005 parallel (config.go); T006/T007 parallel (aggregator.go):
Task T003 → T004            # jira.go changes (sequential)
Task T005                   # config.go (parallel with T003/T004)
Task T006 && Task T007      # aggregator.go structs (parallel)
Task T008                   # aggregator.go Aggregate() (after T006, T007)

# Phase 3 — T009 and T010 parallel (same file but different functions), T012 and T013 parallel:
Task T009 && Task T010      # renderer + console functions
Task T011                   # wiring (after T009, T010)
Task T012 && Task T013      # Makefile targets (parallel, after T011)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (**CRITICAL** — blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: `./bin/get-sprint-label-report -sprint="<sprint>" -debug`
5. Demo short-format report with real sprint data

### Incremental Delivery

1. Phase 1 + Phase 2 → Foundation ready
2. Phase 3 → Short-format summary report works (MVP ✅)
3. Phase 4 → Full per-issue breakdown added
4. Phase 5 → Unlabeled issues table added (complete feature ✅)
5. Phase 6 → Clean, lint-pass, validated

---

## Notes

- [P] tasks touch different files — safe to parallelize
- [Story] label maps each task to its user story for traceability
- `internal/labelreport` has zero imports from `jiraservice` or `word` — pass `jiraservice.Issue` slices in; keep it clean
- The `Aggregate()` function is the key algorithmic piece; get it right before rendering
- Commit after each phase checkpoint to preserve working state
- Run `make build` after each phase to catch compile errors early
