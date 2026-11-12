# Go Word Create

A Go project that integrates with Jira to generate Word reports and serve a web dashboard for sprint label management.

## Project Overview

This project provides command-line tools and a web dashboard for Jira issue reporting:

- **Get Month Issues**: Fetch all issues that were "In Progress" during a specific month, grouped by team/component, and export to Word
- **Get Sprint Issues**: Fetch all issues from a specific sprint and export to Word
- **Get Sprint Label Report**: Fetch sprint issues and generate a label/AI-usage report across all configured teams
- **Web Sprint Labels Dashboard**: Browser dashboard to browse sprint issues per team and toggle Jira labels directly from the UI

## Features

- 📊 **Jira Integration**: Connect to Jira Cloud to fetch issues, sprints, and epic information
- 📄 **Word Document Generation**: Create formatted Word documents with tables containing issue details
- 🌐 **Web Dashboard**: Browser-based sprint issues viewer with live label assignment
- 🔍 **Issue Filtering**: Filter by issue type (Bug, Feature, Task, etc.)
- 👥 **Multi-Team Support**: Process multiple teams/components via the `TEAMS` environment variable
- 🏷️ **Label Management**: View and toggle configured labels on Jira issues from the dashboard
- 📅 **Month-based Filtering**: Find all issues that transitioned to "In Progress" during a specific month
- 🎨 **Formatted Tables**: Custom fonts (Aptos Narrow, 8pt), proper margins, and styled headers

## Prerequisites

- **Go 1.24.0** or higher
- **Jira** account with API token
- **make** (optional, for using Makefile)

## Environment Setup

1. Clone the repository:
```bash
git clone <repository-url>
cd go-word-create
```

2. Install dependencies:
```bash
go mod download
```

3. Create a `.env` file based on `.env.example`:
```bash
cp .env.example .env
```

4. Configure your `.env` file with Jira credentials:
```env
JIRA_URL=https://your-jira-instance.atlassian.net
JIRA_USERNAME=your-email@example.com
JIRA_API_TOKEN=your-api-token
JIRA_PROJECT_KEY=PROJ
JIRA_EPIC_FIELD=customfield_10014
JIRA_SP_FIELD=customfield_10015
JIRA_COMPONENT_FIELD=components
TEAMS=PROCESSING|"PROCESSING Team",STABLETEK|"STABLETEK Team"
DEFAULT_OUTPUT_FILE=output.docx
```

The delivery dashboard and the RAID scan have their own optional `DELIVERY_*` and `RAID_*` settings; see `.env.example`.

### Getting Jira API Token

1. Go to https://id.atlassian.com/manage-profile/security/api-tokens
2. Click "Create API token"
3. Copy the token and paste it in your `.env` file

### Finding Custom Field IDs

Run the following to find your custom field IDs:
```bash
curl -u your-email@example.com:your-api-token \
  https://your-jira-instance.atlassian.net/rest/api/3/fields
```

Look for `customfield_XXXXX` entries for Epic Link and Story Points fields.

## Building

### Using Make

```bash
# Build all binaries
make build

# Build specific binary
make build-month
make build-sprint
make build-sprint-label-report
make build-web-dashboard
make build-delivery-report
make build-raid-report
make build-mcp

# Run commands
make run-month MONTH=2025.10              # Generate Word document
make run-month MONTH=2025.10 LOGONLY=1    # Print to console only (debug mode)
make run-sprint-label-report SPRINT="Sprint 16"
make run-sprint-label-report SPRINT="Sprint 16" FORMAT=full
make run-sprint-label-report SPRINT="Sprint 16,Sprint17"
make run-chart                            # Fill empty sprint columns of proj-docs/AI Assisted Work Items Chart.xlsx
make run-chart DRYRUN=1                   # Show what would be filled, write nothing
make run-chart FILE=book.xlsx OUTPUT=copy.xlsx
make run-web-dashboard                    # Start dashboard on port 8080
make run-web-dashboard PORT=9090          # Start dashboard on custom port
make run-delivery-report TEAM=STABLETEK SPRINTS=6   # Print delivery metrics as JSON
make run-raid-report TEAM=STABLETEK                  # Print the RAID scan as JSON

# Show all available targets
make help
```

### Using Go directly

```bash
# Build month issues fetcher
go build -o bin/get-month-issues ./cmd/get-month-issues-from-jira

# Build sprint issues fetcher
go build -o bin/get-sprint-issues ./cmd/get-sprint-issues-from-jira

# Build sprint label report
go build -o bin/get-sprint-label-report ./cmd/get-sprint-label-report

# Build Excel chart filler
go build -o bin/fill-sprint-chart-xlsx ./cmd/fill-sprint-chart-xlsx

# Build web dashboard
go build -o bin/web-sprint-labels-report ./cmd/web-sprint-labels-report

# Build delivery metrics report
go build -o bin/get-delivery-report ./cmd/get-delivery-report

# Build RAID scan report
go build -o bin/get-raid-report ./cmd/get-raid-report

# Build MCP server
go build -o bin/mcp-jira-reports ./cmd/mcp-jira-reports
```

## Running

### Get Month Issues

Fetch all issues that were "In Progress" during October 2025, grouped by teams configured in `TEAMS`:
```bash
make run-month MONTH=2025.10
```

To see issues in console without generating a Word document:
```bash
make run-month MONTH=2025.10 LOGONLY=1
```

Or run directly:
```bash
./bin/get-month-issues -month="2025.10" -output="october-report.docx"
```

#### Flags:
- `-month="YYYY.MM"` (required): Month to fetch issues from (e.g., "2025.10")
- `-output="file.docx"` (optional): Output file name (default: from .env)
- `-debug`: Print issues to console instead of generating Word document

#### How it works:
The month command processes each team configured in the `TEAMS` environment variable:
- For each team, it fetches issues that were "In Progress" during the specified month
- Issues are filtered by the team's component name in Jira
- Results are grouped by team in the output document
- Each team gets separate sections for "Closed Issues" and "Open Issues"

### Get Sprint Issues

Fetch all issues from a specific sprint:
```bash
./bin/get-sprint-issues -sprint="Sprint 16" -output="sprint-16.docx"
```

Uses the board name from the first team entry in `TEAMS`.

#### Flags:
- `-sprint="Sprint Name"` (required): Sprint name to fetch issues from
- `-output="file.docx"` (optional): Output file name (default: from .env)
- `-debug`: Print issues to console instead of generating Word document

### Fill the AI Assisted Work Items Chart workbook

Fills `proj-docs/AI Assisted Work Items Chart.xlsx` with the numbers of the sprint label report:
```bash
./bin/fill-sprint-chart-xlsx            # updates the workbook in place
./bin/fill-sprint-chart-xlsx -dry-run   # show the values without writing
```
- Columns: row 1 holds the sprint (e.g. `S26.20`), row 2 the team prefix (`ST` = STABLETEK component, `PC` = PROCESSING). The Jira sprint name is prefix + number, e.g. `ST26.20`.
- Rows: found through the `jira label` column (B) and the row title (`WI count` vs `SP`); rows with an empty label whose title starts with `total` get the share of issues/story points that have any tracked label.
- Chart: series ranges are widened to the last header column (so columns added beyond `AD` are plotted) and the cached chart values are refreshed.
- Only empty cells under existing headers are filled, values are whole percentages. Sprints missing in Jira are skipped and reported.
- Flags: `-file` (default `proj-docs/AI Assisted Work Items Chart.xlsx`), `-output` (write a copy instead of updating in place), `-dry-run`.

### Get Sprint Label Report

Generate a label/AI-usage report for a sprint, covering all teams configured in `TEAMS`:
```bash
./bin/get-sprint-label-report -sprint="Sprint 16"
```

#### Flags:
- `-sprint="Sprint Name"` (required): Sprint name, comma separatred list of sprints is accepted
- `-format="short|full"` (optional, default `short`): Report format

#### How it works:
Iterates over every team in `TEAMS`, fetches sprint issues from each board, and produces a combined report with one labeled section per team. If one board fails, that team is skipped with an error message and the remaining teams are still processed.

### Web Sprint Labels Dashboard

Start a browser dashboard showing sprint issues per team, with label-toggle buttons that write back to Jira:

```bash
make run-web-dashboard [PORT=8080]
```

Or run directly:
```bash
./bin/web-sprint-labels-report -port 8080
```

Then open `http://localhost:8080` in a browser.

#### Flags:
- `-port=8080` (optional): HTTP listen port (default 8080)
- `-debug`: Enable verbose logging

#### Features:
- Team selector buttons load the 5 most recent sprints for each configured board
- Sprint table shows: issue type icon, issue key (Jira link), summary, epic, implementer, story points, status, and label buttons
- Implementer is the assignee before the issue first moved to "In Progress"; if there was none, the assignee before it first moved to "In Review."; otherwise the current assignee
- Clicking a label button adds or removes that label on the Jira issue instantly
- Dark/light theme toggle at the far right of the toolbar; theme is remembered across sessions

### Delivery Dashboard

The same server also serves a project delivery dashboard at `http://localhost:8080/delivery` (linked from the toolbar of both pages). It has one tab per team in `TEAMS`, and each tab shows seven charts for the last N closed sprints (default 6, set with `DELIVERY_SPRINT_COUNT` or the "Sprints" pills):

1. **Velocity**: committed vs. completed story points per sprint, plus the average of completed over the last 3 sprints
2. **Commitment rate**: completed ÷ committed, in %
3. **Backlog health**: story points of open issues outside any sprint when the sprint started (hover for issue count, unestimated issues and runway in sprints)
4. **Lead and cycle time**: average days for issues resolved in the sprint (lead = created → resolved, cycle = first "In Progress" → resolved)
5. **Open bugs**: bugs not yet closed at the end of each of the last 30 days (`DELIVERY_BUG_DAYS`)
6. **Bug dynamics**: bugs logged vs. fixed per sprint
7. **Open bugs by priority**: bugs not yet closed, per priority, at the end of each sprint

Every card has a **Table** button that shows the numbers behind the chart. **Refresh** re-reads Jira (closed-sprint results stay cached).

Data comes from Jira only. Velocity uses the board's velocity chart (`/rest/greenhopper/...`, an undocumented Jira Software endpoint that returns just the last 7 sprints); everything else is computed from issue fields and changelogs. To keep page loads fast and to keep history beyond Jira's 7-sprint window, closed-sprint results (backlog, lead/cycle time, velocity) are stored in `DELIVERY_CACHE_DIR` (default `.cache/delivery`, one JSON file per team) and reused; bug series are cheap and read live. The first load of a team scans issue history and can take up to a minute; the server warms both teams at startup. Delete the cache directory to rebuild from Jira, but note that velocity for sprints older than Jira's 7-sprint window cannot be rebuilt.

The same numbers are available as JSON for scripts:
```bash
make run-delivery-report TEAM=PROCESSING SPRINTS=6   # all teams when TEAM is omitted
```

Metric definitions worth knowing:
- Team scope is the board's own filter: `project = JIRA_PROJECT_KEY AND component = <team component>`. A bug tagged with several components counts for each of those teams (about a fifth of STABLETEK's bugs also carry Compliance, Issue, Faucet or Redeem).
- "Open" means the status category is not Done, not "has no resolution": some teams close issues without setting a resolution. For issues closed that way, the close date is the time they entered their current status, read from the changelog once and remembered in the cache.
- Sprint windows do not overlap: a sprint's events start at the later of its start and the previous sprint's close, and end at its close.
- Fixed bugs, lead time and cycle time count every resolution (Done, Duplicate, Won't Fix, ...) unless `DELIVERY_COMPLETED_RESOLUTIONS` narrows it. Checked against Delivery Central for PROCESSING: fixed bugs match exactly, logged bugs are within 1 to 2, lead time is close in most months; its cycle time is about half of ours because it appears to measure to an earlier status (details in `specs/012-delivery-dashboard/plan.md`).
- Priorities and story points are read as they are now, not as they were at the time.

### Risk / RAID Scan

`http://localhost:8080/raid` (linked from the toolbar of every page) scans a team's open Jira work for RAID signals, one tab per team in `TEAMS`. It reads the whole open backlog (`project` + `component`, status category not Done) with its changelog, so blocked work, dependencies and critical bugs are found outside the sprint too.

- **View switch** (above the tiles): "All open work" or "Current sprint <name>". The sprint view narrows the tiles, the counts in the filter chips and the watchlist to issues in the running sprint; the sprint pulse, scope charts and the work-in-progress chart always cover what they say. The choice is remembered in the browser; the sprint option is disabled when no sprint is running.
- **Tiles**: Risks, Assumptions, Issues and Dependencies counts, colored red when an issue has a high-severity signal, amber for other signals, green when clear (the status is also written as text). Click a tile to filter the watchlist. The fifth tile is the scope drift of the running sprint.
- **Sprint pulse**: time elapsed vs. share of scope completed in the running sprint, with the committed/added/removed/re-estimated story points and how many items have not started.
- **Watchlist**: issues ranked by their most severe signal, with a chip per signal (hover for the reason) and filters per signal.
- **Charts**: work in progress by age, scope added/removed per sprint, scope drift per sprint against `RAID_SCOPE_DRIFT_PCT`, and a daily trend of flagged, blocked and aging issues. The two scope charts plot the running sprint plus the last `DELIVERY_SPRINT_COUNT` closed ones; the page has no sprint selector (the API and `get-raid-report` accept `sprints`/`-sprints`).

| Category | Signals |
|---|---|
| Risks | aging (in progress `RAID_AGING_DAYS`+, high at twice that), stale (no update for `RAID_STALE_DAYS`), late start (still to do after `RAID_LATE_START_PCT` of the sprint), rolled over (`RAID_ROLLOVER_SPRINTS`+ sprints), overdue |
| Assumptions | sprint work without story points (bugs excluded), in-progress work without an assignee |
| Issues | blocked (status in `RAID_BLOCKED_STATUSES`, label in `RAID_BLOCKED_LABELS`, or flagged as an impediment), critical bug (`RAID_CRITICAL_PRIORITIES`) |
| Dependencies | linked "is blocked by" / "depends on" issue that is not done |

Scope drift comes from Jira's own sprint report (`/rest/greenhopper/...`, undocumented like the velocity chart): committed is the estimate of issues in the sprint when it started and matches the velocity chart, added and removed are changes after the start, and re-estimated is the estimate change of issues that stayed. Closed-sprint scope and the daily trend are stored in `DELIVERY_CACHE_DIR` (`raid-<team>.json`); the watchlist is read live and reused for `DELIVERY_CACHE_TTL`. Jira has no history of what was blocked on a given day, so the trend starts the day the scan first ran.

```bash
make run-raid-report TEAM=PROCESSING SPRINTS=6   # all teams when TEAM is omitted
```

### MCP Server (for LLM clients)

A [Model Context Protocol](https://modelcontextprotocol.io) server over stdio that exposes the same Jira data as typed JSON tools, so Claude Code, Claude Desktop, Codemie and other MCP clients can answer sprint/label questions directly.

```bash
make build-mcp
./bin/mcp-jira-reports -workdir /path/to/go-word-create
```

#### Flags:
- `-workdir=.` (optional): directory containing `.env`; the process changes into it before loading configuration. Set it when the client starts the server from another directory.
- `-debug`: Enable verbose logging (stderr only; stdout carries the protocol)

#### Tools:
| Tool | Purpose |
|---|---|
| `list_teams` | Configured teams (component + board), tracked report labels, issue types |
| `list_sprints` | Latest sprints of a team's board (default 5), with IDs |
| `get_sprint_issues` | Issues of a sprint by ID, optional status filter, per-label counts and story points |
| `sprint_issues` | Issues of one or more sprints by name (same data as `get-sprint-issues`), optional team/status filters; `format=csv` also returns CSV text and a suggested `<sprint>_<date>.csv` name per sprint (the server never writes files) |
| `sprint_label_report` | Label report by sprint name(s), grouped by component (same data as `get-sprint-label-report -format full`) |
| `month_issues` | Issues "In Progress" during a month, Closed vs Open per team (same data as `make run-month LOGONLY=1`), optional `team`; `format=docx` also returns the Word report as base64 plus a suggested `<DEFAULT_OUTPUT_FILE> - YYYY-MM.docx` name (same document as `make run-month`; the server never writes files) |
| `fill_sprint_chart` | Fills the empty sprint columns of the chart workbook (same as `make run-chart`). Without `write=true` nothing is saved and the filled `.xlsx` is returned as base64; **`write=true` updates the file** in the server working directory |
| `set_issue_label` | **Write.** Add/remove a tracked label on an issue. Without `confirm=true` it only returns a preview |

#### Registration:
The repository ships a project-scoped `.mcp.json` (no secrets; credentials stay in `.env`). Build the binary first, then start Claude Code in the repo. To register it elsewhere:

```bash
claude mcp add jira-reports -- /abs/path/bin/mcp-jira-reports -workdir /abs/path/go-word-create
```

Claude Desktop / Codemie (`mcpServers` JSON):

```json
{
  "mcpServers": {
    "jira-reports": {
      "command": "/abs/path/bin/mcp-jira-reports",
      "args": ["-workdir", "/abs/path/go-word-create"]
    }
  }
}
```

A companion skill at `.claude/skills/jira-sprint-reports/` teaches Claude which tool to use and how to read the metrics.

## Project Structure

```
go-word-create/
├── cmd/
│   ├── get-sprint-issues-from-jira/   # Sprint issues fetcher
│   ├── get-month-issues-from-jira/    # Month issues fetcher
│   ├── get-sprint-label-report/       # Sprint label report
│   ├── fill-sprint-chart-xlsx/        # Fills the Excel chart workbook
│   ├── web-sprint-labels-report/      # Web dashboard server (labels and delivery pages)
│   ├── get-delivery-report/           # Delivery metrics as JSON
│   ├── get-raid-report/               # RAID scan as JSON
│   └── mcp-jira-reports/              # MCP server (stdio) for LLM clients
├── internal/
│   ├── chartfill/           # Excel chart workbook filling (CLI and MCP)
│   ├── config/              # Configuration loading from .env
│   ├── dashboard/           # Web dashboard HTTP handlers and embedded SPA
│   ├── deliverystats/       # Delivery metrics: velocity, backlog, lead/cycle time, bug trends, cache
│   ├── raidscan/            # RAID scan: blocked, aging and at-risk items, scope drift, daily trend
│   ├── jiraservice/         # Jira API client and issue fetching
│   ├── labelreport/         # Label aggregation logic
│   ├── mcpserver/           # MCP tool definitions and handlers
│   ├── sprintview/          # Sprint issue view shared by dashboard and MCP
│   └── word/                # Word document generation and table formatting
├── go.mod                   # Go module definition
├── .env.example             # Example environment variables
├── Makefile                 # Build automation
└── README.md                # This file
```

## Development

### Code Formatting

```bash
make fmt
```

### Linting

```bash
make lint
```

(Automatically installs golangci-lint if not present)

### Running Tests

```bash
make test
```

### Cleaning Build Artifacts

```bash
make clean
```

## Configuration Details

### Jira Custom Fields

The project uses these Jira fields, which can be configured via `.env`:

1. **Epic Link** (`JIRA_EPIC_FIELD`, default: `customfield_14500`): Links issues to epics
2. **Story Points** (`JIRA_SP_FIELD`, default: `customfield_10004`): Stores story point estimates
3. **Components** (`JIRA_COMPONENT_FIELD`, default: `components`): Jira components field name or custom field ID

These IDs/names may vary in your Jira instance. Use the API endpoint mentioned above to find the correct values.

### Teams Configuration

The `TEAMS` environment variable configures teams using the format `COMPONENT_NAME|"Board Display Name"`, comma-separated:

```env
TEAMS=PROCESSING|"PROCESSING Team",STABLETEK|"STABLETEK Team"
```

- **Component name** (before `|`): must match the Jira Component name exactly; used to filter issues in month reports
- **Board display name** (after `|`, optional quotes): used for sprint board lookups
- If no `|` separator is given, the component name is used as the board name

In code this is available as `cfg.Teams` (`[]config.TeamEntry`), where each entry has `ComponentName` and `BoardName` fields.

### Output File Format

Generated Word documents include sections for each team configured in `TEAMS`:

**For Month Issues Command:**
- Each team gets two sections:
  1. "Closed Issues During [Month] ([Team])" — issues that were closed
  2. "Issues were in work but not Closed during [Month] ([Team])" — issues still open

**Table Columns:**
- **Type**: Issue type (Bug, Feature, Task)
- **ID**: Jira issue key (e.g., PROJ-123)
- **Description**: Issue title/summary
- **Epic**: Epic name the issue belongs to
- **SP**: Story point estimate

### Table Formatting

Tables in generated documents use:
- **Font**: Aptos Narrow, 8pt
- **Borders**: Single auto-color borders at 1pt
- **Header**: Blue background (`#0070C0`) with white text
- **Cell margins**: Top and bottom only (~1mm / 57 dxa)

## Troubleshooting

### "Sprint not found" error
- Ensure the sprint name matches exactly (case-sensitive)
- Sprint must be associated with the board configured in `TEAMS` for that team

### "Failed to search epics" error
- Verify `JIRA_EPIC_FIELD` is correct for your Jira instance
- Check that your Jira user has permission to view custom fields

### "required environment variable TEAMS is not set" error
- Add `TEAMS=PROCESSING|"PROCESSING Team",STABLETEK|"STABLETEK Team"` (or your team entries) to your `.env` file
- Ensure component names match Jira Component names exactly

### No issues found for a team
- Verify the component name in `TEAMS` matches a Component name in Jira (check spelling and case)
- Ensure issues have the Component assigned in Jira
- Check that issues were actually "In Progress" during the specified month
- Use `-debug` flag to see detailed filtering information

### Document generation fails
- Check that the output directory exists and is writable
- Ensure there's enough disk space
- Verify the output filename doesn't conflict with an open file

### Dashboard shows no teams or "team not found"
- Ensure `TEAMS` is set correctly in `.env` with the `COMP|"Board Name"` format
- Component names in the URL are case-sensitive; they must match exactly what is in `TEAMS`

### Dashboard sprint list is empty
- Verify the board name in `TEAMS` matches the Jira board display name exactly
- The board must have at least one active or closed sprint; future-only boards show no buttons

### Delivery dashboard: a chart says "Could not compute this chart"
- The message names the team and the Jira error; the other charts are unaffected
- Velocity and commitment need the Jira Software velocity chart (`/rest/greenhopper/...`); if that endpoint is blocked, previously stored velocity is used when available

### Delivery dashboard: velocity is empty for older sprints
- Jira's velocity chart only returns the last 7 sprints. Older sprints are filled in once they have been seen, so run the dashboard regularly (or keep `DELIVERY_CACHE_DIR`) to build history

### Delivery dashboard: first load is slow
- The first load of a team scans issue changelogs (up to about a minute for large teams). Results for closed sprints are cached in `DELIVERY_CACHE_DIR`; the next closed sprint triggers one more scan

### Label button click has no effect
- Check the browser console and server logs for error details
- Confirm `REPORT_LABELS` in `.env` includes the label you are trying to assign
- Verify your Jira API token has edit permissions on the issue

## Dependencies

- `github.com/andygrunwald/go-jira` - Jira API client
- `github.com/carmel/gooxml` - Word document generation
- `github.com/joho/godotenv` - Environment variable loading

## License

MIT License - Copyright (c) 2025 Go Word Create Contributors

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

1. Create a feature branch (`git checkout -b feature/amazing-feature`)
2. Commit your changes (`git commit -m 'Add amazing feature'`)
3. Push to the branch (`git push origin feature/amazing-feature`)
4. Open a Pull Request

## Support

For issues and questions, please create an issue in the project repository.
