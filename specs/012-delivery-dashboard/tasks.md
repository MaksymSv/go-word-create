# Tasks: Project delivery dashboard

**Input**: [spec.md](spec.md), [plan.md](plan.md) (retroactive: all tasks done on 2026-11-11 unless noted)

## Phase 1: Research

- [x] T001 Resolve both teams in `delivery_central` and list their metrics (2 of 7 charts, 1 of 2 teams)
- [x] T002 Probe Jira: board velocity chart (7-sprint cap), `sprint WAS` JQL (unsupported), changelog format of the Sprint field, resolutions and bug priorities in use
- [x] T003 Validate bug and lead-time definitions against Delivery Central for PROCESSING

## Phase 2: Backend

- [x] T004 `internal/jiraservice/delivery.go`: `ClosedSprints`, `BoardVelocity`, `SearchHistoryIssues` + tests (`delivery_test.go`)
- [x] T005 `internal/config`: `DELIVERY_*` settings + parsers + tests
- [x] T006 `internal/deliverystats`: types, pure metric functions, disk cache, service with per-team lock and TTL + tests
- [x] T007 `cmd/get-delivery-report` and Makefile targets (`build-delivery-report`, `run-delivery-report`, `make help`)

## Phase 3: Web

- [x] T008 `internal/dashboard/delivery.go`: routes, settings endpoint, startup warm-up + handler tests
- [x] T009 `charts.js`: columns, lines, reference line, hover tooltip, legend, direct end labels, table view, resize
- [x] T010 `delivery.html/.css/.js`: team tabs, sprint-count pills, refresh, per-card error states, shared theme and page nav
- [x] T011 Headless-Chrome checks: light and dark, both teams, 6 and 12 sprints

## Phase 4: Docs and gates

- [x] T012 `.env.example` (all variables), README, AGENTS.md, `.gitignore` (`.cache/`)
- [x] T013 `make fmt`, `make lint`, `make test`, `make build`

## Follow-ups (not done)

- [ ] Cycle time: find which end status Delivery Central uses and offer it as a setting
- [ ] Attribute logged bugs by component at creation (closes the 1 to 2 bug gap vs Delivery Central)
- [ ] Expose `get_delivery_report` as an MCP tool if LLM clients need it
- [ ] Browser-level tests once the repo adopts a JS test setup
