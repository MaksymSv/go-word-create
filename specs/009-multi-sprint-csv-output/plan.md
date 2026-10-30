# Implementation Plan: Multi-Sprint Input + CSV Output (as built)

**Branch**: none — built directly on `dev` (commits `608aa71` 2026-07-07, `4ea2ec1` 2026-10-29, `b1fe6fc` 2026-10-30, `cd0dfcd` 2026-10-30) | **Date**: 2026-10-30 (retroactive) | **Spec**: [spec.md](spec.md)

**Status**: Implemented (retroactive plan written 2026-10-30 from the code). No tasks.md exists; the work was done through direct chat with an LLM, not through Spec Kit.

**Input**: Feature specification from `specs/009-multi-sprint-csv-output/spec.md`

## Summary

Two small extensions to the sprint-oriented command line tools:

1. **Multi-sprint input** — `-sprint "A,B"` is parsed by one shared function, `jiraservice.ParseSprintNames`, and both `get-sprint-issues` and `get-sprint-label-report` loop over the resulting names.
2. **CSV output** — `get-sprint-issues` gains `-format docx|csv` and `-output-dir`. In `csv` mode it writes one file per sprint through the new `internal/issuecsv` package, which the MCP server's `sprint_issues` tool also uses.

No new Jira calls, no new dependencies, no new binaries. The Makefile `run-sprint` target passes the new options through.

History: `608aa71` added the list parsing inside `cmd/get-sprint-label-report` (with its own `main_test.go`). `4ea2ec1` moved the parser into `internal/jiraservice/sprintnames.go` (test moved to `sprintnames_test.go`, the old `main_test.go` was deleted) and adopted it in `get-sprint-issues`. `b1fe6fc` is titled "Added multisprint parameter support" but its diff only adds the Status column to the debug output and the Word table. `cd0dfcd` added CSV.

## Technical Context

**Language/Version**: Go 1.24.0 (module `go-word-create`)

**Primary Dependencies**: No new dependencies. CSV uses stdlib `encoding/csv`, `regexp`, `path/filepath`. Existing `go-jira` (via `internal/jiraservice`) and `gooxml` (via `internal/word`) are unchanged.

**Storage**: Files only — one `.docx` (docx mode) or one `.csv` per sprint (csv mode). No state kept between runs.

**Testing**: `go test ./...`. Covered: `TestParseSprintNames` (single, comma list, whitespace, trailing comma, blank input) in `internal/jiraservice/sprintnames_test.go`; `TestFileName` and `TestWrite` (header, quoting, one-decimal story points) in `internal/issuecsv/issuecsv_test.go`. Not covered: both `cmd/` mains (flag handling, the per-sprint loop, directory creation, `writeCSV`), duplicate names, file-name collisions.

**Target Platform**: macOS / Linux, local command line.

**Project Type**: Command line tools sharing `internal/` packages.

**Performance Goals**: None specified; cost is dominated by Jira calls, repeated per sprint (see design notes).

**Constraints**: Logs on stderr, results on stdout or in files; sprint names are matched exactly.

**Scale/Scope**: A handful of sprints per run; sprints of a few hundred issues.

## Constitution Check

*Checked against `.specify/memory/constitution.md` v1.1.0. Constitution v1.2.0 (2026-10-30) later exempted stdout-only commands such as `get-sprint-label-report` from the `-debug` requirement, so the GAP row below no longer applies; the text is kept as written.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. CLI-First | PASS | New behaviour is reachable through flags (`-sprint`, `-format`, `-output-dir`, `-debug`). Flags use kebab-case. The MCP server reuses `issuecsv` and does not duplicate it. |
| I. Debug mode on every command | GAP | `get-sprint-issues` has `-debug`. `get-sprint-label-report` has no `-debug` flag and the Makefile target has no `LOGONLY`; it only ever prints to the console, so the intent is met in practice but the letter of the principle is not. |
| II. Environment-Driven | PASS | No new variables. The Word output file still defaults to `DEFAULT_OUTPUT_FILE`. `-output-dir` is a flag with default `.`. |
| III. Package Separation | PASS | Parsing is in `internal/jiraservice`, CSV in `internal/issuecsv`; `cmd/` holds wiring. One blemish: `writeCSV` (create directory, open, write, close) lives in `cmd/get-sprint-issues-from-jira/main.go`, and the loop over sprints is also in `main`. The constitution lists `internal/jiraservice` as a Jira client; a pure string parser living there is a loose fit but causes no forbidden import. `issuecsv` imports `jiraservice`, which is allowed. |
| IV. Consistent Document Output | PASS for Word, N/A for CSV | The Word table still goes through `internal/word` with `DefaultConfig()`; only the column set and widths changed (Status added). CSV is outside the Word formatting standard. |
| V. Observability | PARTIAL | Errors name the sprint. Debug output lists issues. Jira lookups log to stderr (`Found sprint ... with ID ...`, board warnings). The `FindSprintInBoards` error does not say which boards were searched. The CSV confirmation line ("Created CSV ...") goes to stdout, which fits the stdout-for-results rule. |
| Tech Stack | PASS | No new libraries; stdlib only. |
| Build Tooling | PASS | `make run-sprint` and `make help` updated. No new binary, so no new target. |

No justified violations; the `-debug` gap above is a known deviation, not an approved one.

## Project Structure

### Documentation (this feature)

```text
specs/009-multi-sprint-csv-output/
├── spec.md      ← retroactive
└── plan.md      ← this file (retroactive; no tasks.md)
```

### Source Code (repository root)

```text
cmd/
├── get-sprint-issues-from-jira/main.go   ← -sprint list, -format, -output-dir, writeCSV, per-sprint loop
└── get-sprint-label-report/main.go       ← -sprint list, per-sprint loop with "=== Sprint: X ===" heading

internal/
├── jiraservice/
│   ├── sprintnames.go                    ← ParseSprintNames (shared parser)
│   ├── sprintnames_test.go
│   └── jira.go                           ← GetSprintIssues / FindSprintInBoards (unchanged, called once per sprint)
├── issuecsv/
│   ├── issuecsv.go                       ← FileName, Write (shared with MCP)
│   └── issuecsv_test.go
└── mcpserver/tools.go                    ← sprint_issues tool uses issuecsv.Write / FileName (see MCP spec)

Makefile                                  ← run-sprint passes -format, -output-dir; help text updated
.gitignore                                ← *.csv added
README.md                                 ← partially updated (see Known inconsistencies)
.claude/skills/jira-sprint-reports/       ← skill mentions the CSV option
```

**Structure Decision**: Parsing and CSV rendering were extracted into `internal/` so the MCP server and the command line tools share one implementation. Everything else stayed in the two existing `cmd/` packages.

## As-Built Design Notes

### Sprint list parsing

`ParseSprintNames(raw)` trims the whole string, rejects empty input (`sprint flag is empty`), splits on `,`, trims each part and rejects any empty part (`sprint list contains an empty sprint name`). It does not de-duplicate, sort or validate against Jira. Both commands call it once, before building anything else that needs the network (the sprint issues command builds the Jira client first, then parses; either way no Jira request is made before parsing).

### Per-sprint processing (both commands)

For each name in order, the command calls `JiraService.GetSprintIssues(projectKey, sprint, allBoardNames, issueTypes)`. That call, per sprint, does: `FindSprintInBoards` (iterates boards in `TEAMS` order, reads all sprints of each board, returns the first exact name match; a failing board is logged and skipped), `LoadEpics` for the project (repeated for every sprint), then `LoadIssuesFromSprint` filtered by `ISSUE_TYPES`. Any error ends the run immediately (`log.Fatalf` in the sprint issues command, message plus `os.Exit(1)` in the label report). There is no skip-and-continue behaviour.

### get-sprint-issues: formats

- Validation order: empty `-sprint`, then unknown `-format`, then Jira client creation, then name parsing.
- **docx** (default): issues of all sprints are appended into one slice; after the loop a single document with one table is created (columns Type, Key, Summary, Epic, Story Points, Status; widths 5.5/9.0/38.0/29.0/5.5/13.0) and saved to `-output`. Sprint boundaries are not visible in the document.
- **csv**: inside the loop, immediately after a sprint's issues are loaded, `filepath.Join(outputDir, issuecsv.FileName(sprint, time.Now()))` is written via `writeCSV` (creates the directory with mode 0755, creates or truncates the file, writes, and returns the close error if no earlier error). The path and issue count are printed. The combined slice is still built but unused; no Word document is created.
- `FileName` replaces each run of characters outside `[A-Za-z0-9._-]` with `_`. The date is `time.Now()` in local time.
- `Write` emits the header and rows through `encoding/csv` (comma separator, `\n` line ends, no BOM) with story points formatted `%.1f`.

### `-debug`

In `get-sprint-issues`, `-debug` prints per sprint the heading (only when several sprints), `Found N issues in sprint '<name>'`, and one line per issue (`type|key|summary(80)|epic(40)|story points|status`, summary and epic truncated with `...`), then the total. In debug mode no CSV is written and no Word document is created. The flag help text still says only "print data without generating Word document". `get-sprint-label-report` has no `-debug`.

### Label report

For each sprint, `labelreport.Aggregate(issues, cfg.ReportLabels)` is printed in `short` or `full` console form. With more than one sprint, each block is preceded by `=== Sprint: <name> ===`. `-format` here means `short|full`, not `docx|csv`.

### Makefile

`run-sprint` requires `SPRINT`, then builds the command line with optional `-output`, `-format`, `-output-dir`, `-debug` from `OUTPUT`, `FORMAT`, `CSV_DIR`, `LOGONLY`. `run-sprint-label-report` passes `FORMAT` as `-format` (`short|full`) and has no `LOGONLY`. The variable `FORMAT` therefore means different things in the two targets.

## Known inconsistencies (found while writing this plan)

1. **README, "Get Sprint Issues"**: says "Uses the board name from the first team entry in `TEAMS`". The code passes all configured boards (`cfg.GetAllBoardNames()`) and takes the first board that has the sprint.
2. **README, "Get Sprint Issues"**: describes a single sprint name and lists only `-sprint`, `-output`, `-debug`. It does not document the comma-separated list, `-format` or `-output-dir`; the Makefile help and the command's own flag help do.
3. **README, "Get Sprint Label Report"**: says "If one board fails, that team is skipped with an error message and the remaining teams are still processed". The code exits on the first sprint that fails, and board read failures during sprint lookup are only logged as warnings.
4. **README sprint-issues Word description** mentions no Status column or CSV; the "Table Columns" section still lists five columns (Type, ID, Description, Epic, SP) and does not mention Status.
5. **Makefile help** for `run-sprint-label-report` shows a single sprint name although a list works; README shows the list form.
6. **Flag help for `-debug`** in `get-sprint-issues` omits that it also suppresses CSV files.
7. **Commit `b1fe6fc`** message ("Added multisprint parameter support") does not match its diff (adds the Status column); the multi-sprint parsing for this command arrived in `4ea2ec1`.
8. **Constitution I/V (v1.1.0)** required a debug mode on every command; `get-sprint-label-report` has none. Resolved by wording in v1.2.0, which exempts stdout-only commands.
9. **Docs listing**: `README.md` Dependencies section does not list `modelcontextprotocol/go-sdk`; unrelated to this feature but noticed while checking the README.
10. **Typo** "separatred" in the README label-report flag description.

## Limitations Worth Knowing (not fixed)

- Duplicate sprint names and different names mapping to the same file name are not detected (see spec edge cases).
- `LoadEpics` runs once per sprint instead of once per run.
- A failure in the middle of a CSV run leaves earlier files in place and later sprints unprocessed.
- No tests for the `cmd/` mains or for `writeCSV` (directory creation, overwrite, close error).
