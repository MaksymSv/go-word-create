package word

import (
	"bytes"
	"log"

	"github.com/carmel/gooxml/document"
	"github.com/carmel/gooxml/measurement"
)

// Doc wraps a Word document with section-tracking state
type Doc struct {
	WordDocument document.Document
	sectionCount int
}

// AddHeading adds a styled heading to the document.
// Before every section after the first, a page break is inserted.
// Level-1 headings receive the EPAM Heading 1 style (Trebuchet MS, 16pt, small caps).
func (d *Doc) AddHeading(headingLevel int, headingText string) {
	if d.sectionCount > 0 {
		d.WordDocument.AddParagraph().AddRun().AddPageBreak()
	}
	d.sectionCount++

	para := d.WordDocument.AddParagraph()
	para.Properties().SetHeadingLevel(headingLevel)

	if headingLevel == 1 {
		para.Properties().SetSpacing(300*measurement.Twips, 40*measurement.Twips)
	}

	run := para.AddRun()
	run.AddText(headingText)

	if headingLevel == 1 {
		run.Properties().SetFontFamily("Trebuchet MS")
		run.Properties().SetSize(16)
		run.Properties().SetSmallCaps(true)
		run.Properties().SetCharacterSpacing(5 * measurement.Twips)
	}
}

// NewDocument creates a new document with default settings
func NewDocument() *Doc {
	wordDocument := document.New()
	return &Doc{WordDocument: *wordDocument}
}

// SaveDocumentToFile saves the Word document to the specified output file
func (d *Doc) SaveDocumentToFile(outputFile *string) error {
	err := d.WordDocument.SaveToFile(*outputFile)
	if err != nil {
		log.Fatalf("Failed to save document: %v", err)
		return err
	}
	return nil
}

// SaveDocument saves the Word document to a buffer
func (d *Doc) SaveDocument(buf bytes.Buffer) error {
	err := d.WordDocument.Save(&buf)
	if err != nil {
		log.Fatalf("Failed to save document: %v", err)
		return err
	}
	return nil
}
