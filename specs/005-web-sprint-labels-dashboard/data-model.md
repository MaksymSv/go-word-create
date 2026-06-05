# Data Model: Web Sprint Labels Dashboard

**Feature**: 005-web-sprint-labels-dashboard
**Date**: 2026-06-05

---

## Entities

### Team (configuration-derived, read-only)

Source: `config.TeamEntry` in `internal/config/config.go`

| Field | Type | Description |
|-------|------|-------------|
| ComponentName | string | Jira component name; used as the team button label |
| BoardName | string | Jira board display name; used for sprint lookups |

**Validation**: Both fields are non-empty strings (enforced by `parseTeams`).
**API representation**: serialized as `{"componentName": "...", "boardName": "..."}`.

---

### Sprint (Jira-derived, read-only)

Source: `jira.Sprint` from go-jira v1.17.0

| Field | Type | Description |
|-------|------|-------------|
| ID | int | Unique sprint identifier (used as route param) |
| Name | string | Human-readable sprint name (e.g., "Sprint 16") |
| State | string | `"active"` \| `"closed"` \| `"future"` |
| StartDate | \*time.Time | Sprint start date; can be nil |
| EndDate | \*time.Time | Sprint end date; can be nil |

**Ordering**: sorted by ID descending (newest first); at most 5 sprints returned per board.
**Filtering**: only `active` and `closed` sprints are returned (future sprints are excluded).

**API representation**:
```json
{"id": 807, "name": "Sprint 16", "state": "active", "startDate": "2026-01-05T09:00:00Z"}
```

---

### DashboardIssue (Jira-derived, partially mutable)

Source: extends `jiraservice.Issue` from `internal/jiraservice/jira.go`

| Field | Type | Description |
|-------|------|-------------|
| Key | string | Jira issue key (e.g., "PROJ-123") |
| Summary | string | Issue title |
| Epic | string | Epic name (resolved from epic link field) |
| StoryPoints | float64 | Story point estimate; 0 if not set |
| Type | string | Issue type name (Bug, Story, Task, Sub-task, Epic, …) |
| Status | string | Current Jira workflow status name |
| URL | string | Full Jira browse URL for the issue |
| Labels | []string | All labels currently on the issue in Jira |
| ActiveLabels | []string | Intersection of Labels and ConfiguredLabels (computed) |

**Mutations**: `Labels` changes when a label button is clicked (via POST/DELETE `/api/issues/{key}/labels`).
**Computed field**: `ActiveLabels` is derived server-side as `intersection(issue.Labels, cfg.ReportLabels)` before sending the response.

**API representation**:
```json
{
  "key": "PROJ-123",
  "summary": "Login fails on mobile",
  "epic": "User Authentication",
  "storyPoints": 3,
  "type": "Bug",
  "status": "In Progress",
  "url": "https://jira.example.com/browse/PROJ-123",
  "activeLabels": ["ai-assisted"]
}
```

---

### SprintIssuesResponse (API response envelope)

| Field | Type | Description |
|-------|------|-------------|
| configuredLabels | []string | Full list from `cfg.ReportLabels` (drives label buttons) |
| issues | []DashboardIssue | Issues in the sprint |

**Rationale**: The frontend needs `configuredLabels` to render all label buttons per row (including inactive ones), not just the labels already assigned.

---

### LabelUpdateRequest (API request body)

| Field | Type | Description |
|-------|------|-------------|
| action | string | `"add"` or `"remove"` |
| label | string | The configured label to add or remove |

**Validation**: `action` must be `"add"` or `"remove"`; `label` must be in `cfg.ReportLabels`.

---

## Relationships

```
Config (TeamEntry[])
  │
  ├── one team ──► one board ──► many Sprints (sorted by ID desc, limit 5)
  │                                     │
  │                                     └──► many DashboardIssues
  │                                                │
  │                                                ├── Labels[] (all Jira labels)
  │                                                └── ActiveLabels[] (intersection with ReportLabels)
  │
  └── ReportLabels[] ──────────────────────────────┘  (drives button set)
```

---

## State Transitions

### Label Button State Machine

```
[inactive] ──click──► [pending] ──API success──► [active]
                              └──API failure──► [inactive] + error toast

[active]   ──click──► [pending] ──API success──► [inactive]
                              └──API failure──► [active] + error toast
```

**Optimistic update**: The button moves to `pending` (disabled) immediately on click while the API call is in flight. On success it flips to the target state. On failure it reverts and shows an error notification.
