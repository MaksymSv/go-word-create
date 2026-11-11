# Implementation Plan: Project delivery dashboard

**Branch**: none (built on `dev`) | **Date**: 2026-11-11 | **Spec**: [spec.md](spec.md)

## Summary

A new `/delivery` page on the existing dashboard server shows seven charts per team. A new package `internal/deliverystats` computes every metric from Jira (board velocity chart, JQL searches, issue changelogs) and keeps closed-sprint results in a per-team JSON cache. `internal/dashboard` only adds routes and the embedded page; `cmd/get-delivery-report` prints the same report as JSON.

## Data source analysis: runtime or stored locally?

The request said to use the `delivery_central` MCP and to decide between reading history at runtime and storing it. Findings:

| Source | What it offers | Verdict |
|---|---|---|
| `delivery_central` MCP | PROCESSING ("Processing Center Team"): *Lead and Cycle Time* (monthly, not per sprint, fixed period) and *Bug Growth by Sprints*. STABLETEK ("StableTek Team"): no metrics at all. No velocity, commitment, backlog or bug-trend data. | Cannot be the source: covers 2 of 7 charts for 1 of 2 teams, and an MCP tool is not callable from the Go server. **Used as a cross-check only.** |
| Jira board velocity chart | Committed and completed SP per sprint, one call per board, but only the last 7 sprints | Live source for charts 1 and 2; values persisted so history outgrows the 7-sprint window |
| Jira JQL (bugs, no changelog) | created, resolutiondate, priority for every bug of the component (about 330 / 125) | Live; 3 to 4 pages per team is cheap, and recomputing it keeps current-day accuracy |
| Jira issue changelogs | Needed for backlog-at-sprint-start (JQL cannot do `sprint WAS ...`) and first In Progress date | Expensive (28 s for 533 issues) and immutable for closed sprints, so **persist** |

Decision (hybrid):
- **Runtime**: sprint list, velocity chart, bug search (charts 5, 6, 7), assembly. Cheap, and always current.
- **Stored locally** (`DELIVERY_CACHE_DIR/<TEAM>.json`): per closed sprint, backlog health and lead/cycle time; per sprint, committed/completed velocity. A closed sprint's history does not change, so entries are valid forever; they are keyed by sprint ID and checked against the sprint's start and window, and derived values carry a fingerprint of (issue types, completed resolutions, cache version) so a changed definition recomputes them. Velocity is never invalidated because it cannot be rebuilt once Jira drops the sprint from its chart.
- **In memory** (`DELIVERY_CACHE_TTL`, 15 min): the assembled report per team and sprint count, with one lock per team so concurrent tabs share a single Jira scan.
- A newly closed sprint triggers exactly one changelog scan, limited to issues unresolved or resolved since the oldest missing sprint's start.

## Technical Context

**Language/Version**: Go 1.24.0. **Dependencies**: none new (`go-jira` for search and raw requests; vanilla JS, no chart library).

**Storage**: JSON files under `DELIVERY_CACHE_DIR` (default `.cache/delivery`, gitignored), written atomically (temp file + rename). Corrupt or stale files are ignored and rebuilt.

**Testing**: no network. `deliverystats` uses a fake `Source`; `jiraservice` uses an `httptest` Jira; `dashboard` tests the non-Jira routes; `config` tests the new parsers. The frontend has no JS test setup and was verified in headless Chrome (light and dark, 6 and 12 sprints, both teams).

**Performance**: warm page about 5 s, cold larger team about 28 s, `get-delivery-report` for both teams warm: 10 s, no changelog scans.

## Project Structure

```
internal/jiraservice/delivery.go     ClosedSprints (chronological), BoardVelocity, SearchHistoryIssues (+ HistoryIssue, Change)
internal/deliverystats/              types.go, calc.go (pure metric functions), cache.go, service.go, config.go
internal/dashboard/delivery.go       GET /delivery, /api/delivery/settings, /api/delivery/{component}, WarmDelivery
internal/dashboard/web/              delivery.html/.css/.js, charts.js (SVG), style.css (shared page nav)
cmd/get-delivery-report/             JSON report, -team -sprints -refresh
```

## Constitution Check

- **I. CLI-first**: `get-delivery-report` exposes the same logic; the page is the secondary surface. Make target `run-delivery-report`, listed in `make help`. PASS.
- **II. Environment-driven**: five new optional variables, all in `.env.example` with comments (the missing `REPORT_LABELS` and `ISSUE_TYPES` entries were added too). PASS.
- **III. Package separation**: logic in `internal/deliverystats` (no HTTP, no Word); `dashboard` only routes and serves; `cmd/` is wiring; no import cycles. PASS.
- **IV. Document output**: not applicable (no Word output). PASS.
- **V. Observability**: requests are logged by the existing middleware; the scan logs team, issue count, sprint count and duration to stderr; every error names team and call. The report command prints JSON to stdout and is covered by the stdout-only exemption. PASS.
- **Stack**: "Jira Cloud only" in the constitution does not match the on-premise instance already in use (an existing deviation, not introduced here). The velocity call uses an undocumented Jira Software endpoint; see Complexity Tracking.

## Complexity Tracking

| Deviation | Why | Alternative rejected |
|---|---|---|
| Undocumented `/rest/greenhopper/...velocity` endpoint | It is the only cheap source of committed SP per sprint; the documented agile API has no committed value | Rebuilding committed SP from sprint membership and estimate history (several calls per sprint, many edge cases) |
| Hand-written SVG charts (about 330 lines) | The frontend has no bundler and the repo rules forbid a second library for a job without need; no chart library is vendored | Chart.js/ECharts from a CDN (offline/corporate network risk, new dependency) |
| Disk cache | Cold changelog scans take about 30 s per team and velocity history cannot be rebuilt | Runtime-only (slow page, history capped at 7 sprints) |

## Validation against Delivery Central (PROCESSING)

Bug Growth by Sprints, logged / fixed (Delivery Central vs this dashboard):

| Sprint | DC | Ours |
|---|---|---|
| PC26.18 | 5 / 9 | 5 / 9 |
| PC26.19 | 5 / 4 | 5 / 4 |
| PC26.20 | 9 / 8 | 11 / 8 |
| PC26.21 | 16 / 10 | 17 / 10 |
| PC26.22 | 11 / 9 | 11 / 9 |

Fixed matches in all five once non-overlapping windows and "every resolution counts" are used (those two rules were adopted because of this comparison). Lead time (monthly, DC vs ours with every resolution): Jun 23.2 / 23.5, Jul 31.8 / 31.8, Aug 28.9 / 33.1, Sep 31.1 / 30.5, Oct 17.1 / 17.3, Nov 20.0 / 26.1 (Nov is a partial month). Cycle time is about twice DC's and is documented as a known difference.

## As-Built Notes

- **Open vs resolved (found after delivery).** STABLETEK closes many issues without setting a resolution (60 of 328 bugs have status Closed and no resolution). The first version treated "no resolution" as open and showed 75 open bugs where Jira shows 19; backlog health and fixed-bug counts were skewed the same way. Fixed by deciding open/closed from the status category and by reading the close date from the changelog (`internal/deliverystats/settle.go`); close dates for such bugs are looked up once per issue and stored in the cache (`closed` map). Cache version bumped to 2, which recomputes derived values but keeps stored velocity. Today's counts now match Jira (STABLETEK 19, PROCESSING 10), and 8 of 10 past-day spot checks against Jira's `status WAS NOT IN (Closed) ON <date>` are exact (the other two differ by 1).
- Component scope was verified against the board filters (`project = CNPYBCSN AND component = STABLETEK|PROCESSING`) and is identical. No open bug in the project has an empty component; 68 of STABLETEK's 328 bugs also carry another component and count for both teams, as on the Jira board. Other bug-like issue types exist in the instance (Prod Bug, Defect, Bug ST, ...) but none is used in these two components.

- Sprint IDs are not chronological on the STABLETEK board (ST26.16 has a lower ID than ST26.15 but starts later). `ClosedSprints` therefore sorts by start date; ordering by ID produced an empty window for one sprint.
- Jira's velocity chart returns at most 7 sprints regardless of parameters, which is why velocity is persisted.
