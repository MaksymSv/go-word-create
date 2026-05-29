# Data Model: Sprint Label Aggregation Report

**Feature**: Sprint Label Aggregation Report
**Branch**: `001-sprint-label-report`
**Date**: 2026-05-29

## Modified Entities

### `jiraservice.Issue` (extended)

Existing struct in `internal/jiraservice/jira.go`. Two new fields are added:

```
Issue
├── Key          string       — Jira issue key (e.g. PROJ-123)          [existing]
├── Summary      string       — Issue title                              [existing]
├── Epic         string       — Resolved epic name                       [existing]
├── StoryPoints  float64      — Story point estimate (0 if unestimated)  [existing]
├── Type         string       — Issue type (Bug, Feature, Task, …)       [existing]
├── Status       string       — Current workflow status                  [existing]
├── URL          string       — Full Jira browse URL                     [existing]
├── Labels       []string     — Jira labels assigned to the issue        [NEW]
└── Components   []string     — Jira component names the issue belongs to[NEW]
```

**Population**:
- `Labels` ← `issue.Fields.Labels` (standard go-jira field, no custom field ID needed)
- `Components` ← `[c.Name for c in issue.Fields.Components if c != nil]`

**Change scope**: `LoadIssuesFromSprint` in `internal/jiraservice/jira.go`. No other
existing methods need modification (they discard or pass through the `Issue` slice).

---

### `config.Config` (extended)

Existing struct in `internal/config/config.go`. One new field:

```
Config
├── JiraURL             string    [existing]
├── JiraUsername        string    [existing]
├── JiraAPIToken        string    [existing]
├── BoardName           string    [existing]
├── ProjectKey          string    [existing]
├── OutputFile          string    [existing]
├── JiraEpicField       string    [existing]
├── JiraSPField         string    [existing]
├── JiraComponentField  string    [existing]
├── Teams               []string  [existing]
└── ReportLabels        []string  [NEW] — ordered label list for aggregation
```

**Population**: Parse `REPORT_LABELS` env var (comma-separated). If absent or empty,
default to `["ai-assisted", "ai-assisted-ba", "ai-assisted-dev", "ai-assisted-qa"]`.
`REPORT_LABELS` is optional — no startup failure if not set.

---

## New Entities

### `labelreport.LabelGroup`

Lives in `internal/labelreport/aggregator.go`.

```
LabelGroup
├── LabelName  string          — Configured label string (e.g. "ai-assisted")
├── Issues     []jiraservice.Issue — Issues in this sprint that carry this label
├── Count      int             — len(Issues); pre-computed for rendering convenience
└── TotalSP    float64         — Sum of StoryPoints across Issues
```

**Invariants**:
- `LabelGroup` entries in a `[]LabelGroup` slice MUST appear in the same order as the
  configured label list.
- An issue may appear in multiple `LabelGroup` entries (non-exclusive membership).
- `Count == len(Issues)` and `TotalSP == sum(issue.StoryPoints for issue in Issues)`.

---

### `labelreport.ComponentReport`

Groups a component's label groups and unlabeled issues together.

```
ComponentReport
├── ComponentName    string            — Jira component name; "No Component" if issue has none
├── LabelGroups      []LabelGroup      — One entry per configured label, in configured order
└── UnlabeledIssues  []jiraservice.Issue — Issues with no configured label in this component
```

**Invariants**:
- Every issue in the sprint MUST appear in exactly one `ComponentReport` per component
  it belongs to (an issue in two components appears in two reports).
- An issue with no component appears in the `ComponentReport` with `ComponentName == "No Component"`.
- Within a `ComponentReport`, an issue appears in `LabelGroups` entries for each
  configured label it carries, AND in `UnlabeledIssues` only if it carries no configured label.

---

## Aggregation Function Signature

```
// Aggregate groups a flat issue slice into per-component label reports.
// orderedLabels defines both which labels to track and their display order.
func Aggregate(issues []Issue, orderedLabels []string) []ComponentReport
```

- Returns one `ComponentReport` per distinct component found in `issues`, sorted
  alphabetically by `ComponentName` (with "No Component" last).
- Each `ComponentReport.LabelGroups` has exactly `len(orderedLabels)` entries (one per
  label, in order), even if some have zero issues.

---

## State Transitions

Not applicable — this feature is a read-only reporting tool. No persistent state is
modified.
