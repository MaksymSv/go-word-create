# API Contract Changes: Implementer Column

**Branch**: `006-implementer-column` | **Date**: 2026-06-08

This document describes **only the changes** to the existing API surface defined in `specs/005-web-sprint-labels-dashboard/contracts/api.md`.

## Modified Endpoint: GET /api/sprints/{sprintID}/issues

No change to the URL, method, or query parameters.

### Response Body Change

`SprintIssuesResponse.issues[]` — each `DashboardIssue` object gains a new `implementer` field:

**Before**:
```json
{
  "configuredLabels": ["label-a", "label-b"],
  "issues": [
    {
      "key": "PROJ-123",
      "summary": "Fix login bug",
      "epic": "Auth Epic",
      "storyPoints": 3,
      "type": "Bug",
      "status": "Done",
      "url": "https://jira.example.com/browse/PROJ-123",
      "activeLabels": ["label-a"]
    }
  ]
}
```

**After**:
```json
{
  "configuredLabels": ["label-a", "label-b"],
  "issues": [
    {
      "key": "PROJ-123",
      "summary": "Fix login bug",
      "epic": "Auth Epic",
      "implementer": "Jane Smith",
      "storyPoints": 3,
      "type": "Bug",
      "status": "Done",
      "url": "https://jira.example.com/browse/PROJ-123",
      "activeLabels": ["label-a"]
    }
  ]
}
```

**Field specification**:

| Field | Type | Nullable | Description |
|---|---|---|---|
| `implementer` | string | No (empty string `""` when unknown) | Display name of the assignee at the first "In Progress" transition, or current assignee if the issue was never "In Progress", or `""` if neither exists |

**Backward compatibility**: The new field is additive. Existing consumers that ignore unknown fields are unaffected. The frontend `app.js` is updated to render it.

## Unchanged Endpoints

All other endpoints are unchanged:
- `GET /` — serves index.html
- `GET /api/teams`
- `GET /api/teams/{component}/sprints`
- `POST /api/issues/{issueKey}/labels`
