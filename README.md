# Go Word Create

A Go project that generates Word documents from Jira issues. Supports fetching issues by sprint or by month, with filtering by issue type and team/component.

## Project Overview

This project provides command-line tools to fetch Jira issues and generate Word documents:

- **Get Sprint Issues**: Fetch all issues from a specific sprint and export to Word
- **Get Month Issues**: Fetch all issues that were "In Progress" during a specific month, grouped by team/component, and export to Word

## Features

- 📊 **Jira Integration**: Connect to Jira Cloud to fetch issues, sprints, and epic information
- 📄 **Word Document Generation**: Create formatted Word documents with tables containing issue details
- 🔍 **Issue Filtering**: Filter by issue type (Bug, Feature, Task, etc.)
- 👥 **Multi-Team Support**: Process multiple teams/components configured in `TEAMS` environment variable
- 🏷️ **Component Filtering**: Filter issues by Jira Components (team names) for precise reporting
- 📅 **Month-based Filtering**: Find all issues that transitioned to "In Progress" during a specific month
- 🎨 **Formatted Tables**: Custom fonts (Aptos Narrow, size 8), proper margins, and styling

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
JIRA_BOARD_NAME=Your Board Name
JIRA_PROJECT_KEY=PROJ
JIRA_EPIC_FIELD=customfield_10014
JIRA_SP_FIELD=customfield_10015
JIRA_COMPONENT_FIELD=components
TEAMS=PROCESSING,STABLETEK
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

# Run commands
make run-month MONTH=2025.10              # Generate Word document
make run-month MONTH=2025.10 LOGONLY=1    # Print to console only (debug mode)
make run-sprint SPRINT="Sprint 16"        # Generate Word document for sprint
make run-sprint SPRINT="Sprint 16" LOGONLY=1   # Print to console only (debug mode)

# Show all available targets
make help
```

### Using Go directly

```bash
# Build month issues fetcher
go build -o bin/get-month-issues ./cmd/get-month-issues-from-jira

# Build sprint issues fetcher
go build -o bin/get-sprint-issues ./cmd/get-sprint-issues-from-jira
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
make run-sprint SPRINT="Sprint 16"
```

To see issues in console without generating a Word document:
```bash
make run-sprint SPRINT="Sprint 16" LOGONLY=1
```

Or run directly:
```bash
./bin/get-sprint-issues -sprint="Sprint 16" -output="sprint-16.docx"
```

#### Flags:
- `-sprint="Sprint Name"` (required): Sprint name to fetch issues from
- `-output="file.docx"` (optional): Output file name (default: from .env)
- `-debug`: Print issues to console instead of generating Word document

## Project Structure

```
go-word-create/
├── cmd/
│   ├── get-sprint-issues-from-jira/   # Sprint issues fetcher
│   └── get-month-issues-from-jira/    # Month issues fetcher
├── internal/
│   ├── config/              # Configuration loading from .env
│   ├── jiraservice/         # Jira API client and issue fetching
│   └── word/                # Word document generation, table formatting utilities
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

1. **Epic Link** (`JIRA_EPIC_FIELD`, default: `customfield_10014`): Links issues to epics
2. **Story Points** (`JIRA_SP_FIELD`, default: `customfield_10015`): Stores story point estimates
3. **Components** (`JIRA_COMPONENT_FIELD`, default: `components`): Jira components field name or custom field ID

These IDs/names may vary in your Jira instance. Use the API endpoint mentioned above to find the correct values.

### Teams configuration

The `TEAMS` environment variable configures which teams/components are processed by the month issues command:

- **TEAMS** (required): A comma-separated list of team/component names (e.g., `PROCESSING,STABLETEK`).  
  - Each team name should match a Jira Component name exactly (case-insensitive matching)
  - The month command will fetch and group issues separately for each team
  - Issues are filtered by matching the team name against the issue's Components field in Jira

In code this is available as a Go slice `cfg.Teams` (e.g., `[]string{"PROCESSING", "STABLETEK"}`).

**Note**: The team names in `TEAMS` must match the Component names in your Jira instance. Use the Jira UI or API to verify component names.

### Output File Format

Generated Word documents include sections for each team configured in `TEAMS`:

**For Month Issues Command:**
- Each team gets two sections:
  1. "Closed Issues During [Month] ([Team])" - Issues that were closed
  2. "Issues were in work but not Closed during [Month] ([Team])" - Issues still open

**Table Columns:**
- **Type**: Issue type (Bug, Feature, Task)
- **ID**: Jira issue key (e.g., PROJ-123)
- **Description**: Issue title/summary
- **Epic**: Epic name the issue belongs to
- **SP**: Story point estimate

**Additional Issue Information:**
- **Status**: Current issue status (available in debug mode)
- **URL**: Direct link to the issue in Jira (available in code)

### Table Formatting

Tables in generated documents use:
- **Font**: Aptos Narrow
- **Size**: 8pt
- **Borders**: Single black borders
- **Header**: Blue background (#365F91) with white text, bold
- **Margins**: 0.2cm on all sides

## Troubleshooting

### "Board not found" error
- Check that `JIRA_BOARD_NAME` in `.env` matches your Jira board name exactly
- Verify you have access to the board

### "Sprint not found" error
- Ensure the sprint name matches exactly (case-sensitive)
- Sprint must be associated with the board specified in `.env`

### "Failed to search epics" error
- Verify `JIRA_EPIC_FIELD` is correct for your Jira instance
- Check that your Jira user has permission to view custom fields

### "required environment variable TEAMS is not set" error
- Add `TEAMS=PROCESSING,STABLETEK` (or your team names) to your `.env` file
- Ensure team names match Jira Component names exactly

### No issues found for a team
- Verify the team name in `TEAMS` matches a Component name in Jira (check spelling and case)
- Ensure issues have the Component assigned in Jira
- Check that issues were actually "In Progress" during the specified month
- Use `-debug` flag to see detailed filtering information

### Document generation fails
- Check that the output directory exists and is writable
- Ensure there's enough disk space
- Verify the output filename doesn't conflict with an open file

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
