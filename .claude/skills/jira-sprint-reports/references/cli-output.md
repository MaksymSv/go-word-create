# CLI output layouts (fallback path only)

Prefer the MCP tools, which return structured JSON. These layouts matter only when parsing CLI stdout. Logs and errors go to stderr.

## `get-sprint-label-report` (`make run-sprint-label-report`)

With several sprints, each section starts with `=== Sprint: <name> ===`. Each component starts with `Component: <name>`, with "No Component" last. Columns are separated by ` | `.

### `FORMAT=short` (default)

```
Label | Count | Count,% | Total SP | Total SP,%
------------------------------------------------
<label>  | n | pct | sp | pct       (one row per configured label, in configured order)
------------------------------------------------
Total |   | pct (labeled count) |   | pct (labeled SP)
```

### `FORMAT=full`

```
Label | Count | Count,% | Total SP | Total SP,% | Key | Summary (50 chars) | SP
```

One row per issue under each label. The label's Count and percentages repeat on each row. A label with no issues prints one row with empty Key, Summary and SP. A `Total` row follows with the labeled percentages.

### Unlabeled issues (both formats, only if any)

```
Unlabeled Issues:
Key | Summary (60 chars) | SP
```

Summaries are truncated. Percentages are formatted as text, so strip `%` before doing maths.

## `get-month-issues -debug` (`make run-month ... LOGONLY=1`)

Per team, in configured order:

```
Closed Issues for team <component> (<n>):
<Type>|<Key>|<Summary 80>|<Epic 40>|<SP>|<Status>
Open Issues for team <component> (<n>):
...
```

Fields are fixed-width padded and separated by `|`. Trim whitespace. Summary and Epic are truncated. Labels and implementer are not printed by the CLI, so use `month_issues` when you need them. The `Found N issues...` and `Total issues...` lines are log output on stderr.
