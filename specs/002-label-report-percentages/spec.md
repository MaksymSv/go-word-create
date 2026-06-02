# Feature Specification: Label Report Percentage Columns

**Feature Branch**: `002-label-report-percentages`

**Created**: 2026-06-02

**Status**: Draft

**Input**: User description: "I want to extend funtionaly of sprint-label-report. I need to calculate percentage of issues marked with different labels to total count of issues/SP in the sprint. So, the list of columns shlould be Label | Count | Count,% | Total SP | Total SP,%. There also final (Total) line shoulbe added with totals by % columns. Do not add totals by Count and Total SP as they don't have practical sense."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Percentage Columns in Short-Format Report (Priority: P1)

A Scrum Master or delivery lead wants to see not only the raw count and story points for
each label group, but also what percentage of the total sprint issues and total sprint
story points each label group represents — to quickly understand the relative weight of
AI-assisted work in a sprint.

**Why this priority**: This is the primary deliverable of the extension. The percentage
columns provide the proportion context that raw numbers alone cannot communicate. The
short-format report is the most-used view, so it must be enhanced first.

**Independent Test**: Run the short-format report against a sprint with known label
distribution; verify that each row shows correct Count,% and Total SP,% values, and that
the "Total" row at the bottom sums those percentage columns correctly.

**Acceptance Scenarios**:

1. **Given** a sprint with 10 total issues (8 SP total) and a label group `ai-assisted`
   containing 4 issues (5 SP), **When** the short-format report is generated, **Then**
   the `ai-assisted` row shows `Count,% = 40%` and `Total SP,% = 62.5%`.

2. **Given** multiple label groups in a component, **When** the short-format report is
   generated, **Then** the last row is a "Total" row that shows the sum of all Count,%
   values and the sum of all Total SP,% values; the Count and Total SP cells of the
   "Total" row are empty.

3. **Given** a label group with 0 matching issues, **When** the short-format report is
   generated, **Then** the row shows `Count,% = 0%` and `Total SP,% = 0%`.

4. **Given** a sprint with 0 total story points (all issues unestimated), **When** the
   report is generated, **Then** `Total SP,%` is shown as `0%` for all rows rather than
   producing a division-by-zero error.

---

### User Story 2 - Percentage Columns in Full-Format Report (Priority: P2)

A tech lead or QA manager using the full-format (per-issue) report also needs to see the
same percentage context per label group — so they can compare the weight of each group
while drilling into individual issues.

**Why this priority**: Extends the percentage feature to the full-format view. The
full-format report already repeats group-level data (count, total SP) on every issue row;
the percentage columns should follow the same pattern.

**Independent Test**: Run the full-format report; confirm Count,% and Total SP,% appear
on every issue row within a label group with values consistent with the short-format
report; confirm the "Total" row appears at the bottom of each component's table.

**Acceptance Scenarios**:

1. **Given** a label group `ai-assisted` with 2 issues in a sprint of 10 total issues,
   **When** the full-format report is generated, **Then** both issue rows within that
   group show `Count,% = 20%` and the correct Total SP,% value.

2. **Given** a component's full-format table, **When** the report is generated, **Then**
   a "Total" row appears at the bottom with the sum of all Count,% values and the sum of
   all Total SP,% values; Count and Total SP cells are empty in this row.

---

### Edge Cases

- What happens when total sprint issues is 0?
  All percentage values are shown as `0%`; no division-by-zero error occurs.
- What happens when total sprint story points is 0 (all unestimated)?
  `Total SP,%` is shown as `0%` for all rows.
- What happens when a label group count exceeds 100% (non-exclusive groups, double-counting)?
  Percentages above 100% are displayed as-is — this is expected and intentional since
  label groups are non-exclusive (an issue with two configured labels is counted in both).
- What happens when percentage totals in the "Total" row exceed 100%?
  Displayed as-is for the same reason — the "Total" row is a mathematical sum of
  per-label percentages, not a percentage of the whole.
- How are percentages rounded?
  Rounded to one decimal place (e.g., `33.3%`). If the value is a whole number it is
  displayed without a trailing zero (e.g., `40%` not `40.0%`).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST display a `Count,%` column in both short-format and full-format
  label aggregation tables, showing each label group's issue count as a percentage of the
  total number of issues in the sprint component being reported.
- **FR-002**: System MUST display a `Total SP,%` column in both short-format and
  full-format label aggregation tables, showing each label group's total story points as a
  percentage of the total story points across all sprint issues in the component.
- **FR-003**: Percentage denominators MUST be the total count and total SP of all issues
  in the component (labeled + unlabeled combined).
- **FR-004**: System MUST append a "Total" summary row at the bottom of each label
  aggregation table; the "Total" row MUST contain the sum of all `Count,%` values and the
  sum of all `Total SP,%` values; the `Count` and `Total SP` cells in this row MUST be
  left empty.
- **FR-005**: When total issue count for a component is zero, `Count,%` MUST be displayed
  as `0%` for all rows (no division-by-zero error).
- **FR-006**: When total story points for a component is zero, `Total SP,%` MUST be
  displayed as `0%` for all rows (no division-by-zero error).
- **FR-007**: Percentage values MUST be rounded to one decimal place; whole-number values
  MUST be displayed without a trailing decimal (e.g., `40%` not `40.0%`).
- **FR-008**: In full-format output, `Count,%` and `Total SP,%` MUST be repeated on every
  issue row within a label group (same pattern as `Count` and `Total SP`).
- **FR-009**: The console/debug output MUST reflect the same percentage columns and
  "Total" row as the Word document output.

### Key Entities

- **Label Group** (extended): Now also carries `CountPct float64` and `TotalSPPct float64`
  representing the percentage of this group's count and SP against component totals.
- **Component Report** (extended): The rendering layer needs component-level totals
  (total issue count and total SP across all issues in the component) to compute
  percentages; these are derived at render time or pre-computed in the aggregation layer.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can generate a short-format or full-format label report for any sprint
  and see correctly computed percentage columns on every label row without any additional
  configuration changes.
- **SC-002**: The "Total" row at the bottom of each component table shows the exact
  arithmetic sum of the `Count,%` and `Total SP,%` column values for that component.
- **SC-003**: No division-by-zero errors occur for sprints where total issues or total
  story points is zero — the tool exits cleanly with `0%` displayed.
- **SC-004**: Percentage values in the output are consistently rounded to one decimal
  place, and whole-number percentages are displayed without a trailing decimal point.
- **SC-005**: The console/debug output and the Word document output contain identical
  percentage data for the same sprint input.

## Assumptions

- The percentage denominator is the total issues/SP in the component (not across all
  components in the sprint), consistent with how component reports are already isolated.
- Label groups remain non-exclusive; percentages above 100% per label or in the "Total"
  row are acceptable and expected.
- No new configuration is required for this feature — percentages are always displayed
  when using the sprint-label-report command (they are not optional).
- The existing `LabelGroup` struct will be extended with `CountPct` and `TotalSPPct`
  fields; the `Aggregate` function signature will be updated to accept or compute
  component totals.
- The "Total" row label text is the string `"Total"`.
- The feature applies to both the Word document output and the console/debug output.
- Formatting of percentages (one decimal, no trailing zero for whole numbers) is applied
  consistently in both rendering modes.
