# Feature Specification: Teams & Board Name Unified Configuration

**Feature Branch**: `004-teams-board-config`

**Created**: 2026-06-05

**Status**: Draft

**Input**: User description: "I want to change configuration (in .env) for teams (TEAMS) to have both the team name and the board name (JIRA_BOARD_NAME) the same configuration parameter. You can use the following format: TEAMS=PROCESSING|"PROCESSING Team",STABLETEK|"STABLETEK Team". So the unused JIRA_BOARD_NAME should be removed. Apply this change in configuration on to all cmd/* files. Additionaly: get-sprint-label-report works with only one team board name now to build report. Change that this to generate reports for all jira boards in TEAM configuration in cycle (one by one)."

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Unified Team & Board Configuration (Priority: P1)

An operator sets up the tool by editing a single `.env` file. Today they must keep two separate variables in sync: `TEAMS` (a list of Jira component names used to filter month issues) and `JIRA_BOARD_NAME` (the Jira board used for sprint lookups). These two always refer to the same team and must be kept consistent manually. The operator wants to express both pieces of information in one place using the format `TEAMS=COMPONENT_NAME|"Board Display Name",...` so there is no duplication and no risk of the two values drifting apart.

**Why this priority**: This is the foundational change that all other stories depend on. Without the new config format parsed correctly, nothing else can work.

**Independent Test**: Set `TEAMS=PROCESSING|"PROCESSING Team"` in `.env`, remove `JIRA_BOARD_NAME`, run any `cmd/` binary — it must start without errors, and the board name `"PROCESSING Team"` must be used for Jira board lookups.

**Acceptance Scenarios**:

1. **Given** `TEAMS=PROCESSING|"PROCESSING Team",STABLETEK|"STABLETEK Team"` and no `JIRA_BOARD_NAME` in `.env`, **When** any command starts, **Then** configuration loads successfully and produces two team entries: `{component: "PROCESSING", boardName: "PROCESSING Team"}` and `{component: "STABLETEK", boardName: "STABLETEK Team"}`.
2. **Given** `JIRA_BOARD_NAME` is absent from `.env`, **When** configuration loads, **Then** no error is produced about the missing variable.
3. **Given** `TEAMS` contains a single entry with no pipe separator (e.g., `TEAMS=PROCESSING`), **When** configuration loads, **Then** the component name is used as the board name as well (backwards-compatible fallback).
4. **Given** `TEAMS` is empty or missing, **When** any command starts, **Then** a clear error message is printed and the process exits with a non-zero code.

---

### User Story 2 — All Commands Use Per-Team Board Names (Priority: P2)

Every `cmd/` binary that currently reads `cfg.BoardName` must switch to reading the board name from the relevant team entry rather than a single global field. The commands affected are `get-sprint-issues-from-jira` and `get-sprint-label-report`.

**Why this priority**: Without this, the new config format is parsed but ignored — the commands would still rely on the now-removed global `BoardName` field.

**Independent Test**: Set `TEAMS=PROCESSING|"PROCESSING Team"`, run `get-sprint-issues-from-jira -sprint "Sprint 1"` — the board `"PROCESSING Team"` must be used for the Jira query (visible in debug/log output).

**Acceptance Scenarios**:

1. **Given** a single team entry `PROCESSING|"PROCESSING Team"`, **When** `get-sprint-issues-from-jira` is run, **Then** it queries Jira using board name `"PROCESSING Team"`.
2. **Given** `cfg.BoardName` field is removed from the config struct, **When** the project is built, **Then** zero compilation errors occur.
3. **Given** a command that previously used `cfg.BoardName` directly, **When** the code is updated, **Then** it reads the board name from the first or specified team entry instead.

---

### User Story 3 — Sprint Label Report Loops Over All Teams (Priority: P3)

`get-sprint-label-report` currently generates one report for the single configured board. After this change it must iterate over every team defined in `TEAMS`, fetch sprint issues for each board in sequence, and produce a combined report document covering all teams — one section per team.

**Why this priority**: This is an independent capability enhancement on top of US1 and US2. It delivers immediate value for users who manage multiple teams.

**Independent Test**: Set `TEAMS=PROCESSING|"PROCESSING Team",STABLETEK|"STABLETEK Team"`, run `get-sprint-label-report -sprint "Sprint 42"` — the output document must contain separate label-report sections for both `"PROCESSING Team"` and `"STABLETEK Team"`.

**Acceptance Scenarios**:

1. **Given** two teams in `TEAMS`, **When** `get-sprint-label-report` is run for a sprint, **Then** it fetches issues for each board in order and includes both teams' data in the output.
2. **Given** a team's board has no sprint matching the given sprint name, **When** processing that team, **Then** an error is printed for that team, the command continues with remaining teams, and a non-zero exit code is returned at the end.
3. **Given** only one team in `TEAMS`, **When** `get-sprint-label-report` is run, **Then** behaviour is identical to the current single-board behaviour.
4. **Given** the `-debug` flag is set, **When** multiple teams are configured, **Then** each team's report section is printed to stdout in sequence with a visible team header.

---

### Edge Cases

- What happens when a team entry in `TEAMS` uses a pipe character inside the board name (e.g., `PROC|"Board|Name"`)? — Quoted values containing pipes must be handled correctly.
- What happens when `TEAMS` has trailing commas or extra whitespace (`TEAMS= PROC | "Board" , `)?  — Whitespace around separators must be trimmed and empty entries skipped.
- What happens when the same sprint name does not exist on one team's board but does exist on another's? — Per-team error, continue.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The `TEAMS` environment variable MUST accept entries in the format `COMPONENT_NAME|"Board Display Name"`, comma-separated for multiple teams.
- **FR-002**: The `JIRA_BOARD_NAME` environment variable MUST be removed from the required variables list and from validation; its presence MUST be silently ignored (no error).
- **FR-003**: Configuration parsing MUST produce a list of team entries, each carrying a Jira component name and a Jira board name.
- **FR-004**: When no pipe separator is present in a `TEAMS` entry, the component name MUST be used as both the component filter and the board name (backwards-compatible fallback).
- **FR-005**: The global single `BoardName` field MUST be removed from the configuration struct; all consumers MUST be updated to use per-team board names.
- **FR-006**: `get-sprint-issues-from-jira` MUST use the board name from the (single or first) team entry when performing sprint queries.
- **FR-007**: `get-sprint-label-report` MUST iterate over all team entries, fetching sprint issues for each board in sequence, and produce a combined output with one report section per team.
- **FR-008**: If fetching issues fails for one team in `get-sprint-label-report`, an error MUST be logged for that team, processing MUST continue for remaining teams, and the command MUST exit with a non-zero code when at least one team failed.
- **FR-009**: The `.env.example` file MUST be updated to show the new `TEAMS` format and remove the `JIRA_BOARD_NAME` example entry.

### Key Entities

- **TeamEntry**: Represents one configured team. Attributes: `ComponentName` (string, used to filter Jira issues by component), `BoardName` (string, used to look up the Jira board for sprint queries).
- **Config**: Application-level configuration struct. The `Teams []TeamEntry` field replaces the old `Teams []string` + `BoardName string` combination.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An operator can configure any number of teams using a single `TEAMS` variable — zero additional variables required per team.
- **SC-002**: Removing `JIRA_BOARD_NAME` from `.env` produces no startup error; the old variable is fully superseded.
- **SC-003**: `get-sprint-label-report` produces a report covering all configured teams in a single run — no need to run the command once per team.
- **SC-004**: The project builds without errors after the change (`make build` passes with zero warnings).
- **SC-005**: An operator adding a new team to `TEAMS` sees it automatically included in sprint label reports without changing any other configuration.

## Assumptions

- The existing `TEAMS` variable is currently only used by `get-month-issues-from-jira` as a Jira component filter; `get-sprint-issues-from-jira` and `get-sprint-label-report` use `JIRA_BOARD_NAME` independently. This feature merges both into the unified `TEAMS` entries.
- `get-sprint-issues-from-jira` is expected to work with one team at a time (the caller selects the relevant board); only `get-sprint-label-report` changes to loop over all teams.
- The board display name (after the pipe) may contain spaces and is passed verbatim to the Jira board lookup API.
- Users running `get-sprint-issues-from-jira` with multiple teams will continue to invoke it once per board; multi-team looping is out of scope for that command in this feature.
- `.env` file syntax does not support multi-line values; all teams must fit on one line.
