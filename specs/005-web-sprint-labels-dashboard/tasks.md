# Tasks: Web Sprint Labels Dashboard

**Input**: Design documents from `specs/005-web-sprint-labels-dashboard/`

**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/api.md ✅

**Organization**: Tasks are grouped by user story. Phase 2 (Foundational) must be complete before any user story work begins.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US3)

---

## Phase 1: Setup

**Purpose**: Create directory structure so all subsequent tasks have target directories.

- [x] T001 Create directory structure: `mkdir -p cmd/web-sprint-labels-report internal/dashboard web/icons`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Jira service extensions, shared API types, handler skeleton, static asset placeholders, binary entrypoint, and Makefile targets — all must be in place so the binary compiles before user story work begins.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T002 Add `GetSprintsForBoard(boardName string, limit int) ([]jira.Sprint, error)` to `internal/jiraservice/jira.go`: call `GetBoard(boardName)` to get board ID, then paginate `GetAllSprintsWithOptions(boardID, &GetAllSprintsOptions{State:"active,closed", SearchOptions:{StartAt:0, MaxResults:50}})` until `SprintsList.IsLast == true`, collect all sprints into a slice, sort by `ID` descending, return the first `limit` entries (or all if fewer than `limit`)

- [x] T003 [P] Create `internal/dashboard/api.go` in package `dashboard` with the following exported types: `TeamResponse{ComponentName, BoardName string}`; `SprintResponse{ID int, Name, State string, StartDate, EndDate *time.Time}`; `DashboardIssue{Key, Summary, Epic string, StoryPoints float64, Type, Status, URL string, ActiveLabels []string}`; `SprintIssuesResponse{ConfiguredLabels []string, Issues []DashboardIssue}`; `LabelUpdateRequest{Action, Label string}`; all fields must use `json:"..."` tags matching the contracts/api.md field names (camelCase)

- [x] T004 [P] Create all 6 SVG icon files in `web/icons/` so the `//go:embed web` directive succeeds at compile time: `bug.svg` (red `#E5493A` filled circle with a bug/beetle symbol), `story.svg` (green `#63BA3C` filled bookmark/page shape), `task.svg` (blue `#4BADE8` filled checkbox/tick square), `subtask.svg` (blue `#4BADE8` smaller version of task icon), `epic.svg` (purple `#904EE2` filled lightning-bolt shape), `issue.svg` (grey `#8993A4` filled diamond/circle generic icon) — each file must be a valid standalone SVG with `viewBox="0 0 16 16"` and `xmlns` attribute

- [x] T005 Create `internal/dashboard/handler.go` in package `dashboard`: declare `//go:embed web` with an `embed.FS` named `webFiles`; define `DashboardHandler{cfg *config.Config, jira *jiraservice.JiraService, webFS fs.FS}` struct; `NewHandler(cfg, jira)` constructor that calls `fs.Sub(webFiles, "web")` to get a rooted sub-FS; `Routes() http.Handler` method that creates a `http.NewServeMux()`, registers `GET /static/` → `http.StripPrefix("/static/", http.FileServerFS(h.webFS))`, `GET /` → `h.serveIndex`, `GET /api/teams` → `h.getTeams`, `GET /api/teams/{component}/sprints` → `h.getSprints`, `GET /api/sprints/{sprintID}/issues` → `h.getSprintIssues`, `POST /api/issues/{issueKey}/labels` → `h.updateLabel`; wraps the mux in a logging middleware that writes `method path status duration` to stderr; implements `serveIndex` (serves `index.html` from webFS); stubs `getTeams`, `getSprints`, `getSprintIssues`, `updateLabel` as `http.Error(w, "not implemented", 501)` (depends on T003 for API types; depends on T004 for web/ to be non-empty)

- [x] T006 Create `cmd/web-sprint-labels-report/main.go`: parse `-port int` (default 8080) and `-debug bool` flags; call `config.Load()` (fatal on error); call `jiraservice.NewJiraService(cfg.JiraURL, cfg.JiraUsername, cfg.JiraAPIToken, cfg.JiraEpicField, cfg.JiraSPField)` (fatal on error); call `dashboard.NewHandler(cfg, jiraSvc)` to get a handler; log `"Starting dashboard on http://localhost:{port}"`; call `log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), handler.Routes()))` (depends on T005)

- [x] T007 Update `Makefile`: add `WEB_DASHBOARD_CMD=./cmd/web-sprint-labels-report`; add `build-web-dashboard` target that runs `go build -o $(OUTPUT_DIR)/web-sprint-labels-report $(WEB_DASHBOARD_CMD)` with mkdir and echo; add `run-web-dashboard: build-web-dashboard` target that runs `$(OUTPUT_DIR)/web-sprint-labels-report -port=$(or $(PORT),8080)`; update the `build` target to include `build-web-dashboard`; update `make help` echo lines to list the new targets

**Checkpoint**: `make build-web-dashboard` compiles cleanly; `./bin/web-sprint-labels-report` starts and serves a 501 page at `/api/teams`.

---

## Phase 3: User Story 1 — Browse Sprint Issues by Team (Priority: P1) 🎯 MVP

**Goal**: Team buttons and sprint buttons load in the toolbar; clicking a sprint populates a 7-column issues table.

**Independent Test**: Start `./bin/web-sprint-labels-report`; open `http://localhost:8080` in a browser; verify team buttons appear, first is pre-selected, 5 sprint buttons appear; click a sprint button; verify a table with columns Type|Key|Summary|Epic|SP|Status|Labels loads.

### Implementation for User Story 1

- [x] T008 [US1] Implement `getTeams` handler in `internal/dashboard/handler.go`: replace the 501 stub; iterate `h.cfg.Teams`, build `[]TeamResponse`; encode as JSON with status 200

- [x] T009 [US1] Implement `getSprints` handler in `internal/dashboard/handler.go`: replace the 501 stub; read component from `r.PathValue("component")`; look up team in `h.cfg.Teams` by ComponentName (404 if not found); call `h.jira.GetSprintsForBoard(team.BoardName, 5)`; on Jira error return 502 with `{"error":"..."}` including board name in context; map `[]jira.Sprint` → `[]SprintResponse`; encode as JSON (depends on T002 for GetSprintsForBoard)

- [x] T010 [US1] Implement `getSprintIssues` handler in `internal/dashboard/handler.go`: replace the 501 stub; parse `r.PathValue("sprintID")` as int (400 on parse error); call `h.jira.LoadIssuesFromSprint(sprintID, epicNames, emptyFilter)` — load epics via `h.jira.LoadEpics(h.cfg.ProjectKey)` first; compute `activeLabels` for each issue as the intersection of `issue.Labels` and `h.cfg.ReportLabels`; build `SprintIssuesResponse{ConfiguredLabels: h.cfg.ReportLabels, Issues: [...DashboardIssue]}`; encode as JSON (depends on T008 and T009 for handler.go context, but file is the same — implement sequentially)

- [x] T011 [P] [US1] Create `web/index.html`: full HTML5 document with `<meta charset="UTF-8">`, `<meta name="viewport" ...>`, `<title>Sprint Labels Dashboard</title>`, `<link rel="stylesheet" href="/static/style.css">`; inline `<script>` tag in `<head>` that reads `localStorage.getItem('sprint-dashboard-theme')` and sets `document.documentElement.setAttribute('data-theme', ...)` before paint (prevents flash); `<body>` with `<div id="toolbar">` containing `<div id="team-section">` (team buttons rendered here), `<div id="sprint-section">` (sprint buttons rendered here), `<button id="theme-toggle">🌙</button>` at far right; `<div id="main-area">` below toolbar; `<div id="error-banner" hidden>` for error notifications; `<script src="/static/app.js" defer></script>`

- [x] T012 [P] [US1] Create `web/style.css`: CSS reset (box-sizing border-box, margin 0); CSS custom properties for light theme on `:root` (--color-bg, --color-surface, --color-text, --color-text-muted, --color-border, --color-btn-team, --color-btn-team-active, --color-btn-sprint, --color-btn-sprint-active, --color-btn-label-active, --color-btn-label-inactive, --color-btn-label-pending); `#toolbar` as flex row with padding, gap, align-items center, background --color-surface, border-bottom; `#team-section` and `#sprint-section` as flex row with gap; `#theme-toggle` with margin-left auto; button base styles (padding, border-radius, cursor, border, font); `.btn-team.active` using --color-btn-team-active; `.btn-sprint.active` using --color-btn-sprint-active; `#main-area` with padding; table styles (width 100%, border-collapse collapse); `th`/`td` with padding, border, text-align left; `thead tr` with background --color-surface; `.issue-icon` as 16×16 img; `#error-banner` with red/warning colors; `.empty-state` centered muted text

- [x] T013 [US1] Create `web/app.js`: define `ICON_MAP` object mapping issue type names to `/static/icons/*.svg` paths with `issue.svg` fallback; define module-level state `{ selectedTeam: null, selectedSprint: null }`; `fetchTeams()` → GET /api/teams → `renderTeamButtons(teams)` which clears `#team-section`, creates a button per team with class `btn-team`, sets `data-component`, attaches click handler, auto-clicks the first button; `onTeamClick(component)` marks button active, calls `fetchSprints(component)`, clears `#main-area`; `fetchSprints(component)` → GET `/api/teams/${component}/sprints` → `renderSprintButtons(sprints)` which clears `#sprint-section`, creates a button per sprint with class `btn-sprint`, `data-sprint-id`, attaches click handler; `onSprintClick(sprintId)` marks button active, calls `fetchIssues(sprintId)`; `fetchIssues(sprintId)` → GET `/api/sprints/${sprintId}/issues` → `renderTable(data)` which builds a `<table>` with `<thead>` row Type|Key|Summary|Epic|SP|Status|Labels, one `<tr>` per issue where Type cell is `<img class="issue-icon" src="${ICON_MAP[...]}">`, Key cell is `<a href="${issue.url}" target="_blank">${issue.key}</a>`, SP cell shows `—` when 0, Labels cell shows one `<button class="btn-label ${active?'active':''}" data-key="${key}" data-label="${lbl}">` per configured label; shows `.empty-state` div when issues array is empty; all fetch errors display in `#error-banner` (set textContent, remove `hidden`); call `fetchTeams()` on `DOMContentLoaded` (depends on T011 for HTML structure)

**Checkpoint**: US1 fully functional — open dashboard, see teams, click sprint, see issue table with icons, links, and label buttons (styling minimal at this point).

---

## Phase 4: User Story 2 — Assign Labels to Issues (Priority: P2)

**Goal**: Clicking a label button on an issue adds or removes that label in Jira; the button state reflects the result immediately.

**Independent Test**: Load any sprint, find an issue, click an inactive label button; verify button changes to active state; refresh page and verify label persists in Jira.

### Implementation for User Story 2

- [x] T014 [US2] Add `UpdateIssueLabel(issueKey, label string, add bool) error` to `internal/jiraservice/jira.go`: build the Jira update payload `map[string]interface{}{"update": map[string]interface{}{"labels": []map[string]string{{"add"|"remove": label}}}}` (use `"add"` when `add==true`, `"remove"` when `add==false`); call `s.client.Issue.UpdateIssue(issueKey, payload)`; return a wrapped error including issueKey and label in the message context if non-nil

- [x] T015 [US2] Implement `updateLabel` handler in `internal/dashboard/handler.go`: replace the 501 stub; decode request body as `LabelUpdateRequest`; validate `Action` is `"add"` or `"remove"` (400 otherwise); validate `Label` is present in `h.cfg.ReportLabels` (400 with "label not in configured list" otherwise); call `h.jira.UpdateIssueLabel(r.PathValue("issueKey"), req.Label, req.Action=="add")`; return 200 empty body on success; return 502 with error JSON on Jira failure (depends on T014)

- [x] T016 [US2] Add label button click handler to `web/app.js`: in `renderTable()` attach a click listener to each `.btn-label`; on click: read `data-key` and `data-label` from button dataset; read current active state from button classList; compute target action (`add` if currently inactive, `remove` if currently active); immediately disable button and add class `pending`; call `fetch('/api/issues/${key}/labels', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({action, label})})`; on success flip button between `active`/inactive classes and re-enable; on failure revert to original class and show error in `#error-banner`

**Checkpoint**: US2 functional — label buttons toggle state, Jira label is updated, button reverts on failure.

---

## Phase 5: User Story 3 — Dark / Light Theme Toggle (Priority: P3)

**Goal**: Theme toggle button at the far right of the toolbar switches between light and dark; choice persists across page refreshes.

**Independent Test**: Load dashboard in light theme; click theme toggle; verify all UI elements switch colour; refresh; verify dark theme is restored.

### Implementation for User Story 3

- [x] T017 [P] [US3] Add dark theme CSS to `web/style.css`: append `[data-theme="dark"]` rule block that overrides all --color-* custom properties with dark values per plan.md (--color-bg: #1e1e2e, --color-surface: #313244, --color-text: #cdd6f4, --color-text-muted: #a6adc8, --color-border: #45475a, --color-btn-team-active: #89b4fa, --color-btn-label-active: #a6e3a1, --color-btn-label-inactive: #45475a); update `#theme-toggle` button to show ☀️ icon when `[data-theme="dark"]` is set (use CSS content or JS)

- [x] T018 [P] [US3] Add theme toggle logic to `web/app.js`: on `DOMContentLoaded` (alongside `fetchTeams()`), read `localStorage.getItem('sprint-dashboard-theme')` and call `applyTheme(theme || 'light')`; `applyTheme(theme)` sets `document.documentElement.setAttribute('data-theme', theme)` and updates `#theme-toggle` button text (🌙 for light, ☀️ for dark); add click listener on `#theme-toggle` that toggles between `'light'` and `'dark'`, calls `applyTheme()`, and stores new value via `localStorage.setItem('sprint-dashboard-theme', newTheme)`; the inline `<script>` in `index.html` head (T011) already handles the pre-paint theme read to prevent flash — JS here handles the runtime toggle

**Checkpoint**: All three user stories are independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [x] T019 [P] Update `README.md`: add `cmd/web-sprint-labels-report/` to the Project Structure section; add a "Web Sprint Labels Dashboard" subsection under Running with build/run commands, available flags (`-port`, `-debug`), and a link to `http://localhost:8080`; update the `make help` description in README to mention `make run-web-dashboard [PORT=8080]`

- [x] T020 Run `make build` and confirm zero compilation errors across all four binaries (get-month-issues, get-sprint-issues, get-sprint-label-report, web-sprint-labels-report)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on T001; all tasks T002–T007 must complete before user story phases
- **US1 (Phase 3)**: Depends on all of Phase 2
- **US2 (Phase 4)**: Depends on Phase 2; T014 can begin once T002 is done (different file); T015 depends on T014; T016 depends on T015 (same js file, sequential)
- **US3 (Phase 5)**: Depends on T011 (index.html) and T012 (style.css) from US1; T017 and T018 can run in parallel with each other (different files)
- **Polish (Phase 6)**: Depends on completion of US1, US2, US3

### Within Phase 2

```
T001 (mkdir)
  └──► T002 (GetSprintsForBoard in jira.go)
  └──► T003 [P] (api.go types)       ← parallel with T002
  └──► T004 [P] (web/icons/*.svg)    ← parallel with T002, T003
  T003+T004 done ──► T005 (handler.go skeleton)
  T005 done ──► T006 (main.go)
  T001 done ──► T007 (Makefile)      ← parallel with T002–T006
```

### Within US1

```
T005 done ──► T008 (getTeams handler)
T008 done ──► T009 (getSprints handler, same file)   [T002 must be done]
T009 done ──► T010 (getSprintIssues handler, same file)
T011 [P]  ──► can start any time after Phase 2
T012 [P]  ──► can start any time after Phase 2
T011 done ──► T013 (app.js, depends on HTML structure)
```

### Parallel Opportunities

```
# Phase 2 — once T001 done, run T002+T003+T004+T007 in parallel:
T002 (jira.go), T003 (api.go), T004 (svgs), T007 (Makefile)
# then T005 (handler.go), then T006 (main.go)

# Phase 3 US1 — T011, T012 can run in parallel with T008/T009/T010:
T011 (index.html), T012 (style.css)   ← parallel with handler work

# Phase 5 US3 — fully parallel:
T017 (dark CSS), T018 (toggle JS)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 (T001) and Phase 2 (T002–T007)
2. Complete Phase 3 / US1 (T008–T013)
3. **STOP and VALIDATE**: `make run-web-dashboard`, open browser, verify full US1 flow
4. Ship if acceptable

### Incremental Delivery

1. T001–T007 → foundation compiles
2. T008–T013 → US1: read-only dashboard complete (MVP)
3. T014–T016 → US2: label toggles write to Jira
4. T017–T018 → US3: dark/light theme
5. T019–T020 → polish + final build check

---

## Notes

- T002 and T003 are in the same file (`jira.go`); implement sequentially or in one edit
- T005 (handler.go) must import `internal/dashboard` types from `api.go` (T003) — T003 must compile first
- T008, T009, T010 are all in `handler.go` — implement sequentially; each replaces a 501 stub
- T016 is in `app.js` (same as T013) — implement after T013
- T017 appends to `style.css` (same as T012) — implement after T012
- T018 adds to `app.js` (same as T013 + T016) — implement after T016
- The `//go:embed web` directive in handler.go requires `web/` to be non-empty at compile time — T004 (icons) ensures this
