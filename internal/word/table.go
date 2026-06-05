package word

import (
	"github.com/carmel/gooxml"
	"github.com/carmel/gooxml/color"
	"github.com/carmel/gooxml/document"
	"github.com/carmel/gooxml/measurement"
	"github.com/carmel/gooxml/schema/soo/wml"
)

// Table represents a Word document table wrapper
type Table struct {
	table  document.Table
	config TableConfig
}

// NewTable creates a new table with the given config applied
func NewTable(doc *document.Document, config TableConfig) *Table {
	tbl := doc.AddTable()

	tbl.Properties().SetWidthPercent(113)
	tbl.Properties().SetLayout(wml.ST_TblLayoutTypeFixed)

	borders := tbl.Properties().Borders()
	borders.SetAll(wml.ST_BorderSingle, color.Auto, 0.5*measurement.Point)

	// Table-level cell margins: top and bottom only; left/right use Word default
	tblPr := tbl.Properties().X()
	tblPr.TblCellMar = wml.NewCT_TblCellMar()
	tblPr.TblCellMar.Top = setTblWidth(config.CellTopBottomMargin)
	tblPr.TblCellMar.Bottom = setTblWidth(config.CellTopBottomMargin)
	tblPr.TblCellMar.Left = setTblWidth(config.CellLeftRightMargin)
	tblPr.TblCellMar.Right = setTblWidth(config.CellLeftRightMargin)

	// Negative indent so the wider table aligns with the page margin
	tblPr.TblInd = wml.NewCT_TblWidth()
	tblPr.TblInd.TypeAttr = wml.ST_TblWidthDxa
	tblPr.TblInd.WAttr = &wml.ST_MeasurementOrPercent{}
	tblPr.TblInd.WAttr.ST_DecimalNumberOrPercent = &wml.ST_DecimalNumberOrPercent{}
	tblPr.TblInd.WAttr.ST_DecimalNumberOrPercent.ST_UnqualifiedPercentage = gooxml.Int64(-584)

	return &Table{table: tbl, config: config}
}

// AddHeaderRow creates a header row using labels from the config
func (t *Table) AddHeaderRow() {
	headerRow := t.table.AddRow()
	headerRow.Properties().SetHeight(196*measurement.Twips, wml.ST_HeightRuleAtLeast)

	// Mark as repeating header row
	if headerRow.X().TrPr == nil {
		headerRow.X().TrPr = wml.NewCT_TrPr()
	}
	headerRow.X().TrPr.TblHeader = []*wml.CT_OnOff{wml.NewCT_OnOff()}

	for i, h := range t.config.ColumnHeaders {
		cell := headerRow.AddCell()
		if i < len(t.config.ColumnWidths) {
			cell.Properties().SetWidthPercent(t.config.ColumnWidths[i])
		}
		cell.Properties().SetShading(wml.ST_ShdSolid, t.config.HeaderBackgroundColor, color.Auto)
		para := cell.AddParagraph()
		para.Properties().SetAlignment(wml.ST_JcCenter)
		para.Properties().SetSpacing(0, 0)
		run := para.AddRun()
		run.AddText(h)
		run.Properties().SetColor(t.config.HeaderTextColor)
		run.Properties().SetSize(8)
		run.Properties().SetFontFamily("Aptos Narrow")
	}
}

// AddDataRow creates a data row with the specified cell values
func (t *Table) AddDataRow(data []string) {
	dataRow := t.table.AddRow()
	dataRow.Properties().SetHeight(300*measurement.Twips, wml.ST_HeightRuleAtLeast)

	for i, val := range data {
		cell := dataRow.AddCell()
		if i < len(t.config.ColumnWidths) {
			cell.Properties().SetWidthPercent(t.config.ColumnWidths[i])
		}
		para := cell.AddParagraph()
		if isCentered(i, t.config.CenterColumns) {
			para.Properties().SetAlignment(wml.ST_JcCenter)
		}
		run := para.AddRun()
		run.AddText(val)
		run.Properties().SetSize(8)
		run.Properties().SetFontFamily("Aptos Narrow")
	}
}

// setTblWidth builds a dxa-typed CT_TblWidth from a measurement.Distance
func setTblWidth(d measurement.Distance) *wml.CT_TblWidth {
	w := wml.NewCT_TblWidth()
	w.TypeAttr = wml.ST_TblWidthDxa
	w.WAttr = &wml.ST_MeasurementOrPercent{}
	w.WAttr.ST_DecimalNumberOrPercent = &wml.ST_DecimalNumberOrPercent{}
	w.WAttr.ST_DecimalNumberOrPercent.ST_UnqualifiedPercentage = gooxml.Int64(int64(d / measurement.Dxa))
	return w
}

// isCentered reports whether column index i is in the center list
func isCentered(i int, centers []int) bool {
	for _, c := range centers {
		if c == i {
			return true
		}
	}
	return false
}
