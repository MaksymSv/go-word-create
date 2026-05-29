# Feature Specification: Sprint Label Aggregation Report

**Feature Branch**: `001-sprint-label-report`

**Created**: 2026-05-29

**Status**: Draft

**Input**: User description: "I need to create a feature that will collect the workitems in Jira from a specified sprint (same like get-sprint-issues), however I need a different output. I need to calculate issues (count and sum of story points) with a particular label assigned. In short format we need to see only: label name | issues count | sum of story points. In full format I need to see all issues in this group (line by line) like: label name | issues count | sum of story points | issue key | Summary | issue SP. The list of labels to aggregate should be configured. The order of labels is important. Provide the report in that configured order. I also need to list all the rest of issues not labeled with any of those configured labels in a table: Issue key | Summary | SP. Please keep high-level processing flow by jira components like in get-sprint-issues module. The list of labels for start: ai-assisted, ai-assisted-ba, ai-assisted-dev, ai-assisted-qa."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Sprint Label Summary Report (Priority: P1)

A Scrum Master or delivery lead wants a quick, at-a-glance summary of how many sprint
issues — and how many story points — are tagged with each AI-related label, to track
AI adoption across the team in a given sprint.

**Why this priority**: This is the core deliverable; without label aggregation the feature
has no value. The summary (short) format is the most frequently needed view.

**Independent Test**: Run the tool against a sprint that contains issues with known labels;
the output document contains one summary table with one row per configured label, showing
the correct issue count and story-point total for each label.

**Acceptance Scenarios**:

1. **Given** a sprint with 3 issues labeled `ai-assisted` (2 SP, 3 SP, 5 SP) and 2 issues
   labeled `ai-assisted-dev` (1 SP, 4 SP), **When** the short-format report is generated,
   **Then** the summary table shows `ai-assisted | 3 | 10` and `ai-assisted-dev | 2 | 5`
   in the configured label order.

2. **Given** a configured label with no matching issues in the sprint, **When** the report
   is generated, **Then** a row for that label still appears with a count of 0 and 0 SP.

3. **Given** a sprint with issues across multiple Jira components, **When** the report is
   generated, **Then** label aggregation is performed separately per component, producing
   one summary table per component.

---

### User Story 2 - Detailed Per-Issue Breakdown (Priority: P2)

A tech lead or QA manager needs to see exactly which issues contribute to each label
group — including their keys, summaries, and individual story points — to verify the
categorization and drill into specifics.

**Why this priority**: Important for auditability and spot-checks, but the summary view
(US1) must be established first.

**Independent Test**: Run the full-format report against a sprint; each label group shows
its header data (label name, count, total SP) on every issue row, and every labeled issue
key appears in its matching group(s).

**Acceptance Scenarios**:

1. **Given** a label group `ai-assisted` containing issues PROJ-10 (3 SP) and PROJ-11
   (5 SP), **When** the full-format report is generated, **Then** the table contains two
   rows: `ai-assisted | 2 | 8 | PROJ-10 | <summary> | 3` and
   `ai-assisted | 2 | 8 | PROJ-11 | <summary> | 5`.

2. **Given** an issue that carries two configured labels (e.g., `ai-assisted` AND
   `ai-assisted-dev`), **When** the full-format report is generated, **Then** the issue
   appears in both label groups independently.

3. **Given** a sprint with no issues for a configured label, **When** the full-format
   report is generated, **Then** the label group header row still appears with count 0
   and 0 SP, and no issue detail rows are shown for that group.

---

### User Story 3 - Unlabeled Issues Table (Priority: P2)

A project manager needs to identify sprint issues that carry none of the configured
labels, to flag unclassified work and ensure full sprint coverage in AI reporting.

**Why this priority**: Equally important as the detailed breakdown; ensures no sprint work
is silently omitted from the report.

**Independent Test**: Run the report against a sprint with some issues that lack all
configured labels; a separate table lists those issues by key, summary, and SP.

**Acceptance Scenarios**:

1. **Given** two issues in the sprint that have no configured labels, **When** the report
   is generated, **Then** a separate table lists those issues with key, summary, and SP.

2. **Given** an issue that has at least one configured label, **When** the report is
   generated, **Then** that issue does NOT appear in the unlabeled issues table.

3. **Given** all sprint issues have at least one configured label, **When** the report is
   generated, **Then** the unlabeled issues table is either omitted or clearly shown empty.

---

### Edge Cases

- What happens when an issue has a story-points value of zero or is unestimated?
  It MUST still be counted and contribute 0 to the sum.
- What happens when the configured label list is empty?
  All issues fall into the unlabeled table and the user receives a warning.
- What happens when the sprint has no issues at all?
  Each label group shows count 0 and the unlabeled table is empty.
- What happens when an issue has multiple configured labels?
  It appears in each matching label group independently (groups are not mutually exclusive).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST accept a sprint name and a report-format mode (short or full)
  and produce a formatted sprint label report for that sprint.
- **FR-002**: System MUST read an ordered list of target labels from configuration;
  the configured label order MUST determine the display order in the report.
- **FR-003**: In short format, the report MUST contain one row per configured label
  showing: label name, issue count, and total story points.
- **FR-004**: In full format, the report MUST contain one row per issue within each label
  group; every row MUST include: label name, group issue count, group total SP, issue key,
  issue summary, and issue story points.
- **FR-005**: Label groups MUST be independent: an issue with multiple configured labels
  MUST appear in every matching group, contributing its story points to each group total.
- **FR-006**: System MUST produce a separate unlabeled issues table listing every sprint
  issue that carries none of the configured labels, with columns: issue key, summary, SP.
- **FR-007**: Report content MUST be organized by Jira component at the top level; each
  component section MUST contain that component's label aggregation tables.
- **FR-008**: System MUST validate at startup that all required configuration is present
  and exit with a clear, actionable error message if any required value is missing.
- **FR-009**: System MUST support a console/debug mode that prints the report to the
  terminal instead of generating a Word document.
- **FR-010**: The default ordered label list for the initial release MUST be:
  `ai-assisted`, `ai-assisted-ba`, `ai-assisted-dev`, `ai-assisted-qa`.

### Key Entities

- **Label Group**: A named aggregation of sprint issues sharing one configured label.
  Attributes: label name (position in ordered config list), issue count, total SP,
  list of matching issues.
- **Sprint Issue**: A work item from Jira. Attributes: key, summary, story points,
  labels (multi-value), Jira component.
- **Label Configuration**: An ordered list of label strings used to group sprint issues.
- **Unlabeled Set**: The subset of sprint issues matching none of the configured labels.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can generate a short-format label summary for any sprint and receive
  a correctly formatted document in under 30 seconds for sprints of up to 200 issues.
- **SC-002**: Every sprint issue appears either in the unlabeled table or in one or more
  label-group sections — no issues are silently omitted.
- **SC-003**: The label order in the generated report matches the configured order 100%
  of the time.
- **SC-004**: Running the tool in console/debug mode produces output that matches the
  content that would be written to the document.
- **SC-005**: Changing the label list configuration and regenerating the report produces
  updated groupings without any code changes.

## Assumptions

- The sprint name provided matches the sprint name in Jira exactly (same case-sensitivity
  rules as the existing sprint-issues command).
- Story points are stored as a numeric value; issues with no estimate are treated as 0 SP.
- Jira labels are compared case-sensitively against the configured label list.
- Label groups are not mutually exclusive: an issue with multiple matching labels appears
  in every relevant group; story-point double-counting across groups is intentional.
- The default label list (`ai-assisted`, `ai-assisted-ba`, `ai-assisted-dev`,
  `ai-assisted-qa`) can be overridden via configuration without code changes.
- Document formatting (fonts, margins, header colours) follows the project's established
  Word document standard.
- Jira component is used as the top-level grouping dimension, consistent with the
  existing month-issues report architecture.
- The feature is delivered as a standalone command (separate binary), consistent with
  how other report commands are structured in this project.
