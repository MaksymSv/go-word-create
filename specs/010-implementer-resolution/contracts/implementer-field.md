# Contract: `implementer` field

**Branch**: `010-implementer-resolution` | **Date**: 2026-10-30

The shape of the field does not change; only its meaning does. No endpoint, tool, flag or response schema is added or removed.

## Where it appears

| Surface | Location | Type |
|---------|----------|------|
| Dashboard API | `GET /api/sprints/{sprintID}/issues` -> `issues[].implementer` | string, `""` when unknown |
| MCP | `get_sprint_issues` -> `issues[].implementer` | string, `""` when unknown |
| Dashboard UI | "Implementer" column, shown as "—" when empty | text |

## Meaning (replaces the meaning in `specs/006-implementer-column/contracts/api.md`)

Display name of the person assigned to the issue:

1. just before the issue first moved to "In Progress"; otherwise
2. just before the issue first moved to "In Review."; otherwise
3. currently assigned; otherwise
4. `""`.

An assignee set in the same change as the move counts. The creation-time assignee counts even if never changed.

## Not affected

`sprint_issues`, `sprint_label_report`, `month_issues`, CSV and Word outputs do not carry the field.
