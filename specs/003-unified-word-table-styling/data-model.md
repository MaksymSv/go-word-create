# Data Model: Unified Word Table Styling

**Feature**: 003-unified-word-table-styling  
**Date**: 2026-06-05

---

## TableConfig (updated)

File: `internal/word/config.go`

```
TableConfig
├── HeaderBackgroundColor  color.Color       // header row fill (#0070C0 default)
├── HeaderTextColor        color.Color       // header text color (White default)
├── CellTopBottomMargin    measurement.Distance // top and bottom cell margin (57 dxa default ≈ 1mm)
├── ColumnHeaders          []string          // header cell labels (configurable per caller)
├── ColumnWidths           []float64         // column widths as % of table (must sum ≤ 100)
└── CenterColumns          []int             // 0-indexed columns that get center alignment
```

**Validation rule**: `len(ColumnHeaders) == len(ColumnWidths)` (enforced at runtime; mismatch silently uses min length).

**DefaultConfig()** returns:
```
HeaderBackgroundColor  = RGB(0x00, 0x70, 0xC0)
HeaderTextColor        = color.White
CellTopBottomMargin    = 57 * measurement.Dxa
ColumnHeaders          = ["Type", "ID", "Description", "Epic", "SP"]
ColumnWidths           = [5.5, 13.3, 41.5, 34.2, 5.4]
CenterColumns          = [0, 1, 4]
```

**Removed fields**:
- `CellMargin float64` — replaced by `CellTopBottomMargin measurement.Distance` (left/right now use Word default)
- `Width int` — table is now always ~113% (5650 pct units); not caller-configurable

---

## Table (updated)

File: `internal/word/table.go`

```
Table
├── table    document.Table   // gooxml table handle
└── config   TableConfig      // held for AddHeaderRow / AddDataRow
```

**Constructor**: `NewTable(doc *document.Document, config TableConfig) *Table`
- Replaces both `NewTable(doc)` and `WithConfig(doc, config)`
- Applies to the table: width (~113%), layout fixed, borders (single 1pt auto), table-level cell margins (top+bottom only), table indent (−584 dxa)

**Methods**:

| Method | Signature change | Notes |
|--------|-----------------|-------|
| `AddHeaderRow` | `()` — no args | Reads `config.ColumnHeaders`, `config.ColumnWidths`, `config.CenterColumns` |
| `AddDataRow` | `(data []string)` — unchanged | Reads `config.ColumnWidths` for cell width, `config.CenterColumns` for alignment |

**`AddHeaderRow()` behavior**:
1. Add row, mark as tblHeader (repeating) via `row.X().TrPr.TblHeader`
2. Set row height 196 twips (AtLeast)
3. For each header label in `config.ColumnHeaders`:
   - Add cell, set shading to `config.HeaderBackgroundColor`
   - Set cell width via `config.ColumnWidths[i]`
   - Add paragraph, center-align
   - Add run: text = label, color = `config.HeaderTextColor`, size = 8pt, font = Aptos Narrow (not bold — matches reference)

**`AddDataRow(data []string)` behavior**:
1. Add row, set height 300 twips (AtLeast)
2. For each value in `data`:
   - Add cell, set width via `config.ColumnWidths[i]` (if index in range)
   - Add paragraph, center-align if `i` is in `config.CenterColumns`
   - Add run: text = value, size = 8pt, font = Aptos Narrow

**Removed**: `setCellMargins(cell)` helper — replaced by table-level margin in `NewTable`

---

## Doc (updated)

File: `internal/word/doc.go`

```
Doc
├── WordDocument   document.Document   // gooxml document
└── sectionCount   int                 // tracks how many sections added (unexported)
```

**`AddHeading(level int, text string)` behavior (updated)**:
1. If `sectionCount > 0`: insert page break paragraph (`AddParagraph().AddRun().AddPageBreak()`)
2. Increment `sectionCount`
3. Add paragraph, call `SetHeadingLevel(level - 1)` (gooxml uses 0-based index)
4. Set paragraph spacing: before = 300 twips, after = 40 twips
5. Add run with text:
   - Font: Trebuchet MS
   - Size: 16pt
   - SmallCaps: true
   - Character spacing: 5 twips

**Note on heading level**: `SetHeadingLevel` in gooxml is 0-based (`0` → Heading1, `1` → Heading2). Current callers pass `1` for Heading1 and `2` for Heading2. The updated implementation needs to subtract 1, or keep the existing convention — verify against current callers.

Looking at current usage: `heading1.Properties().SetHeadingLevel(headingLevel)` where callers pass `1`. So the existing code sets `pStyle` to `Heading1` via level=1. gooxml's `SetHeadingLevel` likely uses the passed int directly to construct `"Heading" + strconv.Itoa(idx)`. This convention must be preserved.

**Heading style applies to level 1 only**: Levels 2+ get `SetHeadingLevel` only (no custom styling, to avoid over-engineering Heading2).

---

## Caller Changes (cmd/ packages)

### `cmd/get-month-issues-from-jira/main.go`

`addTableToDocument`:
- Before: `word.NewTable(&doc.WordDocument)` then `closedIssuesTable.AddHeaderRow(headers)`
- After: `word.NewTable(&doc.WordDocument, word.DefaultConfig())` then `closedIssuesTable.AddHeaderRow()`
- Remove `headers := []string{...}` local variable

### `cmd/get-sprint-issues-from-jira/main.go`

- Before: `table := word.NewTable(&doc.WordDocument)` then `table.AddHeaderRow(headers)`
- After: `cfg := word.DefaultConfig(); cfg.ColumnHeaders = []string{"Type", "Key", "Summary", "Epic", "Story Points"}; table := word.NewTable(&doc.WordDocument, cfg)` then `table.AddHeaderRow()`
- The sprint command uses different header labels ("Key" instead of "ID", "Summary" instead of "Description", "Story Points" instead of "SP")

### `cmd/get-sprint-label-report/main.go`

Multiple tables with different column schemas:
- Short format: `[]string{"Label", "Count", "Count,%", "Total SP", "Total SP,%"}`, 5 columns
- Full format: `[]string{"Label", "Count", "Count,%", "Total SP", "Total SP,%", "Key", "Summary", "SP"}`, 8 columns
- Unlabeled sub-table: `[]string{"Key", "Summary", "SP"}`, 3 columns

Each creates a `TableConfig` with custom `ColumnHeaders` and `ColumnWidths`.

**Column widths for sprint label report** (5-col short format):
```
[20.0, 10.0, 10.0, 10.0, 10.0]  // Label wide, others even
```

**Column widths for sprint label report full format** (8-col):
```
[15.0, 7.0, 7.0, 7.0, 7.0, 10.0, 35.0, 7.0]
```

**Column widths for unlabeled sub-table** (3-col):
```
[15.0, 75.0, 10.0]
```

**Center columns**: for label report, center all numeric columns.

### `internal/server/handler.go`

- Before: `word.NewTable(&doc.WordDocument)` then `table.AddHeaderRow(headers)`
- After: `cfg := word.DefaultConfig(); cfg.ColumnHeaders = []string{"types", "id", "name", "epic", "SP"}; table := word.NewTable(&doc.WordDocument, cfg)` then `table.AddHeaderRow()`

---

## State Transitions

`Doc.sectionCount`:
- Starts at `0`
- Incremented by `AddHeading` after inserting (or skipping) the page break
- No reset mechanism — document generation is one-pass

---

## Invariants

- `len(data)` in `AddDataRow` may differ from `len(config.ColumnHeaders)` — extra data values are silently ignored; missing values produce empty cells.
- `config.ColumnWidths` may not sum to exactly 100; table layout is fixed so Word uses the declared widths.
- Negative table indent (−584 dxa) is set unconditionally; this matches the reference document's style of extending the table slightly beyond the text margin.
