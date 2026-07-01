# Feature Specification: Status Filter + Summary Row

**Feature Branch**: `007-status-filter-summary`

**Created**: 2026-06-26

**Status**: Draft

**Input**: User description: "Add a status filter bar (pill buttons for 'To Do', 'In Progress', 'Closed', etc.) above the issues table, plus a totals row below showing count and story points per label for the filtered view."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Filter Sprint Issues by Status (Priority: P1)

A Scrum Master opens the dashboard, selects a sprint, and wants to see only the issues that are still open (not "Closed"). They click status pills for the statuses they care about (e.g., "To Do", "In Progress", "In Review"), and the table updates to show only matching issues.

**Why this priority**: This is the core value proposition — without filtering, the dashboard shows all statuses equally, making it impossible to answer "what is still open?" at a glance. This is the most common daily workflow for sprint monitoring.

**Independent Test**: A user can load a sprint, toggle status pills, and verify the table rows update to match the selected statuses.

**Acceptance Scenarios**:

1. **Given** a sprint with issues in multiple statuses, **When** the user clicks a status pill (e.g., "Closed"), **Then** the issues table updates to show only issues whose status matches the selected pills, and issues with non-selected statuses are hidden.

2. **Given** no status pills are selected, **When** the user views the sprint, **Then** all issues are shown (equivalent to no filter — the default state shows everything).

3. **Given** some status pills are selected, **When** the user clicks a selected pill to deselect it, **Then** the table updates to include issues with that status again, and the pill visually reflects the deselected state.

4. **Given** exactly one status pill is selected, **When** the user views the sprint, **Then** only issues matching that single status are displayed.

---

### User Story 2 - View Filtered Summary Totals (Priority: P2)

A delivery lead wants to answer: "How many story points labeled `ai-assisted-dev` are still not Closed?" They apply a status filter and look at the summary row beneath the table to see counts and story-point totals per configured label, computed only over the currently filtered issues.

**Why this priority**: This turns raw data into actionable sprint health information. Without the summary row, users must manually count and sum story points from filtered rows — error-prone and slow.

**Independent Test**: A user can filter to a single status (e.g., "In Progress") and verify the summary row shows correct counts and story-point totals per label for only those issues.

**Acceptance Scenarios**:

1. **Given** a sprint loaded with issues and status pills selected, **When** the table renders, **Then** a summary row appears below the table showing, for each configured label, the count of filtered issues bearing that label and the total story points of those issues.

2. **Given** a status filter is applied that excludes all issues with a particular configured label, **When** the user views the summary row, **Then** that label shows a count of 0 and story points of 0 (or "—" if no issues have story points).

3. **Given** the user changes the status filter (adds/removes a pill), **When** the table updates, **Then** the summary row recalculates to reflect only the newly filtered issues.

---

### Edge Cases

- What happens when a sprint has only one unique status (e.g., all issues are "Closed")? The status filter bar should still render all statuses, but only the matching pill will have any effect; selecting non-matching statuses will show an empty table with "No issues match selected statuses."
- How does the system handle issues with no status value (null/empty)? These are treated as matching no status pill — they are hidden when any pill is selected, and shown when no pills are selected (default state).
- How does the system handle an empty sprint (zero issues)? The status filter bar and summary row are not rendered; instead, the existing "No issues found" message is shown.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST dynamically discover all unique status values from the issues loaded for the selected sprint and render a status pill for each one.
- **FR-002**: System MUST allow users to select and deselect status pills, where each pill toggles the inclusion of issues matching that status.
- **FR-003**: System MUST update the issues table in real time when the user selects or deselects a status pill, showing only issues whose status matches at least one selected pill.
- **FR-004**: System MUST display a summary row below the issues table showing, for each configured label, the count of filtered issues bearing that label and the total story points of those issues.
- **FR-005**: System MUST recalculate the summary row whenever the status filter changes (pills selected/deselected).
- **FR-006**: When no status pills are selected, System MUST show all issues (no filtering applied).

### Key Entities *(include if feature involves data)*

- **Status Filter State**: The set of status values currently selected by the user (a subset of all statuses present in the sprint). Empty set means "show all."
- **Filtered Issue Set**: The subset of sprint issues whose status matches at least one selected status pill. Used for both table rendering and summary calculation.
- **Summary Totals**: Per-label aggregates (count of filtered issues, total story points) computed over the filtered issue set.

## Success Criteria *(mandatory)*

### Measurable Outables

- **SC-001**: Users can filter a sprint's issues to a single status and see the correct subset of rows within 1 second of clicking a pill (no full page reload).
- **SC-002**: The summary row accurately reflects the count and story-point totals of the currently filtered issues, matching what a user would obtain by manually counting the visible rows.
- **SC-003**: 90% of users who apply a status filter can identify the total story points of issues with a specific label from the summary row without needing to count visible rows manually.
- **SC-004**: The status filter bar renders all unique statuses present in a sprint's issue set, ensuring no status value is hidden from the user.

## Assumptions

- The dashboard is used by a single user per session (no real-time multi-user sync required).
- The number of unique statuses in a typical sprint is small (under 10), so rendering a pill for each is practical.
- The existing `labelreport.Aggregate()` backend logic already computes the per-label count and story-point totals needed for the summary row; this feature primarily wires that logic to the frontend filtered view.
- Status values come from Jira and may vary between teams (e.g., one team uses "In Review.", another uses "Done"); the filter bar adapts to whatever statuses exist in the loaded sprint.
- Story points may be null or zero for some issues (e.g., tasks without estimates); the summary row handles these gracefully (showing "—" or counting them as 0 story points).
