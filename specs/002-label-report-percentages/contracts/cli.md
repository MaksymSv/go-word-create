# CLI Contract: Label Report Percentage Columns

**Feature**: Label Report Percentage Columns
**Binary**: `bin/get-sprint-label-report`
**Date**: 2026-06-02

## Flags (unchanged from Feature 001)

| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `-sprint` | string | Yes | — | Sprint name (exact Jira match) |
| `-output` | string | No | `sprint-label-report.docx` | Output file path |
| `-format` | string | No | `short` | `short` or `full` |
| `-debug` | bool | No | `false` | Print to stdout instead of writing a file |

No new flags are added by this feature.

## Environment Variables (unchanged from Feature 001)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JIRA_URL` | Yes | — | Jira Cloud base URL |
| `JIRA_USERNAME` | Yes | — | Jira account email |
| `JIRA_API_TOKEN` | Yes | — | Jira API token |
| `JIRA_PROJECT_KEY` | Yes | — | Jira project key |
| `JIRA_BOARD_NAME` | Yes | — | Agile board name |
| `JIRA_EPIC_FIELD` | Yes | — | Custom field ID for Epic Link |
| `JIRA_SP_FIELD` | Yes | — | Custom field ID for Story Points |
| `REPORT_LABELS` | No | `ai-assisted,ai-assisted-ba,ai-assisted-dev,ai-assisted-qa` | Ordered comma-separated label list |

## Output Schema — Short Format (updated)

Word document table per component. **Old schema → New schema**:

```
OLD:  Label | Count | Total SP
NEW:  Label | Count | Count,% | Total SP | Total SP,%
```

Plus a "Total" row appended at the end of the label table:

```
Row:  "Total" | (empty) | <sum of Count,%> | (empty) | <sum of Total SP,%>
```

### Short-format console (debug) example

```
Component: Team Alpha
  Label                               | Count | Count,%  | Total SP | Total SP,%
  -----------------------------------------------------------------------
  ai-assisted                         |     4 |    40%   |     10.0 |     50%
  ai-assisted-ba                      |     2 |    20%   |      4.0 |     20%
  ai-assisted-dev                     |     1 |    10%   |      2.0 |     10%
  ai-assisted-qa                      |     0 |     0%   |      0.0 |      0%
  -----------------------------------------------------------------------
  Total                               |       |    70%   |          |     80%
```

## Output Schema — Full Format (updated)

Word document table per component. **Old schema → New schema**:

```
OLD:  Label | Count | Total SP | Key | Summary | SP
NEW:  Label | Count | Count,% | Total SP | Total SP,% | Key | Summary | SP
```

Plus a "Total" row appended at the end of the label table (same as short format):

```
Row:  "Total" | (empty) | <sum of Count,%> | (empty) | <sum of Total SP,%> | (empty) | (empty) | (empty)
```

### Full-format console (debug) example

```
Component: Team Alpha
  Label                          | Count | Count,% | Total SP | Total SP,% | Key          | Summary                                            |   SP
  -----------------------------------------------------------------------...
  ai-assisted                    |     4 |    40%  |     10.0 |       50%  | PROJ-1       | Implement login                                    |  3.0
  ai-assisted                    |     4 |    40%  |     10.0 |       50%  | PROJ-2       | Add OAuth support                                  |  2.0
  ...
  Total                          |       |    70%  |          |       80%  |              |                                                    |
```

## Percentage Semantics

- `Count,%` = `(label group issue count / component total issue count) * 100`
- `Total SP,%` = `(label group total SP / component total SP) * 100`
- Denominator is all issues in the component (labeled + unlabeled combined)
- Values are rounded to 1 decimal place; whole numbers omit the `.0` (e.g. `40%` not `40.0%`)
- Percentages can exceed 100% due to non-exclusive label group membership
- When the component denominator is 0, all percentages show `0%`
