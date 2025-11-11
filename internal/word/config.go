package word

import (
	"github.com/carmel/gooxml/color"
	"github.com/carmel/gooxml/document"
)

// TableConfig holds configuration for table appearance
type TableConfig struct {
	// HeaderBackgroundColor is the background color for header cells
	HeaderBackgroundColor color.Color
	// HeaderTextColor is the text color for header cells
	HeaderTextColor color.Color
	// CellMargin is the margin for all cells in centimeters
	CellMargin float64
	// Width is the table width in percentage (0-100)
	Width int
}

// DefaultConfig returns the default table configuration
func DefaultConfig() TableConfig {
	return TableConfig{
		HeaderBackgroundColor: color.RGB(0x36, 0x5F, 0x91), // Dark blue
		HeaderTextColor:       color.White,
		CellMargin:            0.2,
		Width:                 100,
	}
}

// WithConfig creates a new table with custom configuration
func WithConfig(doc *document.Document, config TableConfig) *Table {
	table := NewTable(doc)
	// Apply configuration
	table.table.Properties().SetWidthPercent(float64(config.Width))
	return table
}
