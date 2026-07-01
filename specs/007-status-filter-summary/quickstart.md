# Quickstart: Status Filter + Summary Row

**Feature**: 007-status-filter-summary
**Date**: 2026-06-26

## What This Feature Does

Adds a status filter bar and summary row to the existing sprint labels web dashboard. No new binaries, no new API endpoints, no new dependencies.

## How to Run

```bash
# Build (existing make target)
make build

# Start the dashboard (existing command)
./bin/web-sprint-labels-report -port 8080

# Open in browser
open http://localhost:8080
```

## How to Use the New Features

### Status Filter Bar

1. Select a team and sprint from the dashboard.
2. Below the team/sprint selector, you will see a row of **status pills** (one per unique status in the sprint).
3. Click a pill to **filter** the table to only issues with that status.
4. Click the pill again to **unfilter** (include that status back).
5. Click all pills to deselect them — this shows **all issues** (no filter).

### Summary Row

1. Below the issues table, a **summary row** shows per-label statistics.
2. For each configured label, the summary displays: **count** (number of filtered issues with that label) and **total story points** (sum of story points for those issues).
3. The summary recalculates automatically when you toggle status pills.

## What Has NOT Changed

- **Backend code**: `internal/dashboard/handler.go` is unchanged. All existing endpoints work exactly as before.
- **Label toggling**: The existing label toggle buttons work exactly as before.
- **API responses**: All existing API responses are unchanged.
- **Configuration**: No new `.env` variables.
- **CLI binaries**: No new binaries; existing binaries unchanged.

## Files Modified

| File | Change |
|------|--------|
| `web/index.html` | Added `<div id="status-filter-bar">` |
| `web/style.css` | Added `.pill`, `.pill.active`, `.summary-row` CSS |
| `web/app.js` | Added status filtering and summary row logic |

## Manual Testing

1. Open the dashboard → select team and sprint.
2. Verify status pills appear (one per unique status).
3. Click individual pills → verify table updates.
4. Deselect all pills → verify all issues reappear.
5. Verify summary row shows correct counts and story points.
6. Test dark theme — verify pills and summary render correctly.
