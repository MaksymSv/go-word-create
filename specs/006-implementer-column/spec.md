# Feature Specification: Implementer Column

**Feature Branch**: `006-implementer-column`

**Created**: 2026-06-08

**Status**: Draft

**Input**: User description: "I need to add "Implementer" column after Epic column. This column should indicate a Name of the assignee of the issue at the moment issue was in "In Progress" state. If issue was not In Progress yet, the current assignee should be used."

## Clarifications

### Session 2026-06-08

- Q: Should the Implementer column be added to Word document reports as well as the web dashboard? → A: Dashboard only — no changes to Word documentation.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Implementer in Web Dashboard (Priority: P1)

A dashboard user views the sprint issues table in the web dashboard and sees an "Implementer" column after the Epic column. For each issue, the cell shows the full name of the person who was assigned when the issue first moved to "In Progress". If the issue never reached "In Progress", the cell shows the current assignee instead. If neither is available, the cell shows "—".

**Why this priority**: This is the sole delivery target for the feature. Knowing who implemented each issue gives managers a quick view of ownership directly in the dashboard without generating a report.

**Independent Test**: Open the web dashboard, select a team and sprint, and verify the Implementer column appears after the "Epic" column in the issues table with correct names populated.

**Acceptance Scenarios**:

1. **Given** an issue that transitioned to "In Progress" with person A assigned, and later reassigned to person B, **When** the sprint issues table is displayed, **Then** the Implementer column shows person A's name (the assignee at the first "In Progress" transition).

2. **Given** an issue that was never moved to "In Progress" but has a current assignee, **When** the sprint issues table is displayed, **Then** the Implementer column shows the current assignee's name.

3. **Given** an issue that was never moved to "In Progress" and has no current assignee, **When** the sprint issues table is displayed, **Then** the Implementer column shows "—".

4. **Given** an issue that transitioned to "In Progress" multiple times, **When** the sprint issues table is displayed, **Then** the Implementer column shows the assignee from the **first** "In Progress" transition.

---

### Edge Cases

- What happens when an issue has "In Progress" history but the assignee field was empty at that moment? Show "—" (no assignee was set at that transition).
- What happens when fetching issue history fails for a particular issue? Show current assignee as fallback; do not fail the entire table load.
- How does the system handle "In Progress" state names that differ across Jira configurations? Use case-insensitive exact match for "In Progress".

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The sprint issues table in the web dashboard MUST include an "Implementer" column positioned immediately after the "Epic" column.
- **FR-002**: The "Implementer" value MUST be the full display name of the person assigned to the issue at the time the issue first transitioned to "In Progress" state.
- **FR-003**: If an issue has never transitioned to "In Progress", the system MUST fall back to the issue's current assignee display name.
- **FR-004**: If neither an "In Progress" assignee nor a current assignee exists, the Implementer cell MUST display "—".
- **FR-005**: When an issue transitioned to "In Progress" multiple times, the system MUST use the assignee from the **first** such transition.
- **FR-006**: The feature MUST NOT affect the sprint label report tables, which have a different structure (label-count aggregates, not per-issue rows).
- **FR-007**: If fetching issue history fails for an individual issue, the system MUST degrade gracefully by showing the current assignee (or "—") rather than failing the whole table load.

### Key Entities

- **Issue Changelog**: The ordered history of changes to a Jira issue; each entry records the field changed, old/new values, the author, and a timestamp.
- **Status Transition**: A changelog entry where the field is "status" and the new value is "In Progress"; the assignee at this moment is the candidate implementer.
- **Implementer**: The resolved display name for an issue, derived from the first "In Progress" transition assignee or current assignee as fallback.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of issue rows in the dashboard sprint issues table display the correct implementer name (verified against Jira issue history for a sample sprint).
- **SC-002**: The Implementer column appears in the correct position (after Epic) in the dashboard issues table.
- **SC-003**: Dashboard sprint issues load time does not increase by more than 2× compared to baseline (history fetching is efficient enough to remain acceptable).
- **SC-004**: Issues with no "In Progress" history always show the current assignee (or "—"), never an error or missing cell.

## Assumptions

- The "In Progress" status is named exactly "In Progress" in Jira (case-insensitive match); other in-flight statuses do not qualify.
- Issue changelog (history) is accessible via the existing Jira API connection when fetching sprint issues for the dashboard.
- The feature applies only to the web dashboard sprint issues table.
- Word document generation (month issues, sprint issues, sprint label report) is out of scope for this feature.
- Display names are fetched as-is from Jira without additional formatting.
- When the history API returns no changelog entries for an issue, the current assignee is used as the sole fallback.
