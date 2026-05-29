# CLI Contract: get-sprint-label-report

**Feature**: Sprint Label Aggregation Report
**Branch**: `001-sprint-label-report`
**Binary**: `bin/get-sprint-label-report`
**Date**: 2026-05-29

## Invocation

```
get-sprint-label-report -sprint="<sprint-name>" [options]
```

## Flags

| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `-sprint` | string | Yes | — | Exact Jira sprint name to report on |
| `-output` | string | No | `$DEFAULT_OUTPUT_FILE` | Output `.docx` file path |
| `-format` | string | No | `short` | Report format: `short` or `full` |
| `-debug` | bool | No | `false` | Print report to stdout; do not write Word document |

## Environment Variables

All variables are loaded from `.env` (via `godotenv`) with env override.

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JIRA_URL` | Yes | — | Jira Cloud base URL |
| `JIRA_USERNAME` | Yes | — | Jira account email |
| `JIRA_API_TOKEN` | Yes | — | Jira API token |
| `JIRA_BOARD_NAME` | Yes | — | Board name containing the target sprint |
| `JIRA_PROJECT_KEY` | Yes | — | Jira project key for epic lookup |
| `TEAMS` | Yes | — | Comma-separated list of component names (existing requirement) |
| `DEFAULT_OUTPUT_FILE` | No | `sprint-label-report.docx` | Default output file name |
| `REPORT_LABELS` | No | `ai-assisted,ai-assisted-ba,ai-assisted-dev,ai-assisted-qa` | Ordered comma-separated label list |
| `JIRA_EPIC_FIELD` | No | `customfield_14500` | Custom field ID for Epic Link |
| `JIRA_SP_FIELD` | No | `customfield_10004` | Custom field ID for Story Points |

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Missing required flag or environment variable; see stderr for details |
| `1` | Jira API error (board not found, sprint not found, auth failure) |
| `1` | Document write failure |

## Output: Short Format (`-format=short`)

One Word document section per Jira component. Each section contains:

**Label Aggregation Table** — columns: `Label | Count | Total SP`

```
| Label              | Count | Total SP |
|--------------------|-------|----------|
| ai-assisted        |     5 |     12.0 |
| ai-assisted-ba     |     2 |      4.0 |
| ai-assisted-dev    |     3 |      8.0 |
| ai-assisted-qa     |     1 |      2.0 |
```

Followed by the **Unlabeled Issues Table** — columns: `Key | Summary | SP`

```
| Key      | Summary                          | SP  |
|----------|----------------------------------|-----|
| PROJ-42  | Update onboarding flow           | 3.0 |
| PROJ-55  | Fix null pointer in payment API  | 2.0 |
```

## Output: Full Format (`-format=full`)

One Word document section per Jira component. Each section contains:

**Label Aggregation Table** — columns: `Label | Count | Total SP | Key | Summary | SP`

Each issue in a label group occupies one row. Label name, Count, and Total SP repeat on
every row within the same group (for readability when rows are paginated):

```
| Label           | Count | Total SP | Key     | Summary               | SP  |
|-----------------|-------|----------|---------|-----------------------|-----|
| ai-assisted     |     2 |      8.0 | PROJ-10 | Enable AI suggestions | 3.0 |
| ai-assisted     |     2 |      8.0 | PROJ-11 | AI-assisted code gen  | 5.0 |
| ai-assisted-dev |     1 |      3.0 | PROJ-10 | Enable AI suggestions | 3.0 |
```

Followed by the same **Unlabeled Issues Table** as short format.

## Output: Debug / Console Mode (`-debug`)

Prints the same data structure to stdout in a plain-text tabular layout.
No Word document is written. Useful for verifying data before generating a document.

```
Sprint: "Sprint 42"
Component: PROCESSING
  Label Aggregation:
    ai-assisted        | 2 issues | 8.0 SP
    ai-assisted-ba     | 0 issues | 0.0 SP
    ...
  Unlabeled Issues:
    PROJ-42 | Update onboarding flow | 3.0 SP
    ...
```

## Error Messages

Errors go to stderr. Format: `Error: <context>: <detail>`

Examples:
- `Error: required environment variable JIRA_URL is not set`
- `Error: sprint "Sprint 42" not found on board "My Board"`
- `Error: failed to save document "output.docx": permission denied`
