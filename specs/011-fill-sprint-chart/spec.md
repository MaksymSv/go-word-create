# Feature Specification: Fill the AI Assisted Work Items Chart workbook

**Feature Branch**: none, built on `dev` (retroactive spec written 2026-11-11)

**Status**: Implemented

**Input**: "I need a way to fill the AI Assisted Work Items Chart.xlsx with data calculated with get-sprint-label-report. Create a separate command (include it in make help), also it should be accessible via MCP."

## User Scenarios & Testing

### User Story 1 - Fill the empty sprint columns (Priority: P1)

A delivery lead keeps `proj-docs/AI Assisted Work Items Chart.xlsx` with one column per team and sprint. New sprint columns are added with headers and left empty. They run one command and the empty cells are filled with the label-report percentages, ready for the existing chart.

**Acceptance Scenarios**:

1. **Given** columns with a sprint header (row 1, e.g. `S26.20`) and team prefix (row 2, `ST`/`PC`) and empty data cells, **When** the command runs, **Then** each is filled using sprint `ST26.20` / `PC26.20` and component STABLETEK / PROCESSING respectively.
2. **Given** the `jira label` column (B), **When** values are written, **Then** each row gets the count % (rows titled `WI count`) or story-point % (rows titled `SP`) of the label in column B; rows with no label whose title starts with `total` get the share of issues / story points carrying any tracked label.
3. **Given** cells that already hold numbers, **When** the command runs, **Then** they are never changed, and running it twice changes nothing the second time.
4. **Given** a sprint that does not exist in Jira (yet), **When** the command runs, **Then** that column is skipped with a reason and the other columns are still filled.
5. **Given** `-dry-run`, **Then** the values are printed and nothing is written. **Given** `-output`, **Then** the original file is left untouched.

### User Story 2 - Same from an MCP client (Priority: P2)

`fill_sprint_chart` does the same for LLM clients. Without `write=true` it saves nothing and returns the filled workbook as base64; `write=true` updates the file in the server working directory (the only MCP tool besides `set_issue_label` that changes anything, and the only one that writes a file).

## Requirements

- **FR-001**: Edit the xlsx at the zip level; chart, styles, merges and all other parts must stay byte-identical. No new dependency.
- **FR-002**: Sprint name = team prefix + sprint header without its leading letters. Prefix map: `ST` -> `STABLETEK`, `PC` -> `PROCESSING` (`chartfill.DefaultPrefixes`).
- **FR-003**: Values are whole percentages (rounded), like the existing cells. Cells missing in the template are created next to their left neighbour's style.
- **FR-004**: Shared logic lives in `internal/chartfill`; the CLI (`cmd/fill-sprint-chart-xlsx`, `make run-chart`) and the MCP tool only wire it.
- **FR-005**: MCP `file` must be a local `.xlsx` path inside the server working directory.
- **FR-007**: Chart series ranges that start at the first data column (`C`) are widened to the last header column, so sprint columns added beyond `AD` are plotted; ranges are never shrunk and ranges on other sheets or single cells are left alone. The cached chart values are regenerated from the cells. Saving happens when cells were filled or a range was widened; a second run changes nothing.
- **FR-006**: A label in column B that is not in `REPORT_LABELS` is an error, not a silent zero.

## Known limits

- Only charts using the standard `c:` namespace prefix (what Excel writes) are synced.
- Column widths (`<cols>`) are not extended; columns after `AJ` get Excel's default width.
- The chart's position on the sheet is not moved.
