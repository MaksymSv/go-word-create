# Research: Sprint Label Aggregation Report

**Feature**: Sprint Label Aggregation Report
**Branch**: `001-sprint-label-report`
**Date**: 2026-05-29

## How go-jira exposes Jira Labels

**Decision**: Use `issue.Fields.Labels []string` from `go-jira` directly.

**Rationale**: The `jira.IssueFields` struct in `github.com/andygrunwald/go-jira`
includes a standard `Labels []string` field that maps to Jira's built-in Labels field.
No custom field ID is required — unlike Story Points or Epic Link, Labels is a
first-class field in the Jira REST API v2 response.

**Alternatives considered**:
- Using `issue.Fields.Unknowns["labels"]` — unnecessary; Labels is a standard field
  already available as `issue.Fields.Labels`.

**Impact on implementation**: `LoadIssuesFromSprint` in `internal/jiraservice/jira.go`
must copy `issue.Fields.Labels` into the `Issue.Labels []string` field (new field to add
to the `Issue` struct).

---

## How go-jira exposes Jira Components

**Decision**: Use `issue.Fields.Components []*jira.Component` where `Component.Name`
is the team/component name string.

**Rationale**: `jira.IssueFields.Components` is a standard field in go-jira. Each
`Component` struct has a `Name` string. This is the same field used by the month-issues
module for component-based filtering (see `GetIssuesInProgressDuringMonth`).

**Alternatives considered**:
- Fetching components separately via the Jira Components API — unnecessary overhead; the
  component data is already embedded in each issue's response.

**Impact on implementation**: `Issue` struct needs `Components []string` field; populate
by iterating `issue.Fields.Components` and extracting `c.Name` for each non-nil entry.

---

## Label Aggregation Strategy

**Decision**: Build an independent `internal/labelreport` package with a pure
`Aggregate(issues []Issue, orderedLabels []string)` function.

**Rationale**: Keeping aggregation logic in a dedicated package with no Jira or Word
imports allows isolated unit testing (no Jira mock needed) and respects the Package
Separation principle. The function produces `[]LabelGroup` (in configured order) and
`[]Issue` (unlabeled remainder).

**Alternatives considered**:
- Inline aggregation in `cmd/main.go` — violates Package Separation principle; not
  independently testable.
- Adding aggregation methods to `jiraservice` — creates undesired coupling; jiraservice
  should only concern itself with Jira API calls.

**Multi-label handling**: An issue with multiple configured labels appears in every
matching `LabelGroup` independently (label groups are non-exclusive, per FR-005).
An issue appears in the unlabeled set only if it matches NONE of the configured labels.

---

## Component Grouping Strategy

**Decision**: Fetch all sprint issues without a component filter using the existing
`GetSprintIssues` method; group client-side by the component names in each `Issue`.

**Rationale**: The existing `GetSprintIssues` already correctly fetches all sprint
issues. Adding a `Components []string` field to `Issue` allows the `cmd/` layer (or
a helper in `internal/labelreport`) to group issues by component before aggregating
labels per component. This avoids N+1 Jira requests (one per component) and keeps the
Jira service layer unchanged in behavior.

**Alternatives considered**:
- A new `GetSprintIssuesByComponent` method in jiraservice — extra API round-trips and
  unnecessary complexity; the data is already available in a single sprint fetch.

**Component assignment**: An issue may belong to multiple Jira components. For reporting
purposes, an issue appears once under each component it belongs to (same non-exclusive
logic as labels). If an issue has no component, it appears under a synthetic "No Component"
group to ensure FR-002 (no issues silently omitted) is satisfied.

---

## Report Format Flag

**Decision**: Use a `-format` CLI flag with values `short` (default) and `full`.

**Rationale**: Mirrors the existing `-debug` pattern; keeps the binary simple. Short
format is the default since it is the P1 use case (summary overview). Full format adds
per-issue rows within each label group.

**Alternatives considered**:
- Two separate binaries (one per format) — unnecessary duplication; the difference is
  only in the rendering step, not the data pipeline.
- `-short` / `-full` boolean flags — less ergonomic; mutual exclusion requires extra
  validation. A single `-format` string flag is unambiguous.

---

## Configuration of Label List

**Decision**: Read from `REPORT_LABELS` environment variable (comma-separated, ordered);
default to `ai-assisted,ai-assisted-ba,ai-assisted-dev,ai-assisted-qa` if unset.

**Rationale**: Consistent with the existing `TEAMS` env-var pattern. The default list
covers the initial release requirement (FR-010) while allowing override without code
changes (SC-005).

**Impact on implementation**: Add `ReportLabels []string` to `internal/config/Config`;
parse from `REPORT_LABELS` env var with default fallback (unlike `TEAMS`, not required).

---

## All NEEDS CLARIFICATION items resolved

No NEEDS CLARIFICATION markers were present in the spec. All design decisions above were
inferred from the codebase and documented for implementation reference.
