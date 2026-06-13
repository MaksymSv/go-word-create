# Research: Implementer Column

**Branch**: `006-implementer-column` | **Date**: 2026-06-08

## Decision 1: How to retrieve the assignee at first "In Progress" transition

**Decision**: Fetch sprint issues via `client.Issue.Search` with JQL `sprint = {sprintID}` and `SearchOptions.Expand = "changelog"`, then derive the implementer from the embedded changelog.

**Rationale**: A single batch call returns all issues with changelog data in one round-trip. The existing `Sprint.GetIssuesForSprint` call does not expose `SearchOptions`, so we cannot pass `Expand: "changelog"` through it — a JQL search is the correct alternative. The `GetIssuesInProgressDuringMonth` function in the same package already uses this exact pattern (JQL + `Expand: "changelog"`), confirming it works with the project's go-jira version.

**Alternatives considered**:
- Per-issue API call (`/rest/api/3/issue/{key}?expand=changelog`) — rejected because it creates N+1 API calls, degrading performance significantly for sprints with 30-50+ issues.
- `Sprint.GetIssuesForSprint` with changelog — rejected because the go-jira wrapper does not accept search options for this method.

## Decision 2: Assignee resolution algorithm

**Decision**: Walk the issue changelog in chronological order, tracking the most recently observed assignee. At the first history entry where `status.toString == "In Progress"` (case-insensitive), record the currently tracked assignee as the implementer. If no "In Progress" transition is found, fall back to `issue.Fields.Assignee.DisplayName`. If that is also nil, return an empty string (rendered as "—" in the UI).

**Rationale**: This algorithm correctly handles:
- The common case: assignee set before being moved to "In Progress" (current assignee is the implementer)
- Reassignment after "In Progress": only the assignee at the first transition counts
- Multiple "In Progress" entries (re-opened issues): only the first counts (FR-005)
- Issue never moved to "In Progress": current assignee is used (FR-003)
- Unassigned issues: empty string / "—" (FR-004)

**Alternatives considered**:
- Using `history.Author.DisplayName` (who made the transition) — rejected because the spec requires the **assignee**, not the transition author (a scrum master might move tickets on behalf of developers).
- Fetching a point-in-time snapshot from Jira — rejected; not available via the go-jira REST API without the changelog expand.

## Decision 3: New service method vs. modifying existing method

**Decision**: Add a new `JiraService` method `GetIssuesFromSprintWithChangelog` that returns `[]Issue` with the `Implementer` field populated. The existing `LoadIssuesFromSprint` method is left unchanged.

**Rationale**: `LoadIssuesFromSprint` is used by other code paths (month report, sprint report generation) that do not need changelog data. Changing its signature or adding a flag parameter would add unused complexity to those call sites. A new method keeps concerns clean and avoids unintended side-effects on Word document generation.

**Alternatives considered**:
- Modify `LoadIssuesFromSprint` with an `includeChangelog bool` parameter — rejected because it widens the interface of an internal method without benefit to existing callers.
- Add `resolveImplementer` as a public method with per-issue calls — rejected (N+1 problem, see Decision 1).

## Decision 4: Where to add the `Implementer` field in `Issue` struct

**Decision**: Add `Implementer string` to `jiraservice.Issue`. Existing callers that don't populate it will see an empty string, which is a safe zero value.

**Rationale**: The `Issue` struct is the canonical issue representation. Adding the field there makes it available to any future call site without extra plumbing. The empty-string zero value is harmless — Word report code ignores fields it doesn't read.

**Alternatives considered**:
- Define a separate `IssueWithImplementer` struct — rejected as unnecessary duplication.
