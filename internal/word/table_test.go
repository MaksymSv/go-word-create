package word

import (
	"testing"

	"github.com/carmel/gooxml/document"
	"github.com/carmel/gooxml/schema/soo/wml"
)

func TestAddHeaderRow(t *testing.T) {
	t.Parallel()

	doc := document.New()
	table := NewTable(doc)
	table.AddHeaderRow([]string{"Type", "ID", "Description"})

	tables := doc.Tables()
	if len(tables) != 1 {
		t.Fatalf("len(Tables) = %d, want %d", len(tables), 1)
	}

	rows := tables[0].Rows()
	if len(rows) != 1 {
		t.Fatalf("len(Rows) = %d, want %d", len(rows), 1)
	}

	cells := rows[0].Cells()
	if len(cells) != 3 {
		t.Fatalf("len(Cells) = %d, want %d", len(cells), 3)
	}

	wantTexts := []string{"Type", "ID", "Description"}
	for i, cell := range cells {
		if got := firstCellText(cell); got != wantTexts[i] {
			t.Fatalf("cell %d text = %q, want %q", i, got, wantTexts[i])
		}
	}
}

func TestAddDataRow(t *testing.T) {
	t.Parallel()

	doc := document.New()
	table := NewTable(doc)
	table.AddDataRow([]string{"Bug", "PROJ-1", "Summary", "Epic", "3.0"})

	rows := doc.Tables()[0].Rows()
	if len(rows) != 1 {
		t.Fatalf("len(Rows) = %d, want %d", len(rows), 1)
	}

	cells := rows[0].Cells()
	if len(cells) != 5 {
		t.Fatalf("len(Cells) = %d, want %d", len(cells), 5)
	}

	wantTexts := []string{"Bug", "PROJ-1", "Summary", "Epic", "3.0"}
	for i, cell := range cells {
		if got := firstCellText(cell); got != wantTexts[i] {
			t.Fatalf("cell %d text = %q, want %q", i, got, wantTexts[i])
		}
	}

	if got := cells[0].Paragraphs()[0].Properties().X().Jc.ValAttr; got != wml.ST_JcCenter {
		t.Fatalf("column 0 alignment = %v, want %v", got, wml.ST_JcCenter)
	}
	if got := cells[1].Paragraphs()[0].Properties().X().Jc.ValAttr; got != wml.ST_JcCenter {
		t.Fatalf("column 1 alignment = %v, want %v", got, wml.ST_JcCenter)
	}
	if cells[2].Paragraphs()[0].Properties().X().Jc != nil {
		t.Fatalf("column 2 alignment = %v, want nil", cells[2].Paragraphs()[0].Properties().X().Jc)
	}
	if cells[3].Paragraphs()[0].Properties().X().Jc != nil {
		t.Fatalf("column 3 alignment = %v, want nil", cells[3].Paragraphs()[0].Properties().X().Jc)
	}
	if got := cells[4].Paragraphs()[0].Properties().X().Jc.ValAttr; got != wml.ST_JcCenter {
		t.Fatalf("column 4 alignment = %v, want %v", got, wml.ST_JcCenter)
	}
}

func firstCellText(cell document.Cell) string {
	paragraphs := cell.Paragraphs()
	if len(paragraphs) == 0 {
		return ""
	}

	runs := paragraphs[0].Runs()
	if len(runs) == 0 {
		return ""
	}

	return runs[0].Text()
}
