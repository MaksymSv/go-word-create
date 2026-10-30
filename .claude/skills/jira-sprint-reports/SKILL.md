---
name: jira-sprint-reports
description: "Answer questions about Jira sprints, AI-assisted label usage, and monthly in-progress work using this repo's Jira reporting tools. Use for: sprint label report, AI-assisted label percentages, issues per status in a team's sprint, unlabeled issues, issues In Progress during a month (Closed vs Open), or adding/removing an ai-assisted label on an issue."
user-invocable: true
---

# Jira sprint and label reports

This repo reads Jira (project, teams, boards and report labels come from `.env`) and reports on sprints and AI-usage labels. Prefer the MCP tools. Fall back to the CLI only when the MCP server is not connected.

## Safety rules

- Never read, print, or quote `.env`. It holds a Jira API token. Use `list_teams` to see configuration.
- Jira data changes only through `set_issue_label`, and only after the user has explicitly approved (see Write protocol).

## Tool picker (MCP server `jira-reports`, tools appear as `mcp__jira-reports__<name>`)

| Question | Tool |
|---|---|
| Which teams / labels / issue types are configured? | `list_teams` |
| What are a team's recent sprints? Need a sprint ID | `list_sprints(team, limit?)` |
| Issues of one sprint, optionally by status, with per-label counts | `get_sprint_issues(sprint_id, statuses?)` |
| Issue table (type, key, summary, epic, SP, status) for one or more sprints by name; CSV on request | `sprint_issues(sprints[], team?, statuses?, format?)` |
| Label usage % for one or more sprints, by component | `sprint_label_report(sprints[], team?)` |
| What was In Progress during a month; what closed | `month_issues(month "YYYY.MM", team?)` |
| Add or remove a tracked label on an issue | `set_issue_label(issue_key, label, action, confirm)` |

## Workflows

**Label report for Sprint N.** Call `sprint_label_report` with the exact sprint name(s). Names are case-sensitive and must match Jira exactly. If a name is not found, call `list_teams` then `list_sprints` for the relevant team and use the real name. Present per component: each label's count, count %, story points, SP %, the "labeled" totals, and the unlabeled issues.

**Status breakdown for a team's sprint.** `list_teams` (confirm the component name), `list_sprints(team)` (pick the sprint, usually the newest active one), then `get_sprint_issues(sprint_id, statuses=[...])`. `availableStatuses` lists the real status names. `unmatchedStatuses` shows requested statuses that matched nothing, so check spelling instead of reporting zero. The summary covers only the filtered issues.

**Sprint issue table / CSV.** Call `sprint_issues` with the exact sprint name(s); it mirrors `get-sprint-issues -sprint="A,B"`. For a CSV, pass `format="csv"`: each sprint comes back with `csv` text and a suggested `fileName`. The server cannot write files, so show the CSV or save it yourself under that name.

**Month report.** `month_issues("2025.10")`. "Closed" means current status is exactly `Closed`. Everything else touched during the month is "Open". Issues qualify through the changelog (they were In Progress at some point in the month), not by sprint.

**Tagging issues.** Follow the write protocol below.

## Metric semantics (get these right when explaining numbers)

- Label membership is non-exclusive. An issue with two tracked labels counts in both rows, so label rows do not sum to the component total.
- Percentages use the component's total issues (or total SP) as the denominator, not the sum of label rows.
- "Total" / `labeledCountPct` means issues with at least one tracked label.
- Tracked labels are `configuredLabels` (default ai-assisted, ai-assisted-ba, ai-assisted-dev, ai-assisted-qa). Other Jira labels are ignored.
- An issue in several components appears under each. Issues without a component go under "No Component" (listed last). `issueCount` per component is the unique issue count.
- Only configured issue types (default Bug, Story, Task) are reported by the label and month tools.
- Story points of 0 usually mean unestimated, not zero effort.

## Write protocol (`set_issue_label`)

1. Collect the exact list of changes: issue key, label, add or remove.
2. Call each with `confirm=false` (the default) to validate and get a preview. Nothing changes in Jira.
3. Show the user the full list and ask for approval.
4. Only after an explicit yes for those exact changes, call again with `confirm=true`, one issue at a time. Never apply more than what was approved.
5. Report per issue whether it was applied or failed. Do not retry blindly on failure.

Only labels in `configuredLabels` are accepted. A generic "go ahead" earlier in the conversation does not count as approval of a specific change.

## CLI fallback (MCP server not connected)

Run from the repo root so `.env` resolves.

- Label report: `make run-sprint-label-report SPRINT="Sprint 16" FORMAT=full` (several sprints: `SPRINT="Sprint 16,Sprint 17"`). `FORMAT=short` omits issue rows.
- Month report: `make run-month MONTH=2025.10 LOGONLY=1`. Always pass `LOGONLY=1`. Without it the command writes a `.docx` file instead of printing.
- The CLI prints text tables. See [references/cli-output.md](references/cli-output.md) for the column layouts.
- Live MCP build: `make build-mcp` produces `bin/mcp-jira-reports`, which `.mcp.json` points to.

## Troubleshooting

- "sprint not found": the name must match exactly, including case and spacing. Look it up with `list_sprints`.
- "team not found": use the component name exactly as `list_teams` returns it.
- `list_sprints` returns nothing: the board has no active or closed sprints, or the board name in `TEAMS` is wrong. Tell the user. Do not guess.
- Tool unavailable or connection fails: check that `bin/mcp-jira-reports` exists (`make build-mcp`), then use the CLI fallback. Missing-variable errors name the `.env` key but never show its value.
- Sprint results with suspiciously few issues: the Agile API may paginate. Cross-check with `get_sprint_issues` for the same sprint and flag a mismatch to the user.
