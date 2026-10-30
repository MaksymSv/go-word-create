# Research: Implementer Resolution Rule

**Branch**: `010-implementer-resolution` | **Date**: 2026-10-30

No open clarifications remain in the spec. The decisions below settle the technical unknowns.

## Decision 1: How to recover the assignee that was set at creation

**Decision**: Use the `fromString` of the first `assignee` item in the changelog as the creation-time assignee. If the changelog never changed the assignee, use the issue's current assignee.

**Rationale**: Jira does not write a changelog entry when an issue is created with an assignee; only later changes appear, and each change records the previous value in `fromString`. The first change's `fromString` is therefore the creation-time assignee. If there was no change at all, the current assignee is the creation-time assignee. This is the cause of the empty values today: the current code starts from an empty string and only learns an assignee from `toString`, so an issue assigned at creation and never reassigned yields nothing (the existing test `full_flow_assigned_at_creation_no_history_assignee_change` documents this). It needs no extra requests.

**Alternatives considered**:
- Fetch each issue separately to read its creation-time fields: rejected, one request per issue and Jira still would not expose the creation-time assignee directly.
- Use the current assignee whenever the changelog has no assignee entry before the move: rejected, wrong when the assignee changed only after the move (the first change's `fromString` is the right value then).

## Decision 2: Order of assignee and status items inside one history entry

**Decision**: Apply all `assignee` items of an entry before evaluating its `status` items.

**Rationale**: A workflow that assigns the person at the moment of "Start progress" produces both items in one entry. The spec (FR-005) counts that assignee as the one who held the issue. This is also how the current code behaves, so it is not a behavior change.

**Alternatives considered**: Evaluate status first, then assignee ("strictly before"): rejected, such issues would always fall through to the "In Review." fallback even though the person clearly started the work.

## Decision 3: "First" move handling when the assignee was empty

**Decision**: Only the first move to "In Progress" counts. If the assignee was empty at that moment, later "In Progress" moves are ignored and the "In Review." fallback applies.

**Rationale**: FR-001/FR-002 define the primary moment as the first move. Using a later move would change the meaning of "first" and could pick up a reviewer or tester after a bounce back from QA.

**Alternatives considered**: Take the first "In Progress" move that has an assignee: rejected, contradicts the user's wording ("before the ticket moved to In Progress").

## Decision 4: Where the status names live

**Decision**: Two unexported constants in `internal/jiraservice`: `"In Progress"` and `"In Review."`, compared with `strings.EqualFold`.

**Rationale**: The existing code already hardcodes "in review." and, in `GetIssuesInProgressDuringMonth`, "In Progress". Configuration would be new surface for names that are fixed in this workflow. The deviation from Principle II is recorded in the plan's Complexity Tracking.

**Alternatives considered**: `IMPLEMENTER_*_STATUS` environment variables: rejected for now (see above), easy to add later because the names are isolated in constants.

## Decision 5: Scope of change

**Decision**: Change only `resolveImplementer` (and tests and docs). Keep `GetIssuesFromSprintWithChangelog`, `sprintview`, the dashboard and the MCP tools as they are.

**Rationale**: `Issue.Implementer` is produced in one place and passed through unchanged, so one function change applies the rule on every surface (FR-009). The month report and the Word/CSV outputs never read `Implementer`.

**Alternatives considered**: A separate resolver package: rejected, a single private function does not justify a package.

## Decision 6: History order and completeness

**Decision**: Keep trusting the order and content of the changelog as returned by the search call (oldest first), without re-sorting or re-fetching.

**Rationale**: The current implementation relies on the same assumption and works for the sprints in use. Adding sorting by timestamp needs parsing Jira's time format for no observed problem.

**Alternatives considered**: Sort by the entry's `created` time: deferred; worth doing only if out-of-order histories are ever observed. Truncated histories are listed as a risk in the plan.
