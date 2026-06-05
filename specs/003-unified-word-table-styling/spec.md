# Feature Specification: Unified Word Table Styling

**Feature Branch**: `003-unified-word-table-styling`

**Created**: 2026-06-05

**Status**: Draft

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Match Reference Document Table Design (Priority: P1)

As a report generator, when I run `make run-month`, the output Word document tables should look identical to the reference document `EPAM Delivery Report - 2026-05.docx`: blue header row (#0070C0), white header text (8pt), Aptos Narrow data font (8pt), correct cell margins (top/bottom only), single-line borders at 1pt, and consistent column proportions.

**Why this priority**: Visual fidelity to the reference document is the primary stated goal. All other improvements depend on a correct baseline table design.

**Independent Test**: Run `make run-month MONTH=2026.05` and open the output `.docx` side-by-side with `EPAM Delivery Report - 2026-05.docx`. Compare header background color, text color, font sizes, cell margins, and border appearance. Document is acceptable if a non-technical reviewer cannot distinguish the table styles.

**Acceptance Scenarios**:

1. **Given** a generated Word document, **When** the table header row is viewed, **Then** it has a solid blue background (#0070C0), white text, 8pt font size, and center-aligned text.
2. **Given** a generated Word document, **When** data rows are viewed, **Then** cells use Aptos Narrow font at 8pt with top and bottom margins of ~1mm and no left/right margin override.
3. **Given** a generated Word document, **When** table borders are viewed, **Then** all external and internal borders are single-line at 1pt (no thick, double, or absent borders).
4. **Given** a generated Word document, **When** column widths are compared to the reference, **Then** column proportions match: Type ~5.5%, ID ~13.3%, Description/Task ~41.5%, Epic ~34.2%, SP ~5.4%.

---

### User Story 2 - Page Break Before Each Section (Priority: P2)

As a report generator, when the output document contains multiple sections (each with a heading and a table), each section after the first must begin on a new page so that sections do not run together.

**Why this priority**: Without page breaks, multi-section reports are visually cluttered. This is observable in the reference document where every Heading1 section starts on its own page.

**Independent Test**: Run `make run-month` with a configuration that produces at least two teams. Verify in the output document that a page break appears between each section (heading + table pair), and the first section starts without a leading page break.

**Acceptance Scenarios**:

1. **Given** a document with multiple sections, **When** any section after the first is reached, **Then** a page break precedes the section heading so it starts at the top of a new page.
2. **Given** a document with exactly one section, **When** the document is opened, **Then** no extra page breaks are present (no blank leading page).
3. **Given** a section heading followed by a table, **When** the page break is inserted, **Then** the heading and its table remain together on the same page (the break is before the heading, not between heading and table).

---

### User Story 3 - Configurable Column Header Text (Priority: P3)

As a developer configuring the report, I want to specify the text of each column header through configuration rather than having it hardcoded, so that different commands or report types can display different column labels without code changes.

**Why this priority**: The column headers "Type", "ID", "Description", "Epic", "SP" are currently hardcoded in `main.go`. Making them configurable enables reuse of the same table-generation logic across different commands with different terminology.

**Independent Test**: Change the column header configuration from "Type" to "Тип" and re-run the report. Verify the output document shows "Тип" in the first header cell. Revert and confirm "Type" reappears.

**Acceptance Scenarios**:

1. **Given** a table configuration with custom header labels, **When** a Word document table is generated, **Then** each header cell displays the configured label text, not a hardcoded default.
2. **Given** a table configuration with five header labels, **When** `AddHeaderRow` is called, **Then** exactly five header cells are rendered with the specified labels.
3. **Given** the `get-month-issues-from-jira` command, **When** headers are defined in the table config (not inside `addTableToDocument`), **Then** the header strings can be changed in one place without modifying table-rendering logic.

---

### User Story 4 - Unified Section Header Style (Priority: P4)

As a report generator, section headings (Heading 1) in the output document should match the reference document's Heading 1 style: Trebuchet MS font, 16pt, small caps, letter spacing of 5, with spacing before = 15pt and after = 2pt.

**Why this priority**: Consistent heading style makes all reports look professionally produced and matching the EPAM brand template.

**Independent Test**: Open the generated document, right-click a section heading, and check "Paragraph" and "Font" settings. Compare font name, size, small caps, and spacing to the reference document.

**Acceptance Scenarios**:

1. **Given** a generated document with section headings, **When** a heading is inspected, **Then** it uses Trebuchet MS, 16pt, small caps style.
2. **Given** a section heading in the generated document, **When** paragraph spacing is checked, **Then** spacing before is ~15pt (300 twips) and after is ~2pt (40 twips).
3. **Given** any Word document generated by this application, **When** Heading 1 is applied, **Then** the style matches across all commands (`run-month`, `run-sprint-label-report`, etc.).

---

### Edge Cases

- What happens when a section has zero issues — does the page break and empty table render without errors?
- What happens when a column header string is empty — does the header cell render blank or fall back to a default?
- How does the table handle very long task names that exceed the column width — does it wrap within the cell?
- What happens when only one team is configured — is no page break added before the first (and only) section?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The table header row MUST use background color #0070C0 (EPAM blue) with white text (#FFFFFF).
- **FR-002**: All table text (headers and data) MUST use font size 8pt (16 half-points in OOXML).
- **FR-003**: Table data cells MUST use the Aptos Narrow font family.
- **FR-004**: Table cell margins MUST be set only for top and bottom (57 dxa ≈ 1mm each); left and right cell margins MUST use the document default (no override).
- **FR-005**: All table borders (outer and inner, horizontal and vertical) MUST be single-line style at 1pt width with automatic color.
- **FR-006**: Column widths MUST be set proportionally: Type 5.5%, ID 13.3%, Description/Task 41.5%, Epic 34.2%, SP 5.4% of total table width.
- **FR-007**: The header row MUST be marked as a repeating header so it appears on every page when a table spans multiple pages.
- **FR-008**: A page break MUST be inserted before each section heading except the very first section in the document.
- **FR-009**: The page break MUST be placed before the heading paragraph so that the heading and its table remain together on the new page.
- **FR-010**: Column header labels MUST be configurable via a `TableConfig` struct field (a `[]string` slice), so callers supply the labels rather than hardcoding them inside the table-rendering functions.
- **FR-011**: Section headings (Heading 1) MUST be styled with Trebuchet MS font, 16pt, small caps, letter spacing 5, paragraph spacing before 300 twips (~15pt), and paragraph spacing after 40 twips (~2pt).
- **FR-012**: The unified table design MUST be applied consistently across all Word-generating commands in the project (`get-month-issues-from-jira` and any future commands using `word.NewTable`).
- **FR-013**: The `TableConfig` struct MUST include a `ColumnHeaders []string` field; `AddHeaderRow` MUST use this field instead of accepting a separate parameter.

### Key Entities

- **TableConfig**: Configuration struct holding all visual parameters for a table — header colors, font sizes, font family, cell margins, column widths, column header labels, and table width/indent settings.
- **Table**: The Word table wrapper (`internal/word/table.go`) that applies the `TableConfig` when creating header and data rows.
- **Doc**: The Word document wrapper (`internal/word/doc.go`) responsible for adding headings with the correct style and inserting page breaks between sections.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A non-technical reviewer comparing a generated report and the reference document `EPAM Delivery Report - 2026-05.docx` rates the table visual similarity as "identical or indistinguishable" for header color, text color, font size, borders, and cell margins.
- **SC-002**: All existing `make run-month` and `make run-sprint-label-report` commands produce valid, openable `.docx` files after this change (zero regression in document generation).
- **SC-003**: Changing a column header label requires modifying only the `TableConfig` initialization — no changes to table-rendering functions (`AddHeaderRow`, `AddDataRow`).
- **SC-004**: In a multi-section output document, every section after the first begins on a new page (100% of sections have a preceding page break).
- **SC-005**: Section headings in generated documents visually match Heading 1 from the reference document with respect to font, size, and capitalization style.

## Assumptions

- The `gooxml` library (github.com/carmel/gooxml) supports setting individual column widths, table indentation, repeating header rows, and explicit page breaks — no library replacement is needed.
- The reference document `EPAM Delivery Report - 2026-05.docx` located in the project root is the authoritative design source.
- The "Aptos Narrow" font is available on the target machines where the generated documents will be opened; if not installed, Word will substitute a fallback font.
- The Trebuchet MS font (EPAM theme minor font) is available for section headings.
- Column proportions are fixed for the 5-column layout (Type / ID / Task / Epic / SP); different column counts for other report types are out of scope for this feature.
- The sprint label report (`get-sprint-label-report`) uses a different column schema but should share the same visual styling rules (colors, fonts, borders, margins) through the shared `TableConfig`.
- No alternating row shading is required — the reference document data rows have no background fill.
- The first section in a document does NOT get a leading page break.
