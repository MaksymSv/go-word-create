# Feature Specification: Implementer Resolution Rule

**Feature Branch**: `010-implementer-resolution`

**Created**: 2026-10-30

**Status**: Draft

**Input**: User description: "I need to calculate issue implementer differently. Take the issue assignee before the ticket moved to In Progress. As a fall back (if there wasn't assignee before moving to In Progress) take assignee before moving to In Review."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Implementer Is Whoever Held the Issue When Work Started (Priority: P1)

A delivery lead looks at the issues of a sprint and sees, for each issue, an Implementer: the person who was assigned to the issue just before it first moved to "In Progress". Reassignments after work started (for example to a reviewer or tester) do not change who is shown.

**Why this priority**: This is the core of the request. The Implementer value is only useful if it names the person who actually started the work, and today it is derived from a different moment in the issue's life.

**Independent Test**: Pick a sprint, take several issues whose history is known (assigned, moved to In Progress, then reassigned), and check that the Implementer shown for each is the person assigned right before the first move to In Progress.

**Acceptance Scenarios**:

1. **Given** an issue assigned to person A that moved to "In Progress" and was later reassigned to person B, **When** the sprint issues are shown, **Then** the Implementer is person A.
2. **Given** an issue assigned to person A at creation and never reassigned, which later moved to "In Progress", **When** the sprint issues are shown, **Then** the Implementer is person A.
3. **Given** an issue assigned to person A, then reassigned to person B, then moved to "In Progress", **When** the sprint issues are shown, **Then** the Implementer is person B (the assignee immediately before the move).
4. **Given** an issue that moved to "In Progress" more than once (for example back to "To Do" and in again), **When** the sprint issues are shown, **Then** the Implementer is the assignee before the **first** move to "In Progress".

---

### User Story 2 - Fall Back to the Assignee Before "In Review." (Priority: P2)

When there was no assignee at the time the issue first moved to "In Progress", or the issue never went through "In Progress", the Implementer is the person assigned right before the issue first moved to "In Review.".

**Why this priority**: Issues that skip "In Progress" or are picked up unassigned would otherwise show no implementer at all, which hides who did the work.

**Independent Test**: Take an issue that was unassigned when it moved to "In Progress" and was assigned before it moved to "In Review.", and an issue that went straight to "In Review." without passing through "In Progress"; check that each shows the assignee before the "In Review." move.

**Acceptance Scenarios**:

1. **Given** an issue that was unassigned when it moved to "In Progress" and assigned to person C before it moved to "In Review.", **When** the sprint issues are shown, **Then** the Implementer is person C.
2. **Given** an issue that never moved to "In Progress" but was assigned to person C before it moved to "In Review.", **When** the sprint issues are shown, **Then** the Implementer is person C.
3. **Given** an issue with an assignee before its move to "In Progress", **When** the sprint issues are shown, **Then** the "In Review." history is not consulted and the "In Progress" assignee is shown.

---

### User Story 3 - Sensible Value When Neither Moment Applies (Priority: P3)

When the issue has no usable assignee at either moment (it never reached either status, or it was unassigned each time), the table still shows a sensible value and never fails.

**Why this priority**: Open or newly created issues, and issues closed straight from "To Do", must still render a complete table.

**Independent Test**: Load a sprint that contains an issue still in "To Do" with an assignee and an issue with no assignee at all; check that the first shows its current assignee and the second shows "—".

**Acceptance Scenarios**:

1. **Given** an issue that never moved to "In Progress" or "In Review." and has a current assignee, **When** the sprint issues are shown, **Then** the Implementer is the current assignee.
2. **Given** an issue with no assignee at either moment and no current assignee, **When** the sprint issues are shown, **Then** the Implementer is shown as empty ("—" in the dashboard).
3. **Given** an issue whose history cannot be read, **When** the sprint issues are shown, **Then** that issue shows its current assignee (or "—") and the rest of the table still loads.

---

### Edge Cases

- The assignee was removed (set to unassigned) just before the move to "In Progress": this counts as no assignee, so the "In Review." fallback applies.
- The assignee is set in the same change that moves the issue to "In Progress" (for example a workflow that assigns the person who starts the work): that assignee counts as the one who held the issue when work started.
- The issue is still in "In Progress" (not yet in review): the first move to "In Progress" already exists, so the rule applies.
- The issue's assignee never changed after creation, so there is no assignee entry in its history: the creation-time assignee still counts as the assignee in effect.
- The issue moved to "In Review." several times: the first move is used.
- The "In Review." status is spelled with a trailing period in this Jira workflow and status names are compared case-insensitively; an unrelated status such as "In Review Done" does not count.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The Implementer of an issue MUST be the display name of the person assigned to the issue immediately before the issue **first** moved to "In Progress".
- **FR-002**: If the issue had no assignee at that moment, or never moved to "In Progress", the Implementer MUST be the display name of the person assigned immediately before the issue **first** moved to "In Review.".
- **FR-003**: If neither FR-001 nor FR-002 yields a person, the Implementer MUST be the issue's current assignee; if there is none, the Implementer MUST be empty (shown as "—" in the dashboard).
- **FR-004**: The assignee in effect at a given moment MUST include the person assigned when the issue was created, even if the assignee was never changed afterward.
- **FR-005**: An assignee set in the same change as the status move MUST be treated as the assignee in effect for that move.
- **FR-006**: Removing the assignee MUST be treated as "no assignee" from that point on.
- **FR-007**: Status names MUST be matched case-insensitively and exactly ("In Progress" and "In Review."); other statuses MUST NOT trigger the rule.
- **FR-008**: If the history of an individual issue cannot be read, the system MUST show the current assignee (or empty) for that issue and MUST NOT fail the whole sprint load.
- **FR-009**: The new rule MUST be applied identically everywhere an Implementer is shown (the web dashboard sprint table and the MCP sprint issue results).
- **FR-010**: The change MUST NOT alter any other issue field, the label counts, story point totals, or status filtering.

### Key Entities

- **Issue History**: The ordered record of changes to an issue (status changes and assignee changes, each with a time), plus the assignee the issue was created with.
- **Status Move**: A history entry where the issue's status becomes "In Progress" or "In Review."; the first of each is the reference point for the rule.
- **Assignee In Effect**: The person assigned at a given moment after applying all assignee changes up to and including that moment.
- **Implementer**: The resolved display name for an issue according to FR-001 to FR-003.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For a sample of at least 10 issues from a closed sprint with varied histories (reassigned, unassigned at start, skipped "In Progress"), 100% show the Implementer that a manual reading of the issue history gives under this rule.
- **SC-002**: In a fully closed sprint, the number of issues with an empty Implementer is no greater than the number of issues that never had any assignee.
- **SC-003**: Loading a sprint's issues takes no more than 10% longer than before the change.
- **SC-004**: 100% of issue rows render a complete Implementer cell (a name or "—"); no row shows an error or a missing cell.

## Assumptions

- The status names in this Jira workflow are "In Progress" and "In Review." (with the trailing period); other in-flight statuses do not qualify.
- This rule replaces the rule in spec 006 and the current behavior (which uses the assignee at the first move to "In Review." and falls back to the current assignee).
- When neither moment yields a person, the existing last resort stays: the current assignee, then empty. The request did not say otherwise.
- "Before the move" includes an assignment made in the same change as the move, because that person is the one who took the work; this matches how workflows that auto-assign on "Start progress" behave.
- Sprint ST26.20 currently shows an empty Implementer for roughly half of its issues; part of the cause is likely issues whose assignee was never changed after creation (FR-004). This is to be confirmed during planning.
- The rule applies wherever an Implementer is already shown; no new column or output is added, and Word and CSV exports (which have no Implementer) are unchanged.
- Issue history is already available through the existing Jira connection used for sprint issues.
