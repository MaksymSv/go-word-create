# Research: Web Sprint Labels Dashboard

**Feature**: 005-web-sprint-labels-dashboard
**Date**: 2026-06-05

---

## Decision 1: Frontend Delivery — Embedded SPA vs Separate Build

**Decision**: Vanilla HTML/CSS/JS embedded in the Go binary via `//go:embed` (no build step).

**Rationale**: The spec assumption states "no separate build step or CDN is required for local use." The Go standard library's `embed` package (Go 1.16+, covered by project's Go 1.24 requirement) can serve static assets directly from within the binary. Vanilla JS is sufficient for the required interactivity (team/sprint selection, table rendering, label toggle, theme switch), eliminating any npm/node dependency.

**Alternatives considered**:
- React/Vue SPA: Requires a build step, npm, and bundler — violates the "no separate build step" assumption.
- Server-side rendering (Go templates): Simpler but requires full-page reload on sprint/team switch, which conflicts with the SC-001/SC-002 response-time requirements and the label-toggle UX.

---

## Decision 2: Sprint Retrieval — How to Get the Last 5 Sprints

**Decision**: Use `GetAllSprintsWithOptions(boardID int, &GetAllSprintsOptions{State: "active,closed"})` with pagination (incrementing `StartAt` until `IsLast == true`), then sort the collected slice by Sprint `ID` descending and take the first 5.

**Rationale**: The go-jira v1.17.0 library's `GetAllSprints` fetches only the first page (default ~50 items) without filtering. For boards with many sprints the simple method may miss the most recent ones. The options variant allows filtering to `active,closed` (excluding `future` sprints which haven't started yet) and supports pagination via `SearchOptions.StartAt`. Sprint IDs increase monotonically, so sorting by ID descending is equivalent to sorting newest-first and is cheaper than parsing `StartDate` (which can be nil for sprints in some states).

**Alternatives considered**:
- Sort by `StartDate`: More semantically correct but `StartDate` is a pointer and can be nil; ID sort is more robust.
- Use `GetAllSprints` simple method: Only returns first page — unusable for boards with >50 sprints.

---

## Decision 3: Label Update — go-jira API Method

**Decision**: Use `client.Issue.UpdateIssue(issueKey, map[string]interface{}{"update": map[string]interface{}{"labels": []map[string]string{{"add": label}}}})` for adding a label, and `{"remove": label}` for removing. This is the partial-update (PATCH-like) approach using Jira's `update` operation.

**Rationale**: The `UpdateIssue` with `"update"` operations map sends only the delta (add/remove a single label) without fetching or replacing the entire issue body. This avoids race conditions where two concurrent writes could clobber each other's label changes. Confirmed by the official go-jira `examples/addlabel/main.go`.

**Alternatives considered**:
- `Issue.Update(*Issue)` with full `Fields.Labels` list: Requires fetching the current issue first, then replacing the entire labels array. Race-prone and more network calls.

---

## Decision 4: Issue Type Icons — SVG Strategy

**Decision**: Embed hand-crafted SVG icons for the 6 primary Jira issue types (Bug, Story, Task, Sub-task, Epic, and a generic fallback) as files in `web/icons/`. These are served as static assets; the JavaScript maps the issue type name to the icon filename.

**Rationale**: Jira's official issue type icons are not publicly distributable; using unofficial SVGs that match Jira's visual language (coloured circles/squares with simple symbols) is acceptable for an internal tool. Encoding icons as inline SVG strings in JS would bloat the JavaScript file and make maintenance harder.

**Icon set** (filename → type):
| File | Issue Type | Colour |
|------|-----------|--------|
| `bug.svg` | Bug | Red `#E5493A` |
| `story.svg` | Story | Green `#63BA3C` |
| `task.svg` | Task | Blue `#4BADE8` |
| `subtask.svg` | Sub-task | Blue `#4BADE8` |
| `epic.svg` | Epic | Purple `#904EE2` |
| `issue.svg` | fallback | Grey `#8993A4` |

---

## Decision 5: HTTP Server Routing — stdlib vs Router Library

**Decision**: Use Go's standard `net/http` with manual path prefix matching for the small number of endpoints (5 API routes + 1 static catch-all). No external router dependency.

**Rationale**: The project constitution forbids introducing unnecessary dependencies. The five API routes (`/api/teams`, `/api/teams/{component}/sprints`, `/api/sprints/{sprintID}/issues`, `POST /api/issues/{key}/labels`, `DELETE /api/issues/{key}/labels/{label}`) can be handled with `strings.HasPrefix` routing and `strings.Split` path parsing. Adding `gorilla/mux` or `chi` for five routes would be premature.

**Alternatives considered**:
- `gorilla/mux` or `chi`: Full-featured but unnecessary for 5 routes; adds a dependency.
- Go 1.22 enhanced `net/http` pattern matching: Available (project uses Go 1.24) and eliminates manual parsing — use this instead of `strings.HasPrefix`.

**Revised decision**: Use Go 1.22+ `net/http` pattern matching (`http.NewServeMux()` with method+path patterns like `GET /api/teams/{component}/sprints`). No external dependency needed, and Go 1.24 supports this natively.

---

## Decision 6: Theme Persistence

**Decision**: Store the selected theme in `localStorage` under the key `sprint-dashboard-theme`. On page load, read the value and apply the theme class (`data-theme="dark"`) to the `<html>` element before any rendering to prevent flash of wrong theme.

**Rationale**: `localStorage` is universally available in target browsers, survives page refreshes and tab closes, and requires no server-side state. Setting the theme attribute on `<html>` via an inline `<script>` tag in `<head>` (before CSS renders) prevents a flash of the wrong theme.

---

## Decision 7: New Package — `internal/dashboard` vs Extending `internal/server`

**Decision**: Create a new package `internal/dashboard` to house the dashboard HTTP handlers and API types. Leave `internal/server` (the legacy Word-doc endpoint) unchanged.

**Rationale**: The existing `internal/server/handler.go` serves Word documents — a completely different concern. Mixing dashboard logic there would violate the constitution's Package Separation principle and make both packages harder to maintain. A dedicated `internal/dashboard` package keeps responsibilities clear.

---

## Decision 8: Sprint ID as Route Parameter

**Decision**: Use the integer Sprint ID (not name) as the route parameter for `/api/sprints/{sprintID}/issues`.

**Rationale**: Sprint names can contain spaces and special characters (e.g., "Sprint 16 (Hot-fix)"). Sprint IDs are stable integers that serve as natural primary keys. The frontend receives sprint IDs from the `/api/teams/{component}/sprints` response and can use them directly.
