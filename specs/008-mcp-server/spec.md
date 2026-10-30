# Feature Specification: MCP Server for Jira Sprint & Label Reports

**Feature Branch**: none. Built directly on `dev` (commits `4ea2ec1` on 2026-10-29 and `cd0dfcd` on 2026-10-30) without a feature branch.

**Created**: 2026-10-29

**Status**: Implemented (retroactive spec written 2026-10-30 from the code)

**Input**: User description (reconstructed from commit messages and code): "Add an MCP server that exposes the existing Jira sprint/label reporting as typed tools, so LLM clients such as Claude Code, Claude Desktop and Codemie can answer sprint and AI-label questions directly. Keep it read-only except for one guarded tool that adds or removes a tracked label."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Explore Teams, Sprints and Sprint Issues (Priority: P1)

A Scrum Master asks their LLM client "what is still open in the latest PROCESSING sprint?". The assistant discovers the configured teams, finds the team's recent sprints and their IDs, and loads one sprint's issues, optionally narrowed to certain statuses, together with per-label counts and story points.

**Why this priority**: This is the foundation every other question builds on. Without team and sprint discovery, the assistant cannot resolve exact sprint IDs or names.

**Independent Test**: With a fake Jira source, call the team, sprint and sprint-issue tools in order and check the structured results; no real Jira is needed.

**Acceptance Scenarios**:

1. **Given** the server is connected, **When** the client asks for the configured teams, **Then** it receives each team's component and board name, the tracked report labels and the issue types included in reports.
2. **Given** a configured team, **When** the client asks for its sprints without a limit, **Then** it receives the 5 most recent sprints (id, name, state, start/end dates) of that team's board, newest first.
3. **Given** a sprint ID, **When** the client asks for its issues with a status filter, **Then** it receives only issues in those statuses (case-insensitive), the list of statuses available in the sprint, any requested statuses that matched nothing, and totals, unlabeled count and per-label counts and story points computed over the filtered set only.
4. **Given** no status filter, **When** the client asks for a sprint's issues, **Then** all issues are returned and issues carrying no tracked label are counted as unlabeled.

---

### User Story 2 - Label (AI-usage) Report by Sprint Name (Priority: P1)

A delivery lead asks "what share of Sprint 16 was AI-assisted?". The assistant requests the label report by exact sprint name and gets, per Jira component, each tracked label's issue count, story points and percentages, plus the labeled totals and the list of unlabeled issues.

**Why this priority**: This is the headline reporting question the project exists to answer, previously available only from the command line.

**Independent Test**: Call the report tool with one or more sprint names and compare the result against the same data from the `get-sprint-label-report -format full` command.

**Acceptance Scenarios**:

1. **Given** one or more exact sprint names, **When** the client requests the label report, **Then** it receives one report per sprint with one entry per Jira component (issues without a component under "No Component"), each with per-label count, count %, story points, story-points %, labeled-total percentages and the unlabeled issues.
2. **Given** a team name, **When** the client requests the report, **Then** only that team's board is queried and only that team's component is returned.
3. **Given** an empty sprint list or an unknown team, **When** the client requests the report, **Then** it receives a tool error that names the problem; for an unknown team the error lists the configured teams.

---

### User Story 3 - Change a Label Only After Explicit Approval (Priority: P2)

A user tells the assistant "mark PRJ-123 as ai-assisted-dev". The assistant first obtains a preview that changes nothing, shows it to the user, and only after the user approves repeats the call with confirmation to apply the change in Jira.

**Why this priority**: It is the only operation that modifies Jira data, so it must be safe by construction, but it is secondary to reading data.

**Independent Test**: Call the tool without confirmation and verify Jira is not called; call it with confirmation and verify exactly one Jira update is made.

**Acceptance Scenarios**:

1. **Given** a valid issue key, tracked label and action ("add" or "remove"), **When** the client calls the tool without confirmation, **Then** the result says "PREVIEW ONLY, nothing changed" and Jira is not called.
2. **Given** the same arguments with confirmation, **When** the client calls the tool, **Then** Jira is updated once and the result reports the change as applied.
3. **Given** an invalid issue key, an action other than add/remove, or a label not in the configured list, **When** the client calls the tool (with or without confirmation), **Then** it receives a tool error and Jira is not called.
4. **Given** Jira rejects the update, **When** the client calls with confirmation, **Then** the error names the issue key and is surfaced to the client as a tool error.

---

### User Story 4 - Sprint Issue Table for Several Sprints, Optionally as CSV (Priority: P2)

A user asks "give me the issues of ST26.20 and PC26.20 as CSV". The assistant requests the issue table for both sprints by name and receives, per sprint, the issue rows and, on request, the CSV text and a suggested file name.

**Why this priority**: It gives LLM clients the same data as the `get-sprint-issues` command, including CSV, but depends on the sprint-name handling shared with the CLI.

**Independent Test**: Request two sprints with CSV format and check that each sprint has its own rows, count, story-point total, CSV text and file name.

**Acceptance Scenarios**:

1. **Given** several exact sprint names (as separate entries or comma-separated within one entry), **When** the client requests the issue table, **Then** it receives one result per sprint with type, key, summary, epic, story points, status and URL for each issue, the issue count and story-point total.
2. **Given** a team and/or status filter, **When** the client requests the table, **Then** only that team's board and component and only issues in the listed statuses are returned, with the available statuses and any unmatched requested statuses reported per sprint.
3. **Given** the CSV format, **When** the client requests the table, **Then** each sprint additionally carries the CSV text and a suggested file name in the form `<sprint>_<YYYY-MM-DD>.csv`; the server writes no files.
4. **Given** a format other than "json" or "csv", or no sprint names, **When** the client requests the table, **Then** it receives a tool error.

---

### User Story 5 - Month "In Progress" Report (Priority: P3)

A manager asks "what did PROCESSING work on in October?". The assistant requests the month report and gets, per team, the issues that were "In Progress" at any time in that month, split into Closed and Open.

**Why this priority**: Useful but less frequent than sprint questions; it mirrors the existing month command.

**Independent Test**: Request a month with a fake Jira source returning mixed statuses and check the Closed/Open split and counts.

**Acceptance Scenarios**:

1. **Given** a month in `YYYY.MM` format, **When** the client requests the report, **Then** each team returns its issues split into Closed (current status exactly "Closed") and Open, with counts.
2. **Given** a malformed month, **When** the client requests the report, **Then** it receives a tool error explaining the expected format.

---

### User Story 6 - Easy, Safe Setup in an LLM Client (Priority: P3)

A developer wants Claude Code (or another MCP client) to use the server without copying secrets into client configuration, and wants the assistant to know which tool answers which question.

**Why this priority**: It makes the feature usable but adds no new reporting capability.

**Independent Test**: Build the binary, start Claude Code in the repository and confirm the server is listed and its tools are callable.

**Acceptance Scenarios**:

1. **Given** the repository, **When** Claude Code starts, **Then** the project-scoped client configuration registers the server using the built binary and the repository as the working directory, with no secrets in the file.
2. **Given** the client starts the server from another directory, **When** a working directory flag is supplied, **Then** the server switches into it before loading `.env`.
3. **Given** a missing required setting, **When** the server starts, **Then** it exits with an error naming the missing variable.
4. **Given** the companion skill, **When** a user asks a sprint or label question, **Then** the assistant is guided to the right tool, the metric semantics, the write protocol and a CLI fallback.

---

### Edge Cases

- Tool failures (unknown team, bad month, invalid label, Jira errors) are returned as tool error results, not as protocol failures, so the assistant can read and react to them.
- Team and sprint names are case-sensitive; the "team not found" error lists the configured team names.
- A status filter that matches nothing returns an empty issue list and names the unmatched statuses, so the assistant can tell a typo from a genuinely empty result.
- Label membership is non-exclusive: an issue with two tracked labels is counted under both, so label rows do not sum to the total.
- Failure to load epic names does not fail a sprint query; epic names stay empty.
- The sprint-issue tool applies team scoping by Jira component after fetching, so an issue in several components is included for any of them.
- The sprint-ID tool does not apply the configured issue-type filter, whereas the name-based sprint, label-report and month tools do.
- The server must not print anything but protocol messages to standard output; all logs go to standard error.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a stdio-based MCP server that registers the tools `list_teams`, `list_sprints`, `get_sprint_issues`, `sprint_issues`, `sprint_label_report`, `month_issues` and `set_issue_label`, and supplies usage instructions to the client.
- **FR-002**: System MUST mark every tool except `set_issue_label` as read-only.
- **FR-003**: `list_teams` MUST return the configured teams (component and board name), the tracked report labels and the issue types included in reports.
- **FR-004**: `list_sprints` MUST return the most recent sprints of the named team's board, newest first, with a default limit of 5, and MUST reject unknown team names with an error listing the configured teams.
- **FR-005**: `get_sprint_issues` MUST require a positive sprint ID and return the sprint's issues with key, summary, epic, implementer, story points, type, status, URL and active tracked labels.
- **FR-006**: `get_sprint_issues` and `sprint_issues` MUST support a case-insensitive status filter and report available statuses and requested statuses that matched no issue.
- **FR-007**: `get_sprint_issues` MUST compute totals, unlabeled count and per-label count and story points over the filtered issues only.
- **FR-008**: `sprint_issues` MUST accept one or more exact sprint names (separate entries or comma-separated), an optional team scope and an optional status filter, and return per sprint the rows, count and story-point total.
- **FR-009**: `sprint_issues` MUST support a CSV format that returns the CSV text and a suggested `<sprint>_<YYYY-MM-DD>.csv` file name per sprint, and MUST NOT write files.
- **FR-010**: `sprint_label_report` MUST accept one or more exact sprint names and an optional team scope, and return per sprint and per component each tracked label's count, count %, story points, story-points %, labeled-total percentages and unlabeled issues, with percentages rounded to one decimal.
- **FR-011**: `month_issues` MUST accept a `YYYY.MM` month and optional team, and return per team the issues that were "In Progress" during the month, split into Closed (status exactly "Closed") and Open, with counts.
- **FR-012**: `set_issue_label` MUST validate the issue key format, the action ("add" or "remove") and that the label is one of the configured report labels before doing anything else.
- **FR-013**: `set_issue_label` MUST, unless explicit confirmation is given, return a preview and MUST NOT call Jira; with confirmation it MUST apply the change exactly once.
- **FR-014**: System MUST return handler failures as tool error results with context (team, sprint, month or issue key where applicable).
- **FR-015**: System MUST reuse the shared business logic for the sprint view, label aggregation, sprint-name parsing and CSV output rather than duplicating it for the server.
- **FR-016**: The server binary MUST read configuration from the same `.env` settings as the other commands, accept `-workdir` (default `.`) to locate `.env`, and accept `-debug` for verbose logging.
- **FR-017**: The server MUST send all log output to standard error so that standard output carries only protocol messages.
- **FR-018**: The repository MUST ship a project-scoped client configuration (`.mcp.json`) that registers the server without secrets, and a companion skill that maps questions to tools, explains metric semantics and the write protocol, and documents a CLI fallback.
- **FR-019**: The build tooling MUST provide a target that builds the server binary into `bin/`.

### Key Entities

- **Team**: A configured pairing of a Jira component name and a Jira board name.
- **Sprint**: A board's sprint with an ID, name, state and optional start/end dates.
- **Sprint Issue**: An issue in a sprint with key, summary, epic, implementer, story points, type, status, URL and the subset of its labels that are tracked.
- **Tracked Label**: One of the configured report labels (default `ai-assisted`, `ai-assisted-ba`, `ai-assisted-dev`, `ai-assisted-qa`).
- **Label Bucket**: An issue count with the sum of their story points, per tracked label, plus totals and unlabeled.
- **Label Report**: Per sprint and component, per-label counts, story points and percentages, labeled totals and unlabeled issues.
- **Write Preview**: The no-op result of `set_issue_label` that describes the change to be confirmed.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An LLM client can answer "which labels were used in sprint X and how many story points" using one tool call with the sprint name.
- **SC-002**: No Jira data changes unless `set_issue_label` is called with explicit confirmation; calls without confirmation and calls with invalid input make zero Jira write requests (covered by automated tests).
- **SC-003**: All seven tools are discoverable over the protocol with correct read-only/write annotations, and handler errors arrive as tool errors (covered by an end-to-end protocol test).
- **SC-004**: The sprint and label figures returned by the server match those of the corresponding command-line tools for the same sprint names.
- **SC-005**: Registering the server in Claude Code requires only building the binary and starting the client in the repository; no credentials appear in any committed file.

## Assumptions

- Users are internal team members who already hold Jira credentials in a local `.env` file; the server runs locally as a child process of the LLM client.
- The server reuses the existing Jira client, so it targets Jira Cloud only and inherits its field configuration (epic, story points, components).
- Data is fetched live on every call; there is no caching or storage.
- Percentages and label counts follow the existing label-report semantics (non-exclusive labels, percentages relative to the component total).
- Standard output of the server process is reserved for protocol traffic, so file writing and console printing are out of scope; CSV is returned as text for the client to display or save.
- The `get_sprint_issues` tool intentionally mirrors the web dashboard's sprint view, including its lack of issue-type filtering.
- The write tool is limited to adding or removing the tracked labels; no other Jira field can be changed through the server.
