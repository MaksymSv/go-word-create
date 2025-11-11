package word

import (
	"github.com/carmel/gooxml/color"
	"github.com/carmel/gooxml/document"
	"github.com/carmel/gooxml/measurement"
	"github.com/carmel/gooxml/schema/soo/wml"
)

// Table represents a Word document table wrapper
type Table struct {
	table document.Table
}

// NewTable creates a new table in the document with default settings
func NewTable(doc *document.Document) *Table {
	table := doc.AddTable()
	table.Properties().SetWidthPercent(100)

	// Set table borders
	borders := table.Properties().Borders()
	borders.SetAll(wml.ST_BorderSingle, color.Black, measurement.Point)

	return &Table{table: table}
}

// AddHeaderRow creates a header row with the specified cell values
func (t *Table) AddHeaderRow(headers []string) {
	headerRow := t.table.AddRow()
	for _, h := range headers {
		cell := headerRow.AddCell()
		// Set cell background color to blue
		cell.Properties().SetShading(wml.ST_ShdSolid, color.RGB(0x36, 0x5F, 0x91), color.Auto)
		setCellMargins(cell)
		para := cell.AddParagraph()
		// Center align
		para.Properties().SetAlignment(wml.ST_JcCenter)
		run := para.AddRun()
		run.AddText(h)
		// Set header font and size
		run.Properties().SetBold(true)
		run.Properties().SetColor(color.White) // White text
		// Attempt to set font family and size for table text
		run.Properties().SetSize(10)
		run.Properties().SetFontFamily("Aptos Narrow")
	}
}

// AddDataRow creates a data row with the specified cell values
func (t *Table) AddDataRow(data []string) {
	dataRow := t.table.AddRow()
	for i, val := range data {
		cell := dataRow.AddCell()
		setCellMargins(cell)
		para := cell.AddParagraph()
		if i != 2 && i != 3 {
			para.Properties().SetAlignment(wml.ST_JcCenter)
		}
		run := para.AddRun()
		run.AddText(val)
		// Set table cell font and size
		run.Properties().SetSize(8)
		run.Properties().SetFontFamily("Aptos Narrow")
	}
}

// setCellMargins sets the margins for a table cell
func setCellMargins(cell document.Cell) {
	cell.Properties().Margins().SetTop(measurement.Centimeter * 0.2)
	cell.Properties().Margins().SetBottom(measurement.Centimeter * 0.2)
	cell.Properties().Margins().SetLeft(measurement.Centimeter * 0.2)
	cell.Properties().Margins().SetRight(measurement.Centimeter * 0.2)
}
