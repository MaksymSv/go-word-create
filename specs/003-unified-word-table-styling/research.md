# Research: Unified Word Table Styling

**Feature**: 003-unified-word-table-styling  
**Date**: 2026-06-05

---

## 1. Reference Document Analysis

**Decision**: Use values extracted directly from `EPAM Delivery Report - 2026-05.docx` as the canonical source.

**Rationale**: The DOCX XML was parsed programmatically (Python `zipfile` + `xml.etree.ElementTree`) to obtain exact numeric values for every visual property. No estimation needed.

**Extracted values**:

| Property | OOXML value | Human-readable | Current code |
|----------|-------------|----------------|--------------|
| Table style | `TableGrid1` | EPAM custom table style | none |
| Table width | `w="5650" type="pct"` | ~113% of text area | 100% |
| Table indent | `w="-584" type="dxa"` | −584 twips (extends left beyond margin) | 0 |
| Table layout | `fixed` | Fixed column widths | auto |
| Border style | `single sz=8 color=auto` | 1pt, auto color | single, 1pt, black |
| Cell margin top | `w=57 type=dxa` | ~1mm | 0.2cm (113 dxa) |
| Cell margin bottom | `w=57 type=dxa` | ~1mm | 0.2cm (113 dxa) |
| Cell margin left/right | *(not set — default)* | Word default (~108 dxa) | 0.2cm (113 dxa) |
| Header fill | `#0070C0` | EPAM blue | `#365F91` |
| Header text color | `#FFFFFF` | White | White ✓ |
| Header font size | `sz=16` | 8pt | 10pt (sz=20 via SetSize(10)) |
| Header bold | *(none — only `bCs`)* | Not bold (bold complex script only) | `SetBold(true)` |
| Header alignment | center | Center ✓ | Center ✓ |
| Header row height | `val=196` | 196 twips | not set |
| Header repeat | `tblHeader` | Repeats on page break | missing |
| Data font | Aptos Narrow | Aptos Narrow ✓ | Aptos Narrow ✓ |
| Data font size | `sz=16` | 8pt | 8pt (sz=16 via SetSize(8)) ✓ |
| Data row height | `val=300` | 300 twips | not set |

**Column width proportions** (first table, grid cols: 584, 1408, 4378, 3612, 572 twips):

| Column | Name | % of table |
|--------|------|-----------|
| 0 | Type | 5.5% |
| 1 | ID | 13.3% |
| 2 | Task/Description | 41.5% |
| 3 | Epic | 34.2% |
| 4 | SP | 5.4% |

**Section structure** (document order):
- Title paragraph (not a Heading1)
- Heading2: month/year
- **[No page break before first section]**
- Heading1: section name → Table
- Normal paragraph (empty, line break) → **Page break paragraph** → Heading1: next section → Table
- ...repeats

The page break is a `<w:p><w:r><w:br w:type="page"/></w:r></w:p>` paragraph inserted after each table, before the next Heading1.

---

## 2. Heading 1 Style in Reference Document

**Decision**: Apply Heading 1 style properties explicitly on each heading paragraph and run.

**Rationale**: `gooxml` generates documents without pre-built styles unless a template is used. Relying on `SetHeadingLevel(0)` only sets `pStyle="Heading1"`, which won't render correctly without the style definition. Applying properties directly is portable.

**Reference Heading 1 properties** (from `word/styles.xml`):

| Property | Value | API |
|----------|-------|-----|
| Font (major theme = Trebuchet MS) | Trebuchet MS | `run.Properties().SetFontFamily("Trebuchet MS")` |
| Size | sz=32 → 16pt | `run.Properties().SetSize(16)` |
| Small caps | yes | `run.Properties().SetSmallCaps(true)` |
| Character spacing | val=5 | `run.Properties().SetCharacterSpacing(5)` — value in twips |
| Spacing before | 300 twips | `para.Properties().SetSpacing(300*measurement.Twips, 40*measurement.Twips)` |
| Spacing after | 40 twips | (same call) |
| Paragraph spacing after=0 for the run-level override | yes | `para.Properties().SetSpacingAfter(0)` per spec |

**Note**: `SetCharacterSpacing` in gooxml takes `measurement.Distance`; the OOXML `w:spacing val=5` is in twentieths of a point (= 5 half-points / 2.5 pt). Passing `5 * measurement.Twips` would be wrong. The raw value `5` in OOXML `w:rPr/w:spacing` is measured in twentieths of a point. One `measurement.Twips` = 635 EMU. We need to check what unit `SetCharacterSpacing` expects.

```go
// From gooxml source:
func (r RunProperties) SetCharacterSpacing(size measurement.Distance) {
    // ...stores value as int64(size / measurement.Twips) into CT_Spacing.ValAttr
}
```

So to get OOXML `val=5` (5 twips), pass `5 * measurement.Twips`.

---

## 3. gooxml API Gaps — Raw XML Access Needed

**Decision**: Access raw XML structs (`.X()`) for features not exposed via high-level API.

**Rationale**: `TableProperties` and `RowProperties` do not expose table indent, table-level cell margins, or repeat-header. The library provides `.X()` accessors to the underlying `*wml.CT_TblPr` and `*wml.CT_TrPr`.

**Required raw-XML operations**:

| Feature | Go code |
|---------|---------|
| Table-level cell margin (top only) | `tblPr.TblCellMar.Top = ...` (dxa type) |
| Table indent | `tblPr.TblInd = wml.NewCT_TblWidth()` with `TypeAttr=ST_TblWidthDxa, val=-584` |
| Repeat header row | `row.X().TrPr.TblHeader = []*wml.CT_OnOff{wml.NewCT_OnOff()}` |

**Negative DXA for indent**: `gooxml.Int64(-584)` — the field is `int64`, negative is valid.

---

## 4. Configurable Column Headers and Widths

**Decision**: Add `ColumnHeaders []string` and `ColumnWidths []float64` to `TableConfig`. `AddHeaderRow` and `AddDataRow` no longer take arguments; they read from the config embedded in `Table`.

**Rationale**: This makes the entire table visual spec declarative in one struct. Callers construct a `TableConfig` and pass it once to `NewTable`; no header strings scattered across `cmd/` files.

**API change**:
- `NewTable(doc, config)` replaces `NewTable(doc)` and `WithConfig(doc, config)`
- `AddHeaderRow()` (no args) replaces `AddHeaderRow(headers []string)`
- `AddDataRow(data []string)` unchanged — data rows remain caller-supplied

**Backward compat**: `DefaultConfig()` returns a ready-to-use config with the canonical 5-column headers and widths. Callers that only need defaults can call `word.NewTable(&doc.WordDocument, word.DefaultConfig())`.

---

## 5. Page Breaks Between Sections

**Decision**: Track section count in `Doc`; auto-insert a page break paragraph before every heading after the first.

**Rationale**: The reference document places a `<w:p><w:r><w:br w:type="page"/></w:r></w:p>` paragraph between sections. Tracking count in `Doc` avoids requiring callers to manage state.

**Implementation**:
```go
type Doc struct {
    WordDocument   document.Document
    sectionCount   int  // unexported
}

func (d *Doc) AddHeading(level int, text string) {
    if d.sectionCount > 0 {
        d.WordDocument.AddParagraph().AddRun().AddPageBreak()
    }
    d.sectionCount++
    // ... add heading paragraph with style
}
```

---

## 6. Column Width Implementation

**Decision**: Set each cell's width percentage via `cell.Properties().SetWidthPercent(pct)` using the proportions from the reference document.

**Rationale**: `gooxml` exposes `SetWidthPercent` on `CellProperties`. Setting column widths per-cell (header row drives the layout for fixed tables) is the standard approach when `tblLayout=fixed`.

`SetWidthPercent(pct)` sets `w = pct * 50` in OOXML terms (fiftieths of a percent). So `SetWidthPercent(5.5)` → `w=275 type=pct`, which matches the reference (`w=277` ≈ 5.54%).

**Column widths array in `DefaultConfig()`**:
```go
ColumnWidths: []float64{5.5, 13.3, 41.5, 34.2, 5.4},
```

**Column center-alignment** (reference: Type/ID/SP centered, Task/Epic left):
```go
// Default: center cols 0, 1, 4; left-align cols 2, 3
CenterColumns: []int{0, 1, 4},
```

---

## 7. Header Row Height

**Decision**: Set header row height to `196 * measurement.Twips` using `RowProperties.SetHeight`.

**Decision**: Set data row height to `300 * measurement.Twips` (minimum height, not exact, so content can expand).

**Rationale**: Matches the reference document. `SetHeight` with `ST_HeightRuleAtLeast` allows rows with many lines to expand.

---

## Alternatives Considered

| Area | Rejected Alternative | Reason |
|------|---------------------|--------|
| Column widths | `tblGrid` GridCol elements | Not exposed by gooxml high-level API; cell `SetWidthPercent` achieves same result |
| Heading style | Use `SetHeadingLevel` only | Requires pre-defined styles in document; gooxml generates blank doc without styles |
| Page break | `pageBreakBefore` on paragraph properties | Not supported in gooxml `ParagraphProperties`; explicit break paragraph is simpler |
| Table indent | Wrapper method on `TableProperties` | Not in current API; `.X()` access is explicit and readable |
| Config headers | Keep `AddHeaderRow(headers []string)` | Breaks the "one place to change" goal; headers would still be scattered in `cmd/` |
