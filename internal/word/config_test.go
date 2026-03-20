package word

import (
	"testing"

	"github.com/carmel/gooxml/color"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	if cfg.HeaderBackgroundColor != color.RGB(0x36, 0x5F, 0x91) {
		t.Fatalf("HeaderBackgroundColor = %#v, want %#v", cfg.HeaderBackgroundColor, color.RGB(0x36, 0x5F, 0x91))
	}
	if cfg.HeaderTextColor != color.White {
		t.Fatalf("HeaderTextColor = %#v, want %#v", cfg.HeaderTextColor, color.White)
	}
	if cfg.CellMargin != 0.2 {
		t.Fatalf("CellMargin = %v, want %v", cfg.CellMargin, 0.2)
	}
	if cfg.Width != 100 {
		t.Fatalf("Width = %d, want %d", cfg.Width, 100)
	}
}
