# API Contracts: Web Sprint Labels Dashboard

**Feature**: 005-web-sprint-labels-dashboard
**Date**: 2026-06-05
**Base URL**: `http://localhost:{PORT}` (PORT defaults to 8080)

All API endpoints return `Content-Type: application/json`. Errors return a JSON object `{"error": "message"}`.

---

## GET /

Serves the single-page dashboard HTML. No authentication required.

**Response**: `200 OK` — `text/html` (the embedded `index.html`).

---

## GET /api/teams

Returns the list of configured teams in configuration order.

**Response** `200 OK`:
```json
[
  {"componentName": "PROCESSING", "boardName": "PROCESSING Team"},
  {"componentName": "STABLETEK", "boardName": "STABLETEK Team"}
]
```

**Error cases**: None — derived entirely from in-memory config.

---

## GET /api/teams/{component}/sprints

Returns up to 5 most recent sprints (active + closed) for the board associated with the given component name, sorted newest first.

**Path parameter**: `{component}` — URL-encoded component name (e.g., `PROCESSING`).

**Response** `200 OK`:
```json
[
  {"id": 832, "name": "Sprint 17", "state": "active",  "startDate": "2026-05-19T09:00:00Z", "endDate": "2026-06-02T09:00:00Z"},
  {"id": 807, "name": "Sprint 16", "state": "closed",  "startDate": "2026-05-05T09:00:00Z", "endDate": "2026-05-19T09:00:00Z"},
  {"id": 776, "name": "Sprint 15", "state": "closed",  "startDate": "2026-04-21T09:00:00Z", "endDate": "2026-05-05T09:00:00Z"}
]
```

Dates are ISO 8601 UTC strings; `null` when not set on the sprint.

**Error cases**:
- `404 Not Found` — component name not in config: `{"error": "team 'X' not found"}`
- `502 Bad Gateway` — Jira API unreachable: `{"error": "failed to get sprints: <detail>"}`
- `200 OK` with empty array `[]` — board exists but has no active/closed sprints.

---

## GET /api/sprints/{sprintID}/issues

Returns all issues in the given sprint, together with the full configured label list.

**Path parameter**: `{sprintID}` — integer sprint ID.

**Response** `200 OK`:
```json
{
  "configuredLabels": ["ai-assisted", "ai-assisted-ba", "ai-assisted-dev", "ai-assisted-qa"],
  "issues": [
    {
      "key": "PROJ-123",
      "summary": "Login fails on mobile",
      "epic": "User Authentication",
      "storyPoints": 3,
      "type": "Bug",
      "status": "In Progress",
      "url": "https://jira.example.com/browse/PROJ-123",
      "activeLabels": ["ai-assisted"]
    },
    {
      "key": "PROJ-124",
      "summary": "Add pagination to user list",
      "epic": "User Management",
      "storyPoints": 0,
      "type": "Story",
      "status": "To Do",
      "url": "https://jira.example.com/browse/PROJ-124",
      "activeLabels": []
    }
  ]
}
```

`storyPoints` is `0` when not set. `activeLabels` contains only labels from `configuredLabels` that are currently assigned to the issue in Jira.

**Error cases**:
- `400 Bad Request` — `{sprintID}` is not a valid integer: `{"error": "invalid sprint ID"}`
- `502 Bad Gateway` — Jira API unreachable: `{"error": "failed to load sprint issues: <detail>"}`
- `200 OK` with `{"configuredLabels": [...], "issues": []}` — sprint exists but has no issues.

---

## POST /api/issues/{issueKey}/labels

Adds or removes a single configured label on the given Jira issue.

**Path parameter**: `{issueKey}` — Jira issue key (e.g., `PROJ-123`).

**Request body** (`application/json`):
```json
{"action": "add", "label": "ai-assisted"}
```
or
```json
{"action": "remove", "label": "ai-assisted-dev"}
```

**`action`**: `"add"` or `"remove"` (required).
**`label`**: must be a value from the configured `REPORT_LABELS` list.

**Response** `200 OK` (no body on success).

**Error cases**:
- `400 Bad Request` — missing or invalid `action`/`label` field: `{"error": "invalid request: <detail>"}`
- `400 Bad Request` — label not in configured list: `{"error": "label 'X' is not in the configured label list"}`
- `405 Method Not Allowed` — wrong HTTP method.
- `502 Bad Gateway` — Jira update failed: `{"error": "failed to update label: <detail>"}`

---

## Static Assets

All paths under `/static/` are served from the embedded `web/` directory:
- `/static/style.css` — stylesheet
- `/static/app.js` — application JavaScript
- `/static/icons/*.svg` — issue type SVG icons

The `GET /` route serves `web/index.html` directly (not under `/static/`).
