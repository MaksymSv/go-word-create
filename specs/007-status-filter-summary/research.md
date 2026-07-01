# Research: Status Filter + Summary Row

**Feature**: 007-status-filter-summary
**Date**: 2026-06-26

This feature has **zero unresolved technical unknowns**. All design decisions are documented in the plan. The following records the decisions made during planning.

## Decisions

### Decision 1: Client-Side Filtering (No New API Endpoint)

**Decision**: Status filtering is performed entirely in the browser using JavaScript. No new backend endpoint is created.

**Rationale**: The existing `GET /api/sprints/{sprintID}/issues` endpoint already returns every issue with its `status` field (as `DashboardIssue.Status string`). Filtering 200 issues client-side is trivial — JavaScript can filter an array of ~200 objects in well under 1ms. Creating a new API endpoint would add backend complexity (new route, new handler, new tests) for no performance benefit.

**Alternatives considered**:
- Backend filtering via a `?statuses=` query parameter: would require changes to `handler.go`, `jira.go`, and all response structs. Adds ~50 lines of Go code and new test cases. Rejected because the data is already available client-side.

### Decision 2: Dynamic Status Pill Discovery

**Decision**: Status pills are dynamically discovered from the `issues[].status` values of the currently loaded sprint. No hardcoded status list is used.

**Rationale**: Different teams use different Jira workflows — one team may use "In Review.", another "Done", another "QA Testing". A hardcoded list would be brittle and require configuration changes for every new team. Dynamic discovery adapts automatically.

**Alternatives considered**:
- Hardcoded status list from `.env` (e.g., `DASHBOARD_STATUSES`): rejected because it would require a new environment variable, contradicting Constitution Principle II (Environment-Driven Configuration) by adding config surface for no benefit.

### Decision 3: Client-Side Summary Aggregation

**Decision**: The summary row is computed in JavaScript using the same logic as `labelreport.Aggregate()` but without calling the backend. For each configured label, count issues whose `activeLabels` include that label and sum their `storyPoints`.

**Rationale**: The `labelreport.Aggregate()` function already computes per-label `Count`, `TotalSP`, `CountPct`, and `TotalSPPct`. The frontend needs `Count` and `TotalSP` only — a simpler subset. Computing this client-side avoids a new API endpoint and keeps the backend unchanged.

**Alternatives considered**:
- New `GET /api/sprints/{sprintID}/summary` endpoint: would return pre-computed summary totals. Rejected because (a) the data is already in the response, (b) it would duplicate the aggregation logic in two places (backend + frontend), violating the single-source-of-truth principle.

### Decision 4: No Backend Changes

**Decision**: `internal/dashboard/handler.go`, `internal/dashboard/api.go`, `internal/jiraservice/jira.go`, and all Go structs remain unchanged.

**Rationale**: The existing `DashboardIssue` struct already includes `Status string` and `ActiveLabels []string`. The existing `SprintIssuesResponse` already includes `ConfiguredLabels []string`. No data transformation is needed on the backend.

---

## Constitution Alignment

| Principle | Status |
|-----------|--------|
| I. CLI-First | ✅ Unaffected — no CLI binaries changed |
| II. Environment-Driven | ✅ No new env vars |
| III. Package Separation | ✅ No new packages; existing packages unchanged |
| IV. Consistent Document Output | ✅ N/A — no documents generated |
| V. Observability | ✅ No new endpoints; existing logging covers unchanged routes |
