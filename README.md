# Go Word Create

A Go project that generates Word documents from Jira issues. Supports fetching issues by sprint or by month, with filtering by issue type.

## Project Overview

This project provides multiple command-line tools to fetch Jira issues and generate Word documents:

- **Server**: HTTP server for generating Word documents on demand
- **Get Sprint Issues**: Fetch all issues from a specific sprint and export to Word
- **Get Month Issues**: Fetch all issues that were "In Progress" during a specific month and export to Word

## Features

- 📊 **Jira Integration**: Connect to Jira Cloud to fetch issues, sprints, and epic information
- 📄 **Word Document Generation**: Create formatted Word documents with tables containing issue details
- 🔍 **Issue Filtering**: Filter by issue type (Bug, Feature, Task, etc.)
- 📅 **Month-based Filtering**: Find all issues that transitioned to "In Progress" during a specific month
- 🖥️ **HTTP Server**: REST API for on-demand document generation
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
make build-server
make build-month
make build-sprint

# Show all available targets
make help
```

### Using Go directly

```bash
# Build server
go build -o bin/server ./cmd/server

# Build month issues fetcher
go build -o bin/get-month-issues ./cmd/get-month-issues-from-jira

# Build sprint issues fetcher
go build -o bin/get-sprint-issues ./cmd/get-sprint-issues-from-jira
```

## Running

### Server

Start the HTTP server (default port 8080):
```bash
make run
```

Or run directly:
```bash
./bin/server
```

The server will respond to HTTP requests for document generation.

### Get Month Issues

Fetch all issues that were "In Progress" during October 2025:
```bash
make run-month MONTH=2025.10
```

Or run directly:
```bash
./bin/get-month-issues -month="2025.10" -output="october-report.docx"
```

#### Flags:
- `-month="YYYY.MM"` (required): Month to fetch issues from (e.g., "2025.10")
- `-output="file.docx"` (optional): Output file name (default: from .env)
- `-debug`: Print issues to console instead of generating Word document

### Get Sprint Issues

Fetch all issues from a specific sprint:
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
│   ├── server/              # HTTP server
│   ├── get-sprint-issues-from-jira/   # Sprint issues fetcher
│   └── get-month-issues-from-jira/    # Month issues fetcher
├── internal/
│   ├── config/              # Configuration loading from .env
│   ├── jiraservice/         # Jira API client and issue fetching
│   ├── word/                # Word document generation
│   └── wordtable/           # Table formatting utilities
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

The project uses two custom Jira fields:

1. **Epic Link** (default: `customfield_10014`): Links issues to epics
2. **Story Points** (default: `customfield_10015`): Stores story point estimates

These IDs may vary in your Jira instance. Use the API endpoint mentioned above to find the correct IDs.

### Output File Format

Generated Word documents include:
- **Type**: Issue type (Bug, Feature, Task)
- **Key**: Jira issue key (e.g., PROJ-123)
- **Summary**: Issue title
- **Epic**: Epic name the issue belongs to
- **Story Points**: Story point estimate
- **Status**: Current issue status
- **URL**: Direct link to the issue in Jira

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

### Document generation fails
- Check that the output directory exists and is writable
- Ensure there's enough disk space
- Verify the output filename doesn't conflict with an open file

## Dependencies

- `github.com/andygrunwald/go-jira` - Jira API client
- `github.com/carmel/gooxml` - Word document generation
- `github.com/joho/godotenv` - Environment variable loading

## License

[Add your license here]

## Contributing

1. Create a feature branch (`git checkout -b feature/amazing-feature`)
2. Commit your changes (`git commit -m 'Add amazing feature'`)
3. Push to the branch (`git push origin feature/amazing-feature`)
4. Open a Pull Request

## Support

For issues and questions, please create an issue in the project repository.
