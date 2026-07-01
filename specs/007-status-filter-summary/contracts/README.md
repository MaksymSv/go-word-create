# Contracts: Status Filter + Summary Row

**Feature**: 007-status-filter-summary
**Date**: 2026-06-26

## No New API Contracts

This feature introduces **no new API endpoints, request/response schemas, or external interfaces**. All functionality is delivered through the existing dashboard endpoints:

| Existing Endpoint | Method | Change |
|---|---|---|
| `/api/teams` | GET | No change |
| `/api/teams/{component}/sprints` | GET | No change |
| `/api/sprints/{sprintID}/issues` | GET | **No change** — response already includes `DashboardIssue.Status` |
| `/api/issues/{issueKey}/labels` | POST | No change |

The existing `DashboardIssue` response struct already contains the `Status` field used by the status filter bar. No schema modifications are required.

## Existing Contract References

- Full API contract: [specs/005-web-sprint-labels-dashboard/contracts/api.md](../../005-web-sprint-labels-dashboard/contracts/api.md)
- Frontend architecture: [specs/005-web-sprint-labels-dashboard/plan.md](../../005-web-sprint-labels-dashboard/plan.md) (Frontend Architecture section)
