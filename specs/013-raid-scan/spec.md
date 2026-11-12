# Feature Specification: Risk / RAID scan

**Feature Branch**: none, built on `dev` (retroactive spec written 2026-11-11)

**Status**: Implemented (retroactive; written after the code, from the request and the as-built behavior)

**Input**: "Add another section to the UI dashboard: Risk / RAID scan. Automated scans of Jira for blocked, aging or at-risk items and scope drift. Think how and what might be gathered and how to visualize."

## Design decisions

- **What is gathered.** One query per team reads every open issue (`project` + `component`, `ISSUE_TYPES`, status category not Done) with its changelog; plus the keys of the running sprint's issues and of issues flagged as impediments. Scope comes from Jira's sprint report for the last N closed sprints and the running one.
- **RAID mapping.** Jira has no RAID register, so signals are grouped: Risks (aging, stale, late start, rolled over, overdue), Assumptions (no estimate, no assignee), Issues (blocked, critical bug), Dependencies (open "is blocked by" / "depends on" link).
- **Scope drift source.** The undocumented `rapid/charts/sprintreport` endpoint. Checked on 8 closed sprints of both teams: summing `estimateStatistic` over issues not listed in `issueKeysAddedDuringSprint` equals the velocity chart's committed value exactly, and `currentEstimateStatistic` equals the report's current estimate sums. Reading the sprint's changelog instead would miss issues created straight into a sprint.
- **No history in Jira.** Jira cannot say what was blocked last week, so a daily snapshot (flagged, blocked, aging, open) is stored and the trend chart fills in from the second day.
- **Visualization.** Status tiles (clickable filters), a sprint pulse (progress vs. clock), a ranked watchlist with signal chips, and four charts: work in progress by age, scope added/removed per sprint, scope drift per sprint against a limit, and the daily trend. Color is never the only carrier: tiles state "Needs action / Watch / Clear" and chips carry a severity mark.

## User Scenarios & Testing

### User Story 1 - See what needs attention in a team (Priority: P1)

A delivery manager opens `/raid`, picks a team tab and sees which open issues are blocked, aging or at risk, and how much the running sprint's scope has moved.

**Acceptance Scenarios**:

1. **Given** a team with blocked and aging work, **When** the page loads, **Then** the tiles show counts per RAID category, the watchlist lists issues most severe first, and every chip explains its reason on hover.
2. **Given** a tile or a signal filter, **When** it is clicked, **Then** the watchlist narrows to it; clicking again clears it.
3. **Given** a failure reading one part of Jira (the active sprint, a sprint report, the flagged lookup), **Then** only that card or a warning line reports it, naming the team, and everything else still renders.
4. **Given** the first day of use, **Then** the trend card explains that the history starts today instead of drawing an empty chart.

### User Story 2 - The same data without a browser (Priority: P2)

`get-raid-report` prints the report as JSON (`-team`, `-sprints`, `-refresh`), errors and warnings on stderr.

## Requirements

- **FR-001**: One tab per entry of `TEAMS`; issues are filtered by component, sprints resolved by board.
- **FR-002**: Blocked = status in `RAID_BLOCKED_STATUSES`, label in `RAID_BLOCKED_LABELS`, or flagged as an impediment; severity high. The duration comes from the changelog when known.
- **FR-003**: Aging = status category In Progress for `RAID_AGING_DAYS` or more, counted from the first move to "In Progress" (else the first status change, else creation); high at twice the limit. Stale = in progress and not updated for `RAID_STALE_DAYS`.
- **FR-004**: Late start = status category To Do in the running sprint once `RAID_LATE_START_PCT` of it has elapsed. Rolled over = in a sprint or in progress, and part of `RAID_ROLLOVER_SPRINTS` or more sprints according to the Sprint field history. Overdue = due date before today; severity high.
- **FR-005**: Critical bug = type Bug with a priority in `RAID_CRITICAL_PRIORITIES`. Dependency = a link whose relation contains "blocked by" or "depends on" to an issue whose status category is not Done.
- **FR-006**: No estimate (non-bugs in the running sprint) and no assignee (in progress) are Assumptions with severity low.
- **FR-007**: Items are ranked by their most severe signal, then summed score (high 3, medium 2, low 1), then age, then key.
- **FR-008**: Scope per sprint: Committed (scope at start), Added and Removed (after start), Reestimated (estimate change of issues that stayed), Final = Committed - Removed + Added + Reestimated, Completed, CarriedOver, DriftPct = (Final - Committed) / Committed. Closed-sprint values are stored in `DELIVERY_CACHE_DIR/raid-<team>.json` and never re-read; the running sprint is always live.
- **FR-009**: Drift is amber from half of `RAID_SCOPE_DRIFT_PCT` and red from the limit.
- **FR-010**: One trend snapshot per day (the last build of the day wins), at most 120 days, stored with the scopes; a scope-meaning change drops the scopes but keeps the trend.
- **FR-011**: Built reports are reused in memory for `DELIVERY_CACHE_TTL`; **Refresh** bypasses that copy only. The server warms all teams at startup.
- **FR-012**: Every setting is an environment variable with a default, listed in `.env.example`. No new dependency; the frontend is vanilla JS and reuses `charts.js` (extended with per-bar colors and negative bars) and the new shared `common.js`, which `delivery.js` now also uses.
- **FR-014**: A view switch limits the tiles, the signal-filter counts and the watchlist to the running sprint's issues. The report carries a second summary (`sprintSummary`, same shape as `summary`, counted over the sprint's issues only) so the numbers come from the scan, not from the page; it is `null` when no sprint is running or its issues could not be listed, which disables the sprint option instead of showing zeros. The sprint pulse, scope charts and work-in-progress chart are not affected by the switch.
- **FR-013**: Board lookups by name are cached in `JiraService` (board IDs do not change); bursts of them were answered with 403 by Jira.

## Success Criteria

- **SC-001**: A cold load of a team (about 170 open issues with changelog plus 8 sprint reports) completes in under 30 seconds; measured about 10 s. Warm loads are served from memory.
- **SC-002**: For both teams, Committed equals the velocity chart's committed value in every closed sprint (checked 2026-11-11).
- **SC-003**: `make fmt lint test build` pass.

## Assumptions and known gaps

- Verified only against the two configured teams of one Jira Server/Data Center instance. `flagged is not EMPTY` and the `Flagged` changelog field are assumed standard; if the JQL is rejected the scan continues with a warning and without that signal.
- Rolled over undercounts by one sprint for an issue created straight into a sprint and never moved, since the changelog never mentions that first sprint.
- Story points and priorities are the current values. "Aging" uses time since first "In Progress", so an issue that went back and forth keeps its original start.
- Scope drift counts estimating an unestimated issue after the sprint started as re-estimation, which is why it can be high for teams that estimate during the sprint.
- Not exposed through the MCP server (delivery stats are not either).
