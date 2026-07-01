# Tasks: Status Filter + Summary Row

**Input**: Design documents from `specs/007-status-filter-summary/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: Frontend tested manually via browser (no automated tests requested).

## Phase 1: HTML Structure

**Purpose**: Add the status filter bar container to the dashboard HTML shell.

- [ ] T001 [US1] Add `<div id="status-filter-bar">` container above `#content` in `internal/dashboard/web/index.html`

**Checkpoint**: HTML shell has the container element; CSS will style it, JS will populate it.

---

## Phase 2: CSS Styles

**Purpose**: Add status filter bar and summary row styles matching the existing design system (CSS custom properties, dark theme support).

- [ ] T002 [P] Add `#status-filter-bar` container styles in `internal/dashboard/web/style.css`
- [ ] T003 [P] Add `.pill` and `.pill.active` button styles in `internal/dashboard/web/style.css`
- [ ] T004 [P] Add `.summary-row` table footer styles in `internal/dashboard/web/style.css`

**Checkpoint**: All visual elements (filter bar, pills, summary row) have CSS rules. Dark theme styles use existing CSS custom properties (`--color-bg`, `--color-surface`, `--color-text`, `--color-text-muted`, `--color-border`, `--color-btn-team-active-bg`).

---

## Phase 3: User Story 1 - Status Filter Bar (Priority: P1) 🎯 MVP

**Goal**: Dynamic status pill bar above the issues table; clicking pills filters table rows by status in real time (client-side, no network round-trip).

**Independent Test**: A user can load a sprint, click status pills, and verify the table rows update to show only matching issues.

### Implementation for User Story 1

- [ ] T005 [US1] Extend `state` object in `internal/dashboard/web/app.js` with `selectedStatuses: new Set()` and clear it on team/sprint switch
- [ ] T006 [US1] Implement `renderStatusPills(statuses, configuredLabels)` in `internal/dashboard/web/app.js` — extracts unique statuses from loaded issues, sorts alphabetically, creates pill buttons with `data-status` attributes, appends to `#status-filter-bar`
- [ ] T007 [US1] Implement `onStatusPillClick(e)` in `internal/dashboard/web/app.js` — toggles `data-status` in `state.selectedStatuses`, updates pill `.active` class, triggers table re-filter
- [ ] T008 [US1] Modify `renderTable()` in `internal/dashboard/web/app.js` to call `renderStatusPills()` after table creation (only when issues > 0)
- [ ] T009 [US1] Implement `applyFilter()` in `internal/dashboard/web/app.js` — filters the `issues` array by `status` matching `state.selectedStatuses`, re-renders table body rows (hide/show via `tr.hidden` class), re-renders summary row

**Checkpoint**: User Story 1 is fully functional — status pills render, toggle, and filter table rows in real time.

---

## Phase 4: User Story 2 - Summary Row (Priority: P2)

**Goal**: Summary row below the issues table showing per-label count and story-point totals for the filtered view; recalculates on pill toggle.

**Independent Test**: A user can filter to a single status and verify the summary row shows correct counts and story-point totals per label for only those issues.

### Implementation for User Story 2

- [ ] T010 [US2] Implement `renderSummaryRow(filteredIssues, configuredLabels)` in `internal/dashboard/web/app.js` — computes per-label `{ count, totalSP }` from filtered issues, appends `<tfoot><tr class="summary-row">` to the table
- [ ] T011 [US2] Integrate `renderSummaryRow()` into `applyFilter()` so it recalculates whenever pills are toggled
- [ ] T012 [US2] Handle edge case: when `filteredIssues` is empty (all pills selected but no matches), display "No issues match selected statuses" in the summary row instead of the table

**Checkpoint**: User Story 2 is fully functional — summary row shows correct counts and story points for any filter combination.

---

## Phase 5: Polish & Edge Cases

**Purpose**: Handle edge cases, ensure dark theme consistency, verify no regression on existing functionality.

- [ ] T013 Edge case: When sprint has zero issues, do NOT render status filter bar or summary row (existing "No issues found" message shown as before)
- [ ] T014 Edge case: When sprint has only one unique status, render all statuses as pills (not just the matching one); selecting non-matching statuses shows empty state
- [ ] T015 Edge case: Handle issues with null/empty `status` — these match no status pill; shown when no pills selected, hidden when any pill selected
- [ ] T016 Verify dark theme: status filter bar pills and summary row render correctly in dark theme (uses CSS custom properties consistently)
- [ ] T017 Verify no regression: existing label toggle buttons, team/sprint selection, theme toggle all work unchanged

**Checkpoint**: All edge cases handled, dark theme verified, no regression on existing features.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (HTML)**: No dependencies — can start immediately
- **Phase 2 (CSS)**: No dependencies on other phases — can run in parallel with Phase 1 (different files)
- **Phase 3 (US1)**: Depends on Phase 1 (HTML container must exist) and Phase 2 (CSS styles must exist)
- **Phase 4 (US2)**: Depends on Phase 3 (summary row depends on filtering state)
- **Phase 5 (Polish)**: Depends on Phase 4 (edge cases require full feature)

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Phase 1 + Phase 2 — No dependencies on other stories
- **User Story 2 (P2)**: Can start after Phase 3 — depends on US1 filtering state

### Within Each User Story

- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- **Phase 1 + Phase 2** can run in parallel (HTML vs. CSS — different files)
- **T002, T003, T004** can run in parallel (all write to `style.css` but different CSS blocks — merge conflicts resolved by appending, not replacing)

---

## Parallel Example: Setup (Phases 1 + 2)

```bash
# Launch HTML structure task:
Task: "Add status-filter-bar container to index.html"

# Launch CSS styles tasks in parallel:
Task: "Add status-filter-bar container styles to style.css"
Task: "Add pill and pill.active styles to style.css"
Task: "Add summary-row styles to style.css"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: HTML container (T001)
2. Complete Phase 2: CSS styles (T002, T003, T004)
3. Complete Phase 3: US1 implementation (T005–T009)
4. **STOP and VALIDATE**: Test status filter bar independently — load a sprint, click pills, verify rows update
5. Deploy/demo if ready

### Incremental Delivery

1. Phase 1 (HTML) → Container ready
2. Phase 2 (CSS) → Styles ready
3. Phase 3 (US1) → Status filter bar works (MVP!)
4. Phase 4 (US2) → Summary row added
5. Phase 5 (Polish) → Edge cases, dark theme, no regression

### Files Changed (Summary)

| File | Tasks | Change |
|------|-------|--------|
| `internal/dashboard/web/index.html` | T001 | Add `<div id="status-filter-bar">` above `#content` |
| `internal/dashboard/web/style.css` | T002, T003, T004 | Add `.status-filter-bar`, `.pill`, `.pill.active`, `.summary-row` styles |
| `internal/dashboard/web/app.js` | T005–T009, T010–T012, T013–T017 | Add state, renderStatusPills, onStatusPillClick, applyFilter, renderSummaryRow |

### Files NOT Changed

| File | Reason |
|------|--------|
| `internal/dashboard/handler.go` | Already returns `DashboardIssue.Status` on every issue |
| `internal/dashboard/api.go` | Response structs already include `Status` and `ActiveLabels` |
| `internal/jiraservice/jira.go` | No new Jira API calls needed |
| `cmd/web-sprint-labels-report/main.go` | No CLI changes |
| `go.mod` | No new dependencies |
