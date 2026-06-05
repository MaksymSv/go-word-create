# Feature Specification: Web Sprint Labels Dashboard

**Feature Branch**: `005-web-sprint-labels-dashboard`

**Created**: 2026-06-05

**Status**: Draft

**Input**: User description: "I want to create new cmd/web-sprint-labels-report. I need to have a web dashboard in browser with the following structure. The page should contain a toolbar and main information area below. The toolbar should contain two sections of buttons. Left section should contain the list of teams (component names from configuration) as buttons stacked left to right. The first team button should be selected (like pushed down) by default. The second part of the toolbar should be a list of buttons (stacked left to right) with the names of the last 5 sprints in the jira board associated with the selected component (on the left side of the toolbar). By clicking some particular sprint button the main section of the dashboard should load the list of issues (from jira) in the selected sprint as a table. The format of issue table line should be IssueType|IssueKey|Summary|Epic Name|Story Points|Status|Labels. The type of issue should be displayed as Jira issue icon. Use svg format. The column Labels should contain the list of labels from configured list which are assigned to particular issue but limited by the list of labels in the configuration. Each label should be represented as a small button. Pushing this button the label should be assigned to the target jira issue. The dashboard should have a dark and light theme. Put the theme switcher to the right most side of the toolbar."

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Browse Sprint Issues by Team (Priority: P1) 🎯 MVP

A team lead opens the dashboard in a browser. They see a toolbar with team buttons on the left. The first team is pre-selected, and the 5 most recent sprints for that team's board appear as buttons to the right. By clicking a sprint button, the main area loads a table of all issues in that sprint with columns: issue type icon, issue key, summary, epic name, story points, status, and labels.

**Why this priority**: This is the core read-only view that all other functionality builds on. Without it the dashboard provides no value.

**Independent Test**: Start the server, open the dashboard URL, verify the first team is pre-selected, verify 5 sprint buttons appear, click one sprint button, verify a table of issues loads with all 7 columns visible.

**Acceptance Scenarios**:

1. **Given** the server is running and the browser opens the dashboard, **When** the page loads, **Then** all team names from the configuration appear as buttons in the left toolbar section, the first team button appears visually selected, and 5 sprint buttons for that team's board appear in the right toolbar section.
2. **Given** a team button is clicked, **When** the team selection changes, **Then** the sprint buttons update to show the 5 most recent sprints for the newly selected team's board, and any currently loaded issue table is cleared.
3. **Given** a sprint button is clicked, **When** the sprint is selected, **Then** the main area displays a table of all issues in that sprint; each row shows the issue type as an icon, the issue key, summary, epic name, story point count, current status, and any matching configured labels.
4. **Given** a sprint has no issues, **When** that sprint button is clicked, **Then** the main area shows an empty-state message ("No issues found for this sprint").
5. **Given** the Jira API is unavailable, **When** the user clicks a sprint or team button, **Then** an error message is displayed in the main area without crashing the page.

---

### User Story 2 — Assign Labels to Issues from the Dashboard (Priority: P2)

An operator reviews the issue table and sees which labels from the configured label list are already assigned to each issue. For issues that should have a label, they click the label button next to the issue. The label is applied to the issue in Jira immediately, and the button state updates to reflect the assignment.

**Why this priority**: This is the primary write operation; US1 must be complete before this adds value.

**Independent Test**: Load any sprint, find an issue without a configured label, click a label button next to it, verify the label appears applied (button changes state), refresh the page and verify the label persists on that issue.

**Acceptance Scenarios**:

1. **Given** an issue already has a configured label assigned, **When** the row is rendered, **Then** that label button appears in an "active" / applied visual state.
2. **Given** a label button is in the inactive state, **When** the user clicks it, **Then** the label is added to the issue in Jira, and the button switches to the active state without a full page reload.
3. **Given** a label button is in the active state, **When** the user clicks it, **Then** the label is removed from the issue in Jira, and the button switches to the inactive state.
4. **Given** the Jira label update fails, **When** the button is clicked, **Then** the button reverts to its previous state and an error notification is shown to the user.
5. **Given** an issue has labels outside the configured list, **When** the row is rendered, **Then** those extra labels are not displayed in the Labels column (only configured labels are shown).

---

### User Story 3 — Dark / Light Theme Toggle (Priority: P3)

A user prefers a dark working environment. They click the theme toggle button at the far right of the toolbar to switch between light and dark modes. The chosen theme persists across page refreshes.

**Why this priority**: Comfort feature that doesn't block any core functionality.

**Independent Test**: Load the dashboard, click the theme toggle, verify all UI elements (toolbar, table, buttons) switch to the alternate colour scheme, refresh the page and verify the selected theme is remembered.

**Acceptance Scenarios**:

1. **Given** the dashboard loads for the first time, **When** no preference is stored, **Then** the dashboard defaults to the light theme.
2. **Given** the user clicks the theme toggle, **When** the toggle is activated, **Then** the entire page switches to the alternate theme instantly without a reload.
3. **Given** the user has selected the dark theme, **When** the page is refreshed or revisited, **Then** the dark theme is still active.

---

### Edge Cases

- What happens when a team's board has fewer than 5 sprints? — Show only the available sprints (1–4 buttons).
- What happens when two teams share the same board name? — Each team button is still shown; both will display the same sprint list.
- What happens when an issue key link is clicked? — Issue key is a clickable link that opens the issue in Jira in a new tab.
- What happens when story points are not set for an issue? — Display a dash (`—`) or `0` in the SP column.
- What happens if configured labels list is empty? — The Labels column is hidden or shows "—" for all rows.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The dashboard MUST display one button per team in the left toolbar section, using the component name from configuration; buttons are ordered as configured.
- **FR-002**: The first team button MUST be visually selected by default when the page loads.
- **FR-003**: The right toolbar section MUST display up to 5 buttons representing the most recent sprints (active or closed) for the currently selected team's board, ordered newest first.
- **FR-004**: Clicking a sprint button MUST load all issues from that sprint into the main area table.
- **FR-005**: The issue table MUST have exactly 7 columns: issue type (icon), issue key (link to Jira), summary, epic name, story points, status, and labels.
- **FR-006**: The issue type icon MUST be an SVG representation matching the Jira issue type (Bug, Story, Task, Sub-task, Epic, etc.).
- **FR-007**: The Labels column MUST display only the subset of configured labels that are currently assigned to the issue; unassigned configured labels are shown as inactive buttons.
- **FR-008**: Clicking an inactive label button MUST add that label to the issue in Jira; clicking an active label button MUST remove it.
- **FR-009**: Label assignment and removal MUST reflect in the button state immediately without a full page reload.
- **FR-010**: The theme toggle button MUST be positioned at the far right of the toolbar and MUST switch between light and dark themes.
- **FR-011**: The selected theme MUST be persisted in the browser and restored on next visit.
- **FR-012**: Switching teams MUST reload the sprint list for the new team's board.
- **FR-013**: The issue key in the table MUST be a hyperlink that opens the Jira issue in a new browser tab.

### Key Entities

- **Team**: Configured team entry with a component name and a board name; drives both the team selector buttons and the sprint lookups.
- **Sprint**: A Jira sprint belonging to a board; has a name, state (active/closed), and a set of issues.
- **Issue**: A Jira issue with type, key, summary, epic name, story points, status, and a list of labels.
- **ConfiguredLabel**: A label string from the `REPORT_LABELS` configuration; the dashboard only shows and manages labels from this set.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The issue table for any sprint loads and is fully visible within 3 seconds of clicking a sprint button (under normal network conditions).
- **SC-002**: Clicking a label button applies or removes the label in Jira and the button state updates within 2 seconds.
- **SC-003**: Switching between teams updates the sprint list within 2 seconds.
- **SC-004**: 100% of configured teams appear as toolbar buttons on page load with no manual configuration step beyond the existing `.env` file.
- **SC-005**: The dark/light theme preference persists correctly across at least 5 consecutive browser sessions without intervention.

## Assumptions

- The dashboard is a single-page application served by the Go binary on a configurable port (default 8080); no separate build step or CDN is required for local use.
- All Jira credentials and configuration (teams, board names, configured labels) are read from the existing `.env` file — no new configuration format is introduced.
- The dashboard is for internal/team use only; no user authentication is required.
- The `REPORT_LABELS` configuration variable determines which labels appear as buttons in the Labels column; if not set, the default AI label set is used.
- Sprint ordering uses the sprint's start date or sequence number; "most recent" means highest sequence / latest start.
- The Go HTTP server serves both the static dashboard page and the API endpoints that the browser calls for data and Jira mutations.
- The issue type icon set covers at minimum: Bug, Story, Task, Sub-task, Epic; other types fall back to a generic issue icon.
