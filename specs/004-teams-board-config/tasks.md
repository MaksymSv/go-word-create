# Tasks: Teams & Board Name Unified Configuration

**Input**: Design documents from `specs/004-teams-board-config/`

**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅

**Organization**: Tasks grouped by user story. All user story work is blocked by the single foundational config change (T001).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US3)

---

## Phase 1: Setup

No project initialization required — this feature modifies existing files only.

---

## Phase 2: Foundational (Blocking Prerequisite)

**Purpose**: Introduce `TeamEntry` struct, `parseTeams()` helper, and update `Config` in `internal/config/config.go`. All user story phases depend on this.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T001 Update `internal/config/config.go`: add `TeamEntry` struct with `ComponentName string` and `BoardName string` fields; add unexported `parseTeams(raw string) []TeamEntry` helper that splits on `,` (entries) then on the first `|` (component vs. board), trims whitespace, strips surrounding double-quotes from board name, and falls back to `componentName` as `boardName` when no pipe is present; change `Config.Teams` from `[]string` to `[]TeamEntry`; remove `Config.BoardName string` field; remove `"JIRA_BOARD_NAME"` from the `requiredVars` slice; replace the manual `strings.Split(rawTeams, ",")` loop in `Load()` with a call to `parseTeams(os.Getenv("TEAMS"))`; remove `BoardName: os.Getenv("JIRA_BOARD_NAME")` from the returned config literal

**Checkpoint**: `Config.Teams` is `[]TeamEntry`; `Config.BoardName` does not exist; `JIRA_BOARD_NAME` is no longer required. All downstream `cmd/` updates can now begin.

---

## Phase 3: User Story 1 — Update Month-Issues Command (Priority: P1) 🎯 MVP

**Goal**: `get-month-issues-from-jira` uses `team.ComponentName` instead of a bare `string` from `cfg.Teams`.

**Independent Test**: Run `make build`; set `TEAMS=PROCESSING|"PROCESSING Team"` in `.env`; run `get-month-issues-from-jira -month 2026.05 -debug` — output must show `"PROCESSING"` as the component name and no reference to `JIRA_BOARD_NAME`.

### Implementation for User Story 1

- [x] T002 [US1] Update `cmd/get-month-issues-from-jira/main.go`: change the `for _, team := range cfg.Teams` loop so that `team` is now `config.TeamEntry`; replace all uses of `team` (string) with `team.ComponentName`; replace `teamIssues[team]` map key with `teamIssues[team.ComponentName]`; update the `teamIssues` map type from `map[string]TeamIssues` to `map[string]TeamIssues` (key remains `string`, value of the key changes to `team.ComponentName`)

**Checkpoint**: `get-month-issues-from-jira` compiles and runs; component filter uses `team.ComponentName`.

---

## Phase 4: User Story 2 — Update Sprint-Issues Command (Priority: P2)

**Goal**: `get-sprint-issues-from-jira` reads the board name from `cfg.Teams[0].BoardName` instead of the now-removed `cfg.BoardName`.

**Independent Test**: Run `make build`; set `TEAMS=PROCESSING|"PROCESSING Team"`; run `get-sprint-issues-from-jira -sprint "Sprint 1" -debug` — output or error message must reference board `"PROCESSING Team"`.

### Implementation for User Story 2

- [x] T003 [P] [US2] Update `cmd/get-sprint-issues-from-jira/main.go`: replace `cfg.BoardName` with `cfg.Teams[0].BoardName` in the `jiraService.GetSprintIssues(...)` call and in any log/error messages that reference the board name

**Checkpoint**: `get-sprint-issues-from-jira` compiles and uses the first team's board name.

---

## Phase 5: User Story 3 — Sprint Label Report Loops Over All Teams (Priority: P3)

**Goal**: `get-sprint-label-report` iterates over all `cfg.Teams` entries, fetches sprint issues per board, and produces a combined document with one section per team. A per-team failure is logged and skipped; exit code 1 if any team fails.

**Independent Test**: Set `TEAMS=PROCESSING|"PROCESSING Team",STABLETEK|"STABLETEK Team"`; run `get-sprint-label-report -sprint "Sprint 42" -debug` — output must contain separate labeled sections for both boards. If one board name is wrong, that team's error is printed and the other team's section still appears.

### Implementation for User Story 3

- [x] T004 [P] [US3] Update `cmd/get-sprint-label-report/main.go`: replace the single `issues, err := jiraService.GetSprintIssues(cfg.ProjectKey, cfg.BoardName, ...)` + `Aggregate` + render block with a loop over `cfg.Teams`; inside the loop: fetch issues for `team.BoardName`, on error print to stderr and set `hadError = true` then `continue`; aggregate and render (debug or doc) for the current team; create `doc` once before the loop; in debug mode prefix each team's output with `\n=== Board: <team.BoardName> ===\n`; after the loop, save the document if not in debug mode; exit 1 if `hadError` is true

**Checkpoint**: All three user stories are independently functional. `get-sprint-label-report` produces one section per configured team.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [x] T005 [P] Update `.env.example`: remove the `JIRA_BOARD_NAME=TeamBoardName` line; update the `TEAMS` example to `TEAMS=PROCESSING|"PROCESSING Team",STABLETEK|"STABLETEK Team"` with an explanatory comment
- [x] T006 Run `make build` and confirm zero compilation errors across all binaries

---

## Dependencies & Execution Order

### Phase Dependencies

- **Foundational (Phase 2)**: No dependencies — start immediately
- **US1 (Phase 3)**: Depends on T001
- **US2 (Phase 4)**: Depends on T001; parallel with US1 (different file)
- **US3 (Phase 5)**: Depends on T001; parallel with US1 and US2 (different file)
- **Polish (Phase 6)**: T005 parallel with US phases; T006 depends on T002–T005

### Within US1, US2, US3

Each user story is a single task touching a single file — no intra-story ordering needed.

### Parallel Opportunities

```
T001 (foundational)
  ↓
  ├── T002 [US1] get-month-issues-from-jira/main.go
  ├── T003 [US2] get-sprint-issues-from-jira/main.go  (parallel with T002)
  ├── T004 [US3] get-sprint-label-report/main.go       (parallel with T002, T003)
  └── T005 [P]   .env.example                          (parallel with T002–T004)
        ↓
      T006 make build
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete T001 (Foundational)
2. Complete T002 (US1 — month-issues caller)
3. **STOP and VALIDATE**: `make build` passes; `get-month-issues-from-jira -debug` works with new TEAMS format
4. Ship if acceptable

### Incremental Delivery

1. T001 → foundation ready
2. T002 → US1 done (month issues uses component names from TeamEntry)
3. T003 → US2 done (sprint issues uses per-team board name)
4. T004 → US3 done (label report loops all teams)
5. T005–T006 → polish + build verification

---

## Notes

- T002, T003, T004 touch different files and can run in parallel after T001
- `cfg.Teams[0]` in T003 is safe because `config.Load()` already errors when `TEAMS` is empty
- T004 changes the control flow of `main()` substantially — read `data-model.md` for the exact loop structure before implementing
- `.env.example` update (T005) has no code dependency and can be done at any point after T001
