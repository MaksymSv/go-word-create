# Data Model: Status Filter + Summary Row

**Feature**: 007-status-filter-summary
**Date**: 2026-06-26

## Backend Data Model

**No changes to existing backend data models.** The existing structs already contain all fields needed:

### `DashboardIssue` (existing, unchanged)

```go
type DashboardIssue struct {
    Key          string   // Jira issue key (e.g., "CNPYBCSN-123")
    Summary      string   // Issue summary text
    Epic         string   // Epic name (may be empty)
    StoryPoints  float64  // Story points (0 if not estimated)
    Type         string   // Issue type (Bug, Story, Task, etc.)
    Status       string   // Current Jira status (used for filtering)
    URL          string   // Jira issue URL
    ActiveLabels []string // Labels matching configured REPORT_LABELS
}
```

### `SprintIssuesResponse` (existing, unchanged)

```go
type SprintIssuesResponse struct {
    ConfiguredLabels []string      // Labels from REPORT_LABELS config
    Issues           []DashboardIssue
}
```

## Frontend State Model (new)

### `StatusFilterState`

A `Set<string>` of status values selected by the user.

- **Type**: JavaScript `Set<string>`
- **Default value**: `new Set()` (empty = show all issues)
- **Lifecycle**: Tied to sprint selection; cleared when user switches sprints or teams.
- **Values**: Derived from `DashboardIssue.Status` values of the loaded sprint.

**Operations**:
- `add(status: string)` — triggered by clicking a status pill
- `delete(status: string)` — triggered by clicking an already-selected pill
- `clear()` — triggered by switching sprints or teams
- `size === 0` — signals "no filter" (show all)

### `FilteredIssueSet`

A JavaScript array derived from `SprintIssuesResponse.Issues` by filtering on `Status`:

```javascript
const filtered = statusFilterState.size === 0
  ? issues
  : issues.filter(issue => statusFilterState.has(issue.status));
```

### `SummaryTotals`

Per-label aggregates computed from `FilteredIssueSet`:

```javascript
// For each label in ConfiguredLabels:
{
  label: string,       // Label name (e.g., "ai-assisted-dev")
  count: number,       // Number of filtered issues with this label
  totalSP: number,     // Sum of storyPoints for filtered issues with this label
}
```

**Validation rules**:
- `count >= 0`
- `totalSP >= 0`
- If no issues have a particular label, `count = 0` and `totalSP = 0`.
- If filtered issues have no story points, `totalSP = 0` (displayed as "—" in the UI).

## Relationships

```
SprintIssuesResponse
  ├── ConfiguredLabels[] → SummaryTotals[] (one per label)
  └── Issues[]
        ├── status → StatusFilterState (filtering)
        └── activeLabels[] → SummaryTotals (label matching)
```

## Validation Rules

1. **Status pill values** are derived exclusively from `issues[].status` values — no hardcoded list.
2. **Summary counts** must match the visible row count for each label (verifiable by manual counting).
3. **Summary story points** must match the sum of `storyPoints` for visible rows with that label (verifiable by manual summation).
4. **Empty sprint** (zero issues): status filter bar and summary row are not rendered (existing "No issues found" message shown instead).
