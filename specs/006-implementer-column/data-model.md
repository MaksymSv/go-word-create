# Data Model: Implementer Column

**Branch**: `006-implementer-column` | **Date**: 2026-06-08

## Modified Entities

### jiraservice.Issue (extended)

Existing struct gains one new field:

| Field | Type | Source | Notes |
|---|---|---|---|
| Key | string | Jira | unchanged |
| Summary | string | Jira | unchanged |
| Epic | string | Jira changelog | unchanged |
| StoryPoints | float64 | Jira custom field | unchanged |
| Type | string | Jira | unchanged |
| Status | string | Jira | unchanged |
| URL | string | derived | unchanged |
| Labels | []string | Jira | unchanged |
| Components | []string | Jira | unchanged |
| **Implementer** | **string** | **Jira changelog** | **NEW — display name of assignee at first "In Progress" transition; falls back to current assignee display name; empty string if neither exists** |

### dashboard.DashboardIssue (extended)

Existing API response struct gains one new JSON field:

| Field | JSON key | Type | Notes |
|---|---|---|---|
| Key | `key` | string | unchanged |
| Summary | `summary` | string | unchanged |
| Epic | `epic` | string | unchanged |
| **Implementer** | **`implementer`** | **string** | **NEW — positioned after epic in JSON; empty string serialised as `""`, rendered as "—" by frontend** |
| StoryPoints | `storyPoints` | float64 | unchanged |
| Type | `type` | string | unchanged |
| Status | `status` | string | unchanged |
| URL | `url` | string | unchanged |
| ActiveLabels | `activeLabels` | []string | unchanged |

## New Entities / Value Objects

### ImplementerResolution (algorithm, not a struct)

The resolution logic applied to each `jira.Issue` with changelog:

```
state: currentAssignee = ""
      implementerFound = false

for each history in issue.Changelog.Histories (chronological order):
  for each item in history.Items:
    if item.Field == "assignee":
      currentAssignee = item.ToString          // track latest assignee change

  for each item in history.Items:
    if item.Field == "status" AND
       strings.EqualFold(item.ToString, "in progress") AND
       NOT implementerFound:
         record currentAssignee as implementer
         implementerFound = true
         break outer loop

if NOT implementerFound:
  if issue.Fields.Assignee != nil:
    implementer = issue.Fields.Assignee.DisplayName
  else:
    implementer = ""
```

**Key invariants**:
- Only the **first** "In Progress" transition is used (FR-005)
- Assignee changes in the same history entry as the status change are applied **before** the status check within that entry (loop order: assignee items first, then status check — see implementation note)
- Empty string → rendered as "—" in the frontend

## Unchanged Entities

All other structs (`TeamResponse`, `SprintResponse`, `SprintIssuesResponse`, `LabelUpdateRequest`) are unchanged.
