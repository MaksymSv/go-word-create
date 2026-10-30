# AGENTS.md

Guidance for AI coding agents working in this repository. Human-oriented docs live in `README.md`; the binding project rules live in `.specify/memory/constitution.md` (v1.2.0). If they conflict, the constitution wins.

## What this project is

`go-word-create` (Go module `go-word-create`, Go 1.24+) reads Jira Cloud and reports on sprints and AI-usage labels. It has three surfaces over shared `internal/` logic:

| Surface | Binary (`bin/`) | Entry point | Purpose |
|---|---|---|---|
| CLI | `get-month-issues` | `cmd/get-month-issues-from-jira` | Issues "In Progress" during a month, per team, to Word |
| CLI | `get-sprint-issues` | `cmd/get-sprint-issues-from-jira` | Issues of one or more sprints to Word (`-format docx`) or CSV (`-format csv`, one file per sprint) |
| CLI | `get-sprint-label-report` | `cmd/get-sprint-label-report` | Label/AI-usage report across all teams (`-format short\|full`) |
| HTTP | `web-sprint-labels-report` | `cmd/web-sprint-labels-report` | Browser dashboard that also writes labels back to Jira |
| MCP (stdio) | `mcp-jira-reports` | `cmd/mcp-jira-reports` | Typed JSON tools for LLM clients; registered in `.mcp.json` |

## Commands

`make` is the canonical interface (`make help` lists everything). New binaries need a `make` target and a `make help` entry.

```bash
make build                      # all binaries into bin/
make test                       # go test -v ./...
make fmt                        # go fmt ./...
make lint                       # golangci-lint (auto-installs if missing)
make run-sprint SPRINT="ST26.20,PC26.20" [FORMAT=csv] [CSV_DIR=dir] [LOGONLY=1]
make run-month MONTH=2025.10 [LOGONLY=1]
make run-sprint-label-report SPRINT="Sprint 16" [FORMAT=full]
make run-web-dashboard [PORT=8080]
```

Before finishing any change: `make fmt`, `make lint`, `make test`, `make build` must all pass (constitution gate). `go vet ./...` and `go test ./...` are clean at the time of writing.

## Layout

```
cmd/                    Wiring only: flags, config.Load(), call internal/ packages
internal/
  config/               .env loading, TEAMS / REPORT_LABELS / ISSUE_TYPES parsing
  jiraservice/          Jira client (go-jira): boards, sprints, issues, changelog, label update
  sprintview/           Per-sprint issue view + active report labels (shared by dashboard and MCP)
  labelreport/          Label aggregation per Jira component
  issuecsv/             CSV writer and <sprint>_<YYYY-MM-DD>.csv file naming
  word/                 Word table/document generation (gooxml)
  dashboard/            HTTP handlers; embedded SPA in internal/dashboard/web/ (vanilla JS, no build step)
  mcpserver/            MCP tool definitions, handlers, report shaping
  server/               Leftover sample .docx handler; not wired into any cmd
specs/NNN-name/         Spec Kit feature specs and plans (001-009; 008 and 009 are retroactive)
.specify/               Spec Kit templates, scripts, constitution
.claude/skills/         Project skills (speckit-*, jira-sprint-reports)
```

Dependency direction: `cmd` -> `mcpserver`/`dashboard` -> `sprintview`/`labelreport`/`issuecsv` -> `jiraservice`. `word` must not import Jira or HTTP code; `jiraservice` must not import `word`. No circular imports.

## Configuration (.env)

Loaded by `config.Load()` via godotenv. Required: `JIRA_URL`, `JIRA_USERNAME`, `JIRA_API_TOKEN`, `JIRA_PROJECT_KEY`, `TEAMS`. The process must exit with a clear error if any is missing.

Optional, with defaults: `JIRA_EPIC_FIELD` (`customfield_14500`), `JIRA_SP_FIELD` (`customfield_10004`), `JIRA_COMPONENT_FIELD` (`components`), `DEFAULT_OUTPUT_FILE` (`sprint-issues.docx`), `REPORT_LABELS` (`ai-assisted,ai-assisted-ba,ai-assisted-dev,ai-assisted-qa`), `ISSUE_TYPES` (`Bug,Story,Task`).

`TEAMS` format: `COMPONENT|"Board Display Name",...`. The component filters issues; the board name resolves sprints. Without `|`, the component name is used for both.

Known doc drift: `.env.example` omits `REPORT_LABELS` and `ISSUE_TYPES`, and the README shows `OUTPUT_FILE` where the code reads `DEFAULT_OUTPUT_FILE`. The constitution requires `.env.example` to list every variable, so fix it when touching config.

## Rules that are easy to break

- **Never read, print, or commit `.env`.** It holds the Jira API token. Use the `list_teams` MCP tool to inspect configuration. `.env`, `bin/`, `*.docx` and `*.csv` are gitignored; keep it that way.
- **stdout belongs to the protocol in the MCP server.** All logging there goes to stderr. The MCP server never writes files; CSV is returned as text.
- **Jira writes are guarded.** `set_issue_label` must be called with `confirm=false` first, shown to the user, and only repeated with `confirm=true` after explicit approval. The dashboard `POST /api/issues/{key}/labels` only accepts labels from `REPORT_LABELS`.
- **Sprint names are exact and case-sensitive.** Multiple sprints are comma-separated and parsed by `jiraservice.ParseSprintNames`.
- **Debug mode:** binaries that write a file take `-debug` (or `LOGONLY=1` through make) to print human-readable output instead; for the dashboard and MCP server `-debug` means verbose stderr logging. `get-sprint-label-report` already prints to stdout and has no `-debug`.
- **Errors need context** (team, sprint or month, issue key). Bare "failed" messages are forbidden. Logs go to stderr; documents go to the output file.
- **Word formatting is a standard** applied identically on every document path: Aptos Narrow 8 pt; single auto-color borders at 0.5 pt (the README says 1 pt, the constitution says 0.5); header row `#0070C0` with white non-bold text; cell margins top/bottom 0 and left/right 1.5 mm; Heading 1 in Trebuchet MS 16 pt small caps; page break before every section heading after the first. Change all code paths together.
- **Business logic goes in `internal/`**, not in `cmd/` and not in HTTP handlers.
- **Dependencies:** use `go-jira` for Jira, `gooxml` for OOXML, `modelcontextprotocol/go-sdk` for MCP. Do not add a second library for the same job.
- **Dashboard frontend is vanilla JS** embedded with `//go:embed web` and served at `/static/`; theme and the last issue list are kept in `localStorage`. There is no bundler or JS test setup, so verify UI changes in a browser (`make run-web-dashboard`).

## Testing

Tests exist for `issuecsv`, `jiraservice`, `mcpserver` (using a fake implementing the `mcpserver.Jira` interface) and `sprintview` (using `sprintview.IssueSource`). `config`, `dashboard`, `labelreport`, `word` and all `cmd/` packages have none. Prefer small interfaces plus fakes over a live Jira; tests must not need network or credentials.

## Workflow

- Working branch is `dev`; `master` is the main branch. Feature work follows Spec Kit: `/speckit-specify` -> `plan` -> `tasks` -> `implement`, with artifacts under `specs/NNN-short-name/` and a Constitution Check in each `plan.md`.
- Specs 001-007 were written before implementation; 008 (MCP server) and 009 (multi-sprint and CSV output) were written afterwards from the code and say so in their `Status` line. Where a plan diverges from the code, look for its "As-Built Notes" section (see 007). The frontend lives in `internal/dashboard/web/`, not a repo-root `web/`.
- Work done outside Spec Kit should get a short retroactive spec (constitution v1.2.0, Development Workflow).
- Commits: imperative present tense, subject 72 characters or fewer. One logical change per PR (document formatting, Jira client, and server changes separately where possible). Confirm `.env` is not staged before committing.
- Jira reporting questions (label percentages, issues per status, month activity) are best answered through the `jira-reports` MCP tools and the `jira-sprint-reports` skill rather than by running the CLIs.
