---

description: "Task list for Label Report Percentage Columns"
---

# Tasks: Label Report Percentage Columns

**Input**: Design documents from `/specs/002-label-report-percentages/`

**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/cli.md ✅

**Tests**: No test tasks — not requested in the feature specification.

**Organization**: Tasks are grouped by user story to enable independent implementation
and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1, US2)
- Include exact file paths in descriptions

## Path Conventions

- Aggregation logic: `internal/labelreport/aggregator.go`
- Entry point / renderers: `cmd/get-sprint-label-report/main.go`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: No new files are required; this feature is a pure extension of two existing
files. Phase 1 serves as a confirmation checkpoint only.

*(No setup tasks — proceed directly to Phase 2.)*

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Extend the `LabelGroup` data model with percentage fields and implement the
`formatPct` helper. Both user story phases depend on these changes.

**⚠️ CRITICAL**: Phases 3 and 4 both depend on this phase completing first.

- [x] T001 Add `CountPct float64` and `TotalSPPct float64` fields to the `LabelGroup` struct in `internal/labelreport/aggregator.go`
- [x] T002 In `Aggregate()` in `internal/labelreport/aggregator.go`, pre-compute `componentTotalCount` (len of all issues in component) and `componentTotalSP` (sum of StoryPoints across all issues in component) before the per-group loop; then populate `groups[i].CountPct` and `groups[i].TotalSPPct` using those totals with explicit zero-guards (depends on T001)
- [x] T003 [P] Add `formatPct(v float64) string` helper to `cmd/get-sprint-label-report/main.go`: if `math.Trunc(v) == v` return `strconv.Itoa(int(v)) + "%"`; otherwise return `strconv.FormatFloat(v, 'f', 1, 64) + "%"` (add `"math"` and `"strconv"` imports as needed; `strconv` is already imported)

**Checkpoint**: Data model extended, percentages computed, formatter ready — US1 and US2 can now proceed.

---

## Phase 3: User Story 1 — Percentage Columns in Short-Format Report (Priority: P1) 🎯 MVP

**Goal**: Update short-format Word and console renderers to show `Count,%` and `Total SP,%`
columns plus a "Total" summary row.

**Independent Test**: Run `./bin/get-sprint-label-report -sprint="<sprint>" -debug` and
confirm the console table shows `Count,%` and `Total SP,%` columns with correct values
and a "Total" row at the bottom of each component block.

### Implementation for User Story 1

- [x] T004 [US1] Update `renderShortFormatDoc` in `cmd/get-sprint-label-report/main.go`: change `AddHeaderRow` to `["Label", "Count", "Count,%", "Total SP", "Total SP,%"]`; update each `AddDataRow` call to include `formatPct(g.CountPct)` and `formatPct(g.TotalSPPct)` after Count and Total SP; after the label-group loop, accumulate `totalCountPct` and `totalSPPct` and append a final `AddDataRow` with `["Total", "", formatPct(totalCountPct), "", formatPct(totalSPPct)]` (depends on T001, T002, T003)
- [x] T005 [US1] Update `printShortFormatConsole` in `cmd/get-sprint-label-report/main.go`: widen the format string to include `Count,%` and `Total SP,%` columns; after the per-group loop print a separator and a "Total" row with accumulated `totalCountPct` and `totalSPPct` values formatted via `formatPct` (depends on T004 for consistency, can run after T004 completes)

**Checkpoint**: Short-format report (both Word doc and console) shows percentage columns and "Total" row. US1 is fully functional and testable independently.

---

## Phase 4: User Story 2 — Percentage Columns in Full-Format Report (Priority: P2)

**Goal**: Update full-format Word and console renderers with the same two percentage
columns and "Total" row.

**Independent Test**: Run `./bin/get-sprint-label-report -sprint="<sprint>" -format=full -debug`
and confirm the console table shows `Count,%` and `Total SP,%` on every issue row plus a
"Total" row at the bottom of each component block.

### Implementation for User Story 2

- [x] T006 [US2] Update `renderFullFormatDoc` in `cmd/get-sprint-label-report/main.go`: change `AddHeaderRow` to 8 columns `["Label", "Count", "Count,%", "Total SP", "Total SP,%", "Key", "Summary", "SP"]`; update each `AddDataRow` call to include `formatPct(g.CountPct)` and `formatPct(g.TotalSPPct)` after Count and Total SP; after the label-group loop append a "Total" row `["Total", "", formatPct(totalCountPct), "", formatPct(totalSPPct), "", "", ""]` (depends on T001, T002, T003)
- [x] T007 [US2] Update `printFullFormatConsole` in `cmd/get-sprint-label-report/main.go`: widen format string to include `Count,%` and `Total SP,%` columns between Count/Total SP and Key; after the per-group loop print a "Total" row with accumulated percentage totals (depends on T006 for consistency)

**Checkpoint**: Both short-format (US1) and full-format (US2) reports include percentage columns and "Total" rows. Feature is complete.

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Ensure the changes integrate cleanly with the existing project.

- [x] T008 [P] Run `make build` and verify all binaries (including `bin/get-sprint-label-report`) compile without errors or warnings
- [x] T009 [P] Run `make fmt` and `make lint`; fix any issues reported in the modified files
- [ ] T010 Run the quickstart.md validation checklist (`specs/002-label-report-percentages/quickstart.md`) end-to-end against a real sprint

---

## Dependencies & Execution Order

### Phase Dependencies

- **Foundational (Phase 2)**: No dependencies — can start immediately; **BLOCKS US1 and US2**
- **US1 (Phase 3)**: Depends on Phase 2 completion (T001, T002, T003)
- **US2 (Phase 4)**: Depends on Phase 2 completion; can start after Phase 3 (same file — sequential to avoid conflicts)
- **Polish (Phase 5)**: Depends on Phase 3 and Phase 4 completion

### Within Each Phase

- T001 → T002 (sequential — same file, T002 depends on T001's new fields)
- T003 can run in parallel with T001 (different file: `main.go` vs `aggregator.go`)
- T004 → T005 (sequential — same file, same component; T005 mirrors T004's pattern)
- T006 → T007 (sequential — same file; T007 mirrors T006's pattern)
- T008 and T009 are parallel with each other (different commands)

### Parallel Opportunities

```bash
# Phase 2 — T001/T002 sequential (aggregator.go); T003 parallel (main.go):
Task T001 → T002    # aggregator.go: add fields, then compute
Task T003           # main.go: formatPct helper (parallel with T001)

# Phase 3 — T004 first (doc renderer), T005 after (console renderer):
Task T004 → T005    # main.go: short-format renderers (sequential, same file)

# Phase 4 — T006 first (doc renderer), T007 after (console renderer):
Task T006 → T007    # main.go: full-format renderers (sequential, same file)

# Phase 5 — T008 and T009 in parallel:
Task T008 && Task T009   # build + lint (different commands)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 2: Foundational (**CRITICAL** — blocks both stories)
2. Complete Phase 3: User Story 1 (short-format percentages)
3. **STOP and VALIDATE**: `./bin/get-sprint-label-report -sprint="<sprint>" -debug`
4. Confirm short-format console shows `Count,%`, `Total SP,%`, and "Total" row

### Incremental Delivery

1. Phase 2 → Data model + formatter ready
2. Phase 3 → Short-format percentage report works (MVP ✅)
3. Phase 4 → Full-format percentage report added (complete feature ✅)
4. Phase 5 → Clean, lint-pass, validated

---

## Notes

- T003 [P] touches `main.go` (a different file from `aggregator.go`) — safe to parallelize with T001
- T004 and T006 both touch `main.go` — run sequentially (Phase 3 before Phase 4) to avoid conflicts
- The "Total" row is assembled in the render functions, not stored in the data model (see research.md)
- `math.Trunc(v) == v` detects whole-number floats for the `formatPct` helper
- Commit after Phase 2 checkpoint to preserve a clean, compiling state
- Run `make build` after each phase to catch compile errors early
