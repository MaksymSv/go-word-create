# Feature Specification: Multi-Sprint Input + CSV Output

**Feature Branch**: none — built directly on `dev` without a feature branch (commits `608aa71`, `4ea2ec1`, `b1fe6fc`, `cd0dfcd`)

**Created**: 2026-07-07 (label report) to 2026-10-30 (CSV); this spec written 2026-10-30

**Status**: Implemented (retroactive spec written 2026-10-30 from the code)

**Input**: User description (reconstructed from commit messages): "Added multisprint format into labels report module", "Added multisprint support for get-sprint", "Added multisprint parameter support", "Added support of CSV output format for get-sprint".

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Report on Several Sprints in One Run (Priority: P1)

A delivery lead wants the label/AI-usage report for the current sprint of each team (for example `ST26.20` for one team and `PC26.20` for another) without running the tool once per sprint. They pass a comma-separated list of sprint names and get one report per sprint in a single run.

**Why this priority**: Teams name their sprints differently, so a combined view always needs more than one sprint name. This is the original reason for the feature and the one every other story builds on.

**Independent Test**: Run the label report with two sprint names and confirm each sprint's report appears under its own heading, in the order given.

**Acceptance Scenarios**:

1. **Given** two valid sprint names separated by a comma, **When** the user runs the label report, **Then** the output contains one report per sprint, in the order given, each preceded by a heading with the sprint name.
2. **Given** a single sprint name, **When** the user runs the label report, **Then** the output is the same as before multi-sprint support existed, with no sprint heading.
3. **Given** a sprint list with spaces around names (for example `" Sprint 16 , ST26.13 "`), **When** the list is read, **Then** the surrounding whitespace is ignored and each name is used exactly as trimmed.

---

### User Story 2 - Export Sprint Issues for Several Sprints to One Word Document (Priority: P1)

A delivery lead wants the issue table of several sprints in a single Word document, using the standard table format, for the monthly delivery report.

**Why this priority**: This is the established output of the sprint issues command; multi-sprint input must not break it.

**Independent Test**: Run the sprint issues command with two sprint names and the default format; open the document and confirm it holds one table with the issues of both sprints.

**Acceptance Scenarios**:

1. **Given** two valid sprint names, **When** the user runs the sprint issues command with the default format, **Then** one Word document is produced containing a single table with the issues of every listed sprint, one after another, with columns Type, Key, Summary, Epic, Story Points and Status.
2. **Given** debug mode is requested, **When** the command runs, **Then** each sprint's issues are printed to the console under a heading with the sprint name and no document is written.

---

### User Story 3 - Export Each Sprint to Its Own CSV File (Priority: P2)

An analyst wants to load sprint data into a spreadsheet or another tool. They ask for CSV output and get one CSV file per sprint, named after the sprint and the date of the export, in a directory they choose.

**Why this priority**: It opens the data to downstream tools, but the Word export already covers the reporting need.

**Independent Test**: Run the sprint issues command with CSV format and two sprint names; confirm two files exist, named `<sprint>_<YYYY-MM-DD>.csv`, each with a header row and one row per issue of that sprint only.

**Acceptance Scenarios**:

1. **Given** two valid sprint names and CSV format, **When** the command runs, **Then** two CSV files are written, one per sprint, and the command reports the path and issue count of each.
2. **Given** an output directory that does not exist, **When** the command runs with CSV format, **Then** the directory is created.
3. **Given** a sprint name containing characters unsafe for file names (spaces, slashes, colons), **When** the file is named, **Then** each run of unsafe characters is replaced by a single underscore (`Sprint 16` becomes `Sprint_16_<date>.csv`).
4. **Given** issue summaries containing commas or quotes, **When** the CSV is written, **Then** the values are quoted and escaped so the file round-trips through a standard CSV reader.
5. **Given** CSV format and debug mode together, **When** the command runs, **Then** issues are printed to the console and no file is written.

---

### User Story 4 - Run From the Makefile (Priority: P3)

A user prefers `make` over remembering flags. They pass the sprint list, the format and the output directory as make variables.

**Why this priority**: Convenience only; the binaries work without it.

**Independent Test**: Run `make run-sprint SPRINT="A,B" FORMAT=csv CSV_DIR=out` and confirm it behaves like the equivalent direct invocation.

**Acceptance Scenarios**:

1. **Given** `SPRINT`, optionally `OUTPUT`, `FORMAT`, `CSV_DIR` and `LOGONLY` are set, **When** `make run-sprint` runs, **Then** the matching flags (`-sprint`, `-output`, `-format`, `-output-dir`, `-debug`) are passed to the binary and variables left unset pass nothing.
2. **Given** `SPRINT` is missing, **When** `make run-sprint` runs, **Then** it prints a usage message and stops.

---

### Edge Cases

- **Empty or whitespace-only sprint list**: the command stops with the error "sprint flag is empty". For the sprint issues command, an empty `-sprint` flag is rejected earlier with "Sprint name is required" and the usage text.
- **Empty entry in the list** (`"Sprint 16,"` or `"A,,B"`): the command stops before contacting Jira, with the error "sprint list contains an empty sprint name".
- **Sprint not found in any configured board**: the command stops with an error naming the sprint ("sprint '<name>' not found in any of the provided boards"). Sprint names are matched exactly, including case.
- **Failure on one of several sprints**: the run stops at the first failing sprint with a non-zero exit code. Output already produced for earlier sprints (console report, CSV files) stays in place; later sprints are not processed. Sprints are not skipped.
- **Board that cannot be read while searching**: a warning is logged and the search continues with the next configured board.
- **Duplicate sprint names in the list**: no de-duplication happens. The sprint is processed twice: its issues appear twice in the Word table, and its CSV file is written twice to the same path (the second write replaces the first).
- **Two different sprint names that map to the same file name** (for example `A/B` and `A_B`): the later file overwrites the earlier one.
- **Sprint with no issues**: a CSV with only the header row is written, and the confirmation reports 0 issues.
- **Unknown output format**: the command stops with an error naming the allowed values `docx` and `csv`.
- **Word-only and CSV-only options**: `-output` applies only to Word output and is ignored for CSV; `-output-dir` applies only to CSV output and is ignored for Word.
- **Same date, repeated run**: the file name carries the date only, so a second run on the same day replaces the earlier files.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The sprint issues command and the sprint label report command MUST accept a comma-separated list of sprint names in the `-sprint` flag; a single name MUST keep working as before.
- **FR-002**: System MUST trim whitespace around each sprint name and MUST reject an empty list or any empty entry, with a message that says what is wrong, before any call to Jira.
- **FR-003**: System MUST resolve each sprint name by exact, case-sensitive match against the sprints of all configured team boards, using the first board that contains it.
- **FR-004**: System MUST process sprints in the order given and, when more than one sprint is given, MUST label each sprint's console output with the sprint name.
- **FR-005**: System MUST stop with a non-zero exit code and a message naming the sprint when a sprint cannot be found or its issues cannot be loaded.
- **FR-006**: The sprint issues command MUST support an output format option with the values `docx` (default) and `csv`, and MUST reject any other value.
- **FR-007**: In `docx` format the command MUST write one Word document holding the issues of all requested sprints in a single table with the columns Type, Key, Summary, Epic, Story Points, Status, in the project's standard table style, to the file given by `-output` (default from configuration).
- **FR-008**: In `csv` format the command MUST write one file per sprint, in the directory given by `-output-dir` (default: current directory), creating the directory when missing.
- **FR-009**: Each CSV file MUST be named `<sprint>_<YYYY-MM-DD>.csv`, where the date is the day of the export and every run of characters other than letters, digits, dot, underscore and hyphen in the sprint name is replaced by one underscore.
- **FR-010**: Each CSV file MUST begin with the header row `Type,Key,Summary,Epic,Story Points,Status` followed by one row per issue; story points MUST be written with one decimal place; fields MUST be quoted and escaped per standard CSV rules.
- **FR-011**: For each CSV file written, the command MUST print its path and issue count.
- **FR-012**: With debug mode on, the sprint issues command MUST print each sprint's issues (type, key, summary, epic, story points, status) and total count to the console and MUST NOT write any file, in either format.
- **FR-013**: The Makefile target `run-sprint` MUST accept `SPRINT` (required), `OUTPUT`, `FORMAT`, `CSV_DIR` and `LOGONLY`, and pass only those that are set; `make help` MUST list them.
- **FR-014**: Generated CSV files MUST be excluded from version control.
- **FR-015**: The sprint-name parsing and the CSV rendering (file name and content) MUST live in shared packages, so the command line tools and the MCP server produce identical results.

### Key Entities *(include if feature involves data)*

- **Sprint List**: The ordered list of sprint names parsed from one `-sprint` value. Whitespace-trimmed, never empty, duplicates kept.
- **Sprint Issue**: One Jira issue of a sprint with type, key, summary, epic name, story points and status (plus URL, labels, components and implementer, which the CSV and Word outputs do not include).
- **CSV Export File**: One file per sprint, named from the sprint name and export date, holding the header row and the sprint's issues.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can get the label report for N sprints from one command invocation instead of N invocations.
- **SC-002**: A CSV export of a sprint opens in a standard spreadsheet application with six correctly separated columns, including summaries that contain commas, quotes or non-ASCII text.
- **SC-003**: A malformed sprint list (empty, or with an empty entry) is rejected before any request is sent to Jira, in under 1 second.
- **SC-004**: The CSV produced by the command line tool and by the MCP server for the same sprint and date is identical in file name and content.

## Assumptions

- Sprint names do not contain commas, since the comma is the list separator and there is no escaping.
- Sprint names are unique across the configured boards; when the same name exists on several boards, the first board in configuration order wins.
- The user runs the tool where it can write to the chosen output directory; no overwrite protection is needed because file names carry the date and repeated same-day runs are meant to replace earlier files.
- The date in the file name is the local date of the machine running the command.
- Word output keeps its existing behaviour (single combined table, no per-sprint section headings), because the report consumers only need the combined list.
- CSV carries the same six columns as the Word table; the extra fields available to the MCP server (URL, labels, implementer) are out of scope.
- The sprint label report prints to the console only; it has no Word, CSV or debug option.
