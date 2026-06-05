# Data Model: Teams & Board Name Unified Configuration

## New / Changed Types

### `TeamEntry` (new, in `internal/config/config.go`)

| Field | Type | Description |
|-------|------|-------------|
| `ComponentName` | `string` | Jira component name used to filter issues (e.g. `"PROCESSING"`) |
| `BoardName` | `string` | Jira board display name used for sprint lookups (e.g. `"PROCESSING Team"`) |

**Validation**: Both fields must be non-empty. The config parser enforces this by skipping blank entries and using `ComponentName` as `BoardName` when no pipe separator is present.

---

### `Config` (modified, in `internal/config/config.go`)

| Field | Change | Notes |
|-------|--------|-------|
| `BoardName string` | **Removed** | Replaced by per-team `BoardName` in each `TeamEntry` |
| `Teams []string` | **Changed to** `Teams []TeamEntry` | Now carries both component name and board name per entry |
| All other fields | Unchanged | |

---

## Parsing Logic (`internal/config/config.go`)

### Input format

```
TEAMS=PROCESSING|"PROCESSING Team",STABLETEK|"STABLETEK Team"
```

### Parser (`parseTeams(raw string) []TeamEntry`)

```
Input:  raw string (value of TEAMS env var)
Output: []TeamEntry

Algorithm:
  entries = split(raw, ",")
  for each entry:
    entry = trim(entry)
    if empty: skip
    parts = splitN(entry, "|", 2)
    componentName = trim(parts[0])
    if len(parts) == 2:
        boardName = trim(trim(parts[1]), `"`)  // strip whitespace then quotes
    else:
        boardName = componentName              // fallback
    append TeamEntry{ComponentName: componentName, BoardName: boardName}
```

### Removed from required vars list

`JIRA_BOARD_NAME` is removed from the `requiredVars` slice in `config.Load()`.

---

## Caller Changes

### `cmd/get-month-issues-from-jira/main.go`

| Before | After |
|--------|-------|
| `for _, team := range cfg.Teams { ... team ...}` where `team` is `string` | `for _, team := range cfg.Teams { ... team.ComponentName ...}` |
| `teamIssues[team]` | `teamIssues[team.ComponentName]` |
| `logIssuesTable(..., team, ...)` | `logIssuesTable(..., team.ComponentName, ...)` |
| `doc.AddHeading(..., team)` | `doc.AddHeading(..., team.ComponentName)` |

The loop structure is otherwise unchanged; only the string `team` is replaced with `team.ComponentName`.

### `cmd/get-sprint-issues-from-jira/main.go`

| Before | After |
|--------|-------|
| `jiraService.GetSprintIssues(cfg.ProjectKey, cfg.BoardName, ...)` | `jiraService.GetSprintIssues(cfg.ProjectKey, cfg.Teams[0].BoardName, ...)` |
| Error message references `cfg.BoardName` | Error message references `cfg.Teams[0].BoardName` |

### `cmd/get-sprint-label-report/main.go`

The single `GetSprintIssues` + `Aggregate` + render block becomes a loop:

```
hadError := false
for _, team := range cfg.Teams {
    issues, err := jiraService.GetSprintIssues(cfg.ProjectKey, team.BoardName, *sprintName, nil)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: sprint %q on board %q: %v\n", *sprintName, team.BoardName, err)
        hadError = true
        continue
    }
    reports := labelreport.Aggregate(issues, cfg.ReportLabels)
    // render (debug or doc) with team.BoardName as section title context
}
if hadError { os.Exit(1) }
```

The `doc` is created once before the loop and saved after; one document covers all teams.

For debug mode, each team's report section is prefixed with `\n=== Board: <team.BoardName> ===\n`.

### `.env.example`

| Before | After |
|--------|-------|
| `JIRA_BOARD_NAME=TeamBoardName` | *(removed)* |
| `TEAMS=PROCCESING,STABLETEK` | `TEAMS=PROCESSING\|"PROCESSING Team",STABLETEK\|"STABLETEK Team"` |
