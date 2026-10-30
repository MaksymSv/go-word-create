# Data Model: Implementer Resolution Rule

**Branch**: `010-implementer-resolution` | **Date**: 2026-10-30

No fields, structs or JSON shapes change. `jiraservice.Issue.Implementer` (string) stays the only output. This document defines how its value is computed.

## Inputs (existing)

| Input | Source | Notes |
|-------|--------|-------|
| `current` | `issue.Fields.Assignee.DisplayName`, or `""` | Current assignee. |
| Changelog histories | `issue.Changelog.Histories`, oldest first | May be nil (not expanded) or empty. |
| Item (field, fromString, toString) | `ChangelogItems` | `field` is matched case-insensitively. For `assignee` items, `fromString`/`toString` are the previous/new display names (empty when unassigned). For `status` items, `toString` is the new status name. |

## Constants

| Name | Value | Role |
|------|-------|------|
| `implementerPrimaryStatus` | `In Progress` | First reference move (FR-001). |
| `implementerFallbackStatus` | `In Review.` | Second reference move (FR-002). Includes the trailing period. |

## Algorithm

```text
current = display name of issue.Fields.Assignee, or "" if nil
if issue.Changelog == nil: return current                       # FR-008

assignee = fromString of the first assignee item in all histories
           if there is none: current                            # FR-004

primarySeen = false
fallbackSeen = false
fallback = ""

for each history in order:
    for each item in history where field == "assignee":
        assignee = item.toString                                # "" means unassigned, FR-006
    for each item in history where field == "status":
        if toString equals "In Progress" (case-insensitive) and not primarySeen:
            primarySeen = true
            if assignee != "": return assignee                  # FR-001
        else if toString equals "In Review." (case-insensitive) and not fallbackSeen:
            fallbackSeen = true
            fallback = assignee

if fallback != "": return fallback                              # FR-002
return current                                                  # FR-003 (may be "")
```

Notes:
- Assignee items of an entry are applied before its status items, so an assignment made in the same change as the move counts (FR-005).
- Only the first "In Progress" move and the first "In Review." move are used.
- The "In Review." fallback is only reached when the "In Progress" step returned nothing.

## Worked examples

| Case | History (oldest first) | Current | Result | Why |
|------|------------------------|---------|--------|-----|
| A assigned at creation, never changed | status -> In Progress; status -> In Review. | A | A | No assignee items, so `assignee = current` (FR-004). |
| A, then reassigned to B after start | status -> In Progress; assignee A -> B; status -> In Review. | B | A | First assignee item's `fromString` is A; the move to In Progress happens before the change. |
| A, B, then start | assignee A -> B; status -> In Progress | B | B | Latest assignee before the move. |
| Unassigned at start, C before review | status -> In Progress; assignee "" -> C; status -> In Review. | C | C | Primary empty, fallback C. |
| Skipped In Progress | assignee "" -> C; status -> In Review. | C | C | Fallback C. |
| Assignee removed before start | assignee A -> ""; status -> In Progress; assignee "" -> D; status -> In Review. | D | D | Assignee empty at start (FR-006), fallback D. |
| Never reached either status | status -> Closed | E | E | Current assignee (FR-003). |
| Nothing at all | (empty) | none | "" | Shown as "—" in the dashboard. |
| Several In Progress moves, first empty | status -> In Progress; assignee "" -> F; status -> In Progress; status -> In Review. | F | F | Second In Progress ignored; fallback is F at In Review. |
