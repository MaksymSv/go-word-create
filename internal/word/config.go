package word

import (
	"github.com/carmel/gooxml/color"
	"github.com/carmel/gooxml/measurement"
)

// TableConfig holds configuration for table appearance
type TableConfig struct {
	// HeaderBackgroundColor is the background color for header cells
	HeaderBackgroundColor color.Color
	// HeaderTextColor is the text color for header cells
	HeaderTextColor color.Color
	// CellTopBottomMargin is the top and bottom cell margin; left/right use Word default
	CellTopBottomMargin measurement.Distance
	// CellLeftRightMargin is the left and right cell margin; top/bottom use CellTopBottomMargin
	CellLeftRightMargin measurement.Distance
	// ColumnHeaders contains the label for each column header cell
	ColumnHeaders []string
	// ColumnWidths contains the width of each column as a percentage of table width
	ColumnWidths []float64
	// CenterColumns lists 0-indexed column positions that should be center-aligned
	CenterColumns []int
}

// DefaultConfig returns the default table configuration matching the EPAM reference document
func DefaultConfig() TableConfig {
	return TableConfig{
		HeaderBackgroundColor: color.RGB(0x00, 0x70, 0xC0),
		HeaderTextColor:       color.White,
		CellTopBottomMargin:   0,
		CellLeftRightMargin:   1.5 * measurement.Millimeter,
		ColumnHeaders:         []string{"Type", "ID", "Description", "Epic", "SP"},
		ColumnWidths:          []float64{5.5, 13.3, 41.5, 34.2, 5.4},
		CenterColumns:         []int{0, 1, 4},
	}
}
