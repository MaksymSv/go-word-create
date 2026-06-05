# Tasks: Unified Word Table Styling

**Input**: Design documents from `specs/003-unified-word-table-styling/`

**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅

**Organization**: Tasks grouped by user story. US3 (configurable headers) is fully delivered by the Foundational phase + US1 work and has no separate implementation phase.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US4)

---

## Phase 1: Setup

No project initialization required — this feature modifies existing files only.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Update `TableConfig` to the new schema. All user story phases depend on this.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T001 Update `TableConfig` struct and `DefaultConfig()` in `internal/word/config.go`: remove fields `CellMargin float64` and `Width int`; add `CellTopBottomMargin measurement.Distance`, `ColumnHeaders []string`, `ColumnWidths []float64`, `CenterColumns []int`; set `DefaultConfig()` values to `HeaderBackgroundColor=RGB(0x00,0x70,0xC0)`, `CellTopBottomMargin=57*measurement.Dxa`, `ColumnHeaders=["Type","ID","Description","Epic","SP"]`, `ColumnWidths=[5.5,13.3,41.5,34.2,5.4]`, `CenterColumns=[0,1,4]`

**Checkpoint**: `TableConfig` schema is final — user story implementation can now begin.

---

## Phase 3: User Story 1 — Match Reference Document Table Design (Priority: P1) 🎯 MVP

**Goal**: Tables in all generated documents visually match `EPAM Delivery Report - 2026-05.docx` — correct header blue (#0070C0), 8pt font, auto-color 1pt borders, top/bottom-only cell margins, and proportional column widths.

**Independent Test**: Run `make run-month MONTH=2026.05`; open the output `.docx` and compare side-by-side with the reference document. Header row must be #0070C0 blue (not dark blue), data font 8pt Aptos Narrow, no visible left/right cell padding difference from the reference.

### Implementation for User Story 1

- [x] T002 [US1] Refactor `NewTable()` in `internal/word/table.go`: change signature to `NewTable(doc *document.Document, config TableConfig) *Table`; store `config` in the `Table` struct; set table width to `113%` via `SetWidthPercent(113)`; set layout fixed via `SetLayout(wml.ST_TblLayoutTypeFixed)`; set all borders `single 1pt auto-color` via `borders.SetAll(wml.ST_BorderSingle, color.Auto, measurement.Point)`; set table-level top and bottom cell margins to `57 dxa` by writing directly to `table.Properties().X().TblCellMar` (raw XML); set table indent to `−584 dxa` by writing to `table.Properties().X().TblInd` (raw XML, negative value)
- [x] T003 [US1] Refactor `AddHeaderRow()` in `internal/word/table.go`: remove the `headers []string` parameter; add table row, set row height `196 twips` (AtLeast rule) via `headerRow.Properties().SetHeight`; mark row as repeating header by setting `headerRow.X().TrPr.TblHeader = []*wml.CT_OnOff{wml.NewCT_OnOff()}`; for each index `i` in `config.ColumnHeaders`: add cell, set cell width percentage from `config.ColumnWidths[i]` via `cell.Properties().SetWidthPercent(pct)`, set shading to `config.HeaderBackgroundColor`, add paragraph (center-aligned), add run with label text, color `config.HeaderTextColor`, size `8pt`, font `Aptos Narrow`; do NOT call `SetBold(true)` (reference uses no bold on header text)
- [x] T004 [US1] Refactor `AddDataRow()` in `internal/word/table.go`: add row, set row height `300 twips` (AtLeast); for each cell, set cell width from `config.ColumnWidths[i]` (if index in range); center-align if `i` is in `config.CenterColumns`; add run with 8pt Aptos Narrow; remove all `setCellMargins(cell)` calls
- [x] T005 [P] [US1] Remove dead code from `internal/word/table.go` and `internal/word/config.go`: delete the `setCellMargins()` helper function; delete the `WithConfig()` constructor function (callers now use `NewTable` directly with a config)
- [x] T006 [US1] Update `cmd/get-month-issues-from-jira/main.go`: in `addTableToDocument`, replace `word.NewTable(&doc.WordDocument)` with `word.NewTable(&doc.WordDocument, word.DefaultConfig())`; remove the `headers := []string{...}` local variable; change `closedIssuesTable.AddHeaderRow(headers)` to `closedIssuesTable.AddHeaderRow()`
- [x] T007 [P] [US1] Update `cmd/get-sprint-issues-from-jira/main.go`: build a `word.TableConfig` with `ColumnHeaders = []string{"Type", "Key", "Summary", "Epic", "Story Points"}` and `ColumnWidths = [5.5, 10.0, 44.5, 34.5, 5.5]`; pass it to `word.NewTable`; change `table.AddHeaderRow(headers)` to `table.AddHeaderRow()`; remove the `headers := []string{...}` local variable
- [x] T008 [P] [US1] Update `cmd/get-sprint-label-report/main.go`: create three `word.TableConfig` values — (a) short format 5-col: `ColumnHeaders=["Label","Count","Count,%","Total SP","Total SP,%"]`, `ColumnWidths=[25,10,10,10,10]`, `CenterColumns=[1,2,3,4]`; (b) full format 8-col: `ColumnHeaders=["Label","Count","Count,%","Total SP","Total SP,%","Key","Summary","SP"]`, `ColumnWidths=[13,6,6,6,6,10,47,6]`, `CenterColumns=[1,2,3,4,7]`; (c) unlabeled sub-table 3-col: `ColumnHeaders=["Key","Summary","SP"]`, `ColumnWidths=[13,78,9]`, `CenterColumns=[0,2]`; pass each config to its respective `word.NewTable` call; change all `t.AddHeaderRow([]string{...})` calls to `t.AddHeaderRow()`
- [x] T009 [P] [US1] Update `internal/server/handler.go`: build a `word.TableConfig` with `ColumnHeaders = []string{"types", "id", "name", "epic", "SP"}` and default widths; pass to `word.NewTable`; change `table.AddHeaderRow(headers)` to `table.AddHeaderRow()`; remove the `headers := []string{...}` local variable

**Checkpoint**: After T002–T009, all `make build` targets should compile. `make run-month` output matches reference table appearance.

---

## Phase 4: User Story 2 — Page Break Before Each Section (Priority: P2)

**Goal**: Every section after the first in a generated document starts on a new page.

**Independent Test**: Run `make run-month` with ≥2 teams configured. Open the output `.docx` and verify that the second (and subsequent) section headings appear at the top of a new page. The first section has no leading blank page.

### Implementation for User Story 2

- [x] T010 [US2] Add unexported `sectionCount int` field to the `Doc` struct in `internal/word/doc.go`
- [x] T011 [US2] Update `AddHeading()` in `internal/word/doc.go`: replace the current `d.WordDocument.AddParagraph().AddRun().AddBreak()` line-break call with a conditional page break: if `d.sectionCount > 0`, call `d.WordDocument.AddParagraph().AddRun().AddPageBreak()`; then always increment `d.sectionCount`; keep the rest of the heading paragraph creation unchanged

**Checkpoint**: Running `make run-month` with multiple teams produces a document where each team's section starts on a new page.

---

## Phase 5: User Story 3 — Configurable Column Header Text (Priority: P3)

**Goal**: Column header labels are controlled by `TableConfig.ColumnHeaders`; no header strings are hardcoded in table-rendering functions.

**Independent Test**: Change `ColumnHeaders[0]` from `"Type"` to `"Тип"` in a caller and re-run; the output `.docx` shows `"Тип"` in the first header cell.

US3 is fully delivered by T001 (adding `ColumnHeaders` to `TableConfig`) and T003 (reading it in `AddHeaderRow`) and T006–T009 (callers set custom labels). No additional tasks are needed for this story.

---

## Phase 6: User Story 4 — Unified Section Header Style (Priority: P4)

**Goal**: Section headings (Heading 1) in generated documents match the reference document: Trebuchet MS, 16pt, small caps, letter spacing 5, paragraph spacing before ~15pt / after ~2pt.

**Independent Test**: Open a generated `.docx`, right-click a section heading, inspect font → Trebuchet MS 16pt with small caps; inspect paragraph spacing → before ≈ 300 twips, after ≈ 40 twips.

### Implementation for User Story 4

- [x] T012 [US4] Update `AddHeading()` in `internal/word/doc.go`: for heading level 1 only, after adding the run and text, apply: `run.Properties().SetFontFamily("Trebuchet MS")`, `run.Properties().SetSize(16)`, `run.Properties().SetSmallCaps(true)`, `run.Properties().SetCharacterSpacing(5 * measurement.Twips)`; and on the paragraph: `para.Properties().SetSpacing(300*measurement.Twips, 40*measurement.Twips)` (before, after)

**Checkpoint**: All four user stories are independently functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [x] T013 Run `make build` at project root and confirm zero compilation errors and zero lint errors (`make lint`)
- [x] T014 [P] Update `.specify/memory/constitution.md` Principle IV: replace `#365F91` with `#0070C0` for header background; replace "0.2 cm on all sides" with "top and bottom only (~1mm / 57 dxa), left and right use Word default"; replace "single black borders" with "single auto-color borders at 1pt"

---

## Dependencies & Execution Order

### Phase Dependencies

- **Foundational (Phase 2)**: No dependencies — start immediately
- **US1 (Phase 3)**: Depends on T001 — BLOCKS until T001 complete
- **US2 (Phase 4)**: Depends on T001 (struct exists); can run in parallel with US1 (different file: `doc.go` vs `table.go`)
- **US4 (Phase 6)**: Depends on T011 — sequential within `doc.go`
- **Polish (Phase 7)**: Depends on all prior phases

### User Story Dependencies

- **US1 (T002–T009)**: Depends on T001 (updated TableConfig fields)
- **US2 (T010–T011)**: Depends on T001; can start in parallel with US1
- **US3**: No separate tasks — delivered by T001 + T003 + T006–T009
- **US4 (T012)**: Depends on T011 (same function, sequential edit)

### Within US1

- T002 before T003 (T003 uses config stored by T002)
- T002 before T004 (T004 uses config stored by T002)
- T003 and T004 can be done in parallel (different methods)
- T005 after T003 and T004 (removes code used by both)
- T006–T009 after T002–T004; T006–T009 are parallel to each other (different files)

---

## Parallel Opportunities

```
After T001 completes, run in parallel:
  Batch A (table.go changes):
    T002 → T003 [parallel with T004] → T005 → T006
                                              → T007 [parallel]
                                              → T008 [parallel]
                                              → T009 [parallel]
  Batch B (doc.go changes):
    T010 → T011 → T012

After all above: T013 [parallel with T014]
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete T001 (Foundational)
2. Complete T002–T009 (US1 — table design)
3. **STOP and VALIDATE**: Run `make run-month`, open output, compare with reference document
4. Ship if acceptable

### Incremental Delivery

1. T001 → Foundation ready
2. T002–T009 → US1 done (reference table design + configurable headers = US1 + US3)
3. T010–T011 → US2 done (page breaks)
4. T012 → US4 done (heading style)
5. T013–T014 → Polish + constitution update

---

## Notes

- [P] tasks operate on different files and have no inter-dependencies
- Raw XML access via `.X()` is required for table indent, table-level cell margins, and header row repeat — no high-level gooxml API exists for these
- T008 (sprint-label-report) is the most complex caller update — three distinct table configs
- Heading style (T012) applies only to level 1; Heading 2 uses default gooxml styling
- The `sectionCount` field is unexported — no callers need to inspect it
