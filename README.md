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
OUTPUT_FILE=output.docx
```

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

# Run commands
make run-month MONTH=2025.10              # Generate Word document
make run-month MONTH=2025.10 LOGONLY=1    # Print to console only (debug mode)
make run-sprint-label-report SPRINT="Sprint 16"
make run-sprint-label-report SPRINT="Sprint 16" FORMAT=full LOGONLY=1
make run-web-dashboard                    # Start dashboard on port 8080
make run-web-dashboard PORT=9090          # Start dashboard on custom port

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

# Build web dashboard
go build -o bin/web-sprint-labels-report ./cmd/web-sprint-labels-report
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

### Get Sprint Label Report

Generate a label/AI-usage report for a sprint, covering all teams configured in `TEAMS`:
```bash
./bin/get-sprint-label-report -sprint="Sprint 16" -output="label-report.docx"
```

#### Flags:
- `-sprint="Sprint Name"` (required): Sprint name
- `-output="file.docx"` (optional): Output file name
- `-format="short|full"` (optional, default `short`): Report format
- `-debug`: Print report to console instead of generating Word document

#### How it works:
Iterates over every team in `TEAMS`, fetches sprint issues from each board, and produces a combined document with one labeled section per team. If one board fails, that team is skipped with an error message and the remaining teams are still processed.

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
- Sprint table shows: issue type icon, issue key (Jira link), summary, epic, story points, status, and label buttons
- Clicking a label button adds or removes that label on the Jira issue instantly
- Dark/light theme toggle at the far right of the toolbar; theme is remembered across sessions

## Project Structure

```
go-word-create/
├── cmd/
│   ├── get-sprint-issues-from-jira/   # Sprint issues fetcher
│   ├── get-month-issues-from-jira/    # Month issues fetcher
│   ├── get-sprint-label-report/       # Sprint label report
│   └── web-sprint-labels-report/      # Web dashboard server
├── internal/
│   ├── config/              # Configuration loading from .env
│   ├── dashboard/           # Web dashboard HTTP handlers and embedded SPA
│   ├── jiraservice/         # Jira API client and issue fetching
│   ├── labelreport/         # Label aggregation logic
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
