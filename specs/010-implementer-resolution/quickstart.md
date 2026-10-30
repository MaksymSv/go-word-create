# Quickstart: Implementer Resolution Rule

**Branch**: `010-implementer-resolution` | **Date**: 2026-10-30

## What changes

How the Implementer value is computed (`resolveImplementer` in `internal/jiraservice/jira.go`). No new flags, variables, endpoints or columns.

## Verify with tests (no Jira needed)

```bash
go test ./internal/jiraservice/ -run TestResolveImplementer -v
make fmt && make lint && make test && make build
```

## Verify against real data

Baseline before the change (sprint ST26.20, STABLETEK, sprint ID 950218, read through the MCP `get_sprint_issues` tool on 2026-10-30): 35 issues, 15 with an empty implementer (CNPYBCSN-1750, 1757, 1800, 1805, 1820, 1821, 1894, 1897, 1900, 1911, 1913, 1935, 1940, 1941, 1947).

1. Rebuild and restart: `make build`, then `/mcp` reconnect (or restart the dashboard).
2. Call `get_sprint_issues` for sprint 950218 (or open the dashboard, team STABLETEK, sprint ST26.20).
3. Expect fewer empty implementers: only issues that never had an assignee should remain empty (SC-002).
4. Pick at least 10 issues with varied histories (reassigned, unassigned at start, skipped "In Progress") and compare each with its Jira history (SC-001).

## Notes

- Past sprints show different implementers after the change; that is intended.
- Other outputs (CSV, Word, label report, month report) are unchanged.
