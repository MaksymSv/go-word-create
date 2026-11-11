# Feature Specification: Project delivery dashboard

**Feature Branch**: none, built on `dev` (retroactive spec written 2026-11-11)

**Status**: Implemented (retroactive; written after the code, from the request and the as-built behavior)

**Input**: "Create project delivery dashboard that includes a few charts. Create web page with two tabs for each TEAM (described in .env). Each tab should contain the charts below (stats for the last 6 sprints, configurable): 1) Velocity: committed vs. completed (SP by sprint), column chart + last-3 average as a line; 2) Commitment rate (% by sprint), column; 3) Backlog health (SP by sprint), column; 4) Lead and cycle time (by sprint), column; 5) Open bugs over time by day, last 30 days, line; 6) Bug dynamics by sprint (count), logged vs. fixed, column; 7) Open bugs over time by priority, by sprint, line. Use a similar UX approach as the AI Assisted web dashboard. Use delivery_central MCP. Analyze how it is better to read the historical data: at runtime, or store some historical data locally."

## User Scenarios & Testing

### User Story 1 - See a team's delivery health on one page (Priority: P1)

A delivery manager opens `/delivery`, sees one tab per team from `TEAMS` (STABLETEK, PROCESSING), and gets all seven charts for the last 6 closed sprints without leaving the page.

**Acceptance Scenarios**:

1. **Given** two configured teams, **When** the page opens, **Then** the first team's tab is active, its charts load, and the other team loads when its tab is clicked; `#TEAM` in the URL selects a tab.
2. **Given** the "Sprints" pills (3, 6, 9, 12 plus the configured default), **When** one is chosen, **Then** every sprint-based chart shows that many closed sprints; the choice is remembered.
3. **Given** any chart, **When** the user hovers a column or point, **Then** a tooltip shows the sprint's full name, each series value and extra context (issue counts, runway, net bug change); **When** they press **Table**, **Then** the same numbers appear as a table.
4. **Given** the light/dark toggle, **Then** the page uses the same theme state as the AI Assisted dashboard and both modes are legible.
5. **Given** a failure computing one chart, **Then** that card shows the team and the Jira error and the other six charts still render.

### User Story 2 - Fast repeat visits and history beyond Jira's window (Priority: P2)

Closed sprints never change, so their expensive results are computed once and reused; the page stays fast, and velocity history keeps growing beyond the 7 sprints Jira's velocity chart returns.

### User Story 3 - The same numbers without a browser (Priority: P3)

`get-delivery-report` prints the report as JSON for one or all teams, so scripts and reviews do not depend on the UI (constitution principle I).

## Requirements

- **FR-001**: One tab per entry of `TEAMS`, labeled with the component name; sprint lookup uses the team's board, issue filters use its component.
- **FR-002**: Charts show the last N **closed** sprints, oldest to newest. N comes from `DELIVERY_SPRINT_COUNT` (default 6), is overridable per request (`?sprints=`, 1 to 24), and from the page.
- **FR-003**: Velocity = committed (story points at sprint start) and completed per sprint from Jira's board velocity chart, plus the average of completed over the last 3 sprints (fewer at the start of history; unknown sprints skipped).
- **FR-004**: Commitment rate = completed / committed, in %, with a 100% reference line.
- **FR-005**: Backlog health = story points of open issues (configured `ISSUE_TYPES`) that were in no sprint when the sprint started, reconstructed from changelogs. Tooltip: issue count, issues without estimate, runway = backlog SP / 3-sprint average velocity.
- **FR-006**: Lead time = created to resolved; cycle time = first move to "In Progress" to resolved; averages in days over issues resolved in the sprint window; cycle time only counts issues that have an In Progress transition.
- **FR-007**: Open bugs by day: bugs of the team's component that are not closed (status category not Done) at the end of each of the last `DELIVERY_BUG_DAYS` (30) days, today up to now.
- **FR-008**: Bug dynamics = bugs created (logged) and bugs resolved (fixed) in each sprint window.
- **FR-009**: Open bugs by priority = open bugs (as in FR-007) per priority at each sprint's end; one stable color per priority name; only priorities that occur are shown, most severe first.
- **FR-017**: An issue is open when its status category is not Done, regardless of its resolution. A Done-category issue is closed at its resolution date, or, when it has none, at the time it entered its current status (changelog). A reopened issue (resolution set, status not Done) is open. Team scope is the board filter (`project` + `component`), so multi-component issues count for each team.
- **FR-010**: A sprint's window is `[max(sprint start, previous sprint close), close)` so overlapping sprints never double-count.
- **FR-011**: Closed-sprint backlog, lead/cycle time and velocity are persisted per team under `DELIVERY_CACHE_DIR`; bug series and velocity-chart reads stay live; built reports are reused in memory for `DELIVERY_CACHE_TTL`; **Refresh** bypasses the in-memory copy only.
- **FR-012**: Missing data is `null`, never zero: sprints whose velocity was never seen render no bar and are skipped by the average.
- **FR-013**: Errors name the team and the failing call; one failing chart never blanks the others.
- **FR-014**: No new dependency; the frontend is vanilla JS with hand-written SVG charts, embedded like the existing dashboard. Both pages link to each other.
- **FR-015**: Every new setting is an environment variable listed in `.env.example`.
- **FR-016**: Every column shows its value above it (zeros included): horizontal when it fits, rotated when bars are narrow (about 9+ sprints). Commitment rate and lead/cycle time are labeled in whole numbers; tooltips and tables keep one decimal.

## Success Criteria

- **SC-001**: A warm page load (all closed sprints cached) completes in under 10 seconds per team; measured about 5 s, dominated by the live bug and velocity reads.
- **SC-002**: A cold first load of the larger team (about 530 issues scanned with changelog) completes in under 60 seconds; measured 28 s. The server warms both teams at startup.
- **SC-003**: Bug Growth numbers agree with Delivery Central's "Bug Growth by Sprints" for PROCESSING (see plan, Validation).
- **SC-004**: `make fmt lint test build` pass.

## Assumptions and known gaps

- Story points and priorities are the current values, not the values at the time (a re-prioritized bug is counted under its current priority in older sprints).
- Backlog = open issues in no sprint. Issues sitting in a future sprint are planned, not backlog.
- Velocity before Jira's 7-sprint window can only be shown for sprints the dashboard has seen while they were inside it.
- Our cycle time is roughly twice Delivery Central's; its exact end-status rule is not published, so we measure to resolution.
- "Logged" bugs differ from Delivery Central by 1 to 2 in two of five sprints (it appears to attribute by component at creation; we use the current component).
- The velocity endpoint (`/rest/greenhopper/1.0/rapid/charts/velocity`) is an undocumented Jira Software API.
