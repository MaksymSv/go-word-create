package word

import "testing"

func TestAddHeading(t *testing.T) {
	t.Parallel()

	doc := NewDocument()
	doc.AddHeading(1, "Monthly Report")

	paragraphs := doc.WordDocument.Paragraphs()
	if len(paragraphs) != 2 {
		t.Fatalf("len(Paragraphs) = %d, want %d", len(paragraphs), 2)
	}

	runs := paragraphs[1].Runs()
	if len(runs) != 1 {
		t.Fatalf("len(Runs) = %d, want %d", len(runs), 1)
	}
	if runs[0].Text() != "Monthly Report" {
		t.Fatalf("heading text = %q, want %q", runs[0].Text(), "Monthly Report")
	}
	if paragraphs[1].Style() != "Heading1" {
		t.Fatalf("heading style = %q, want %q", paragraphs[1].Style(), "Heading1")
	}
}
