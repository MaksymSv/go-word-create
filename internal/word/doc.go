package word

import (
	"bytes"

	"github.com/carmel/gooxml/document"
)

// Table represents a Word document table wrapper
type Doc struct {
	WordDocument document.Document
}

// AddHeading adds a heading to the document
func (d *Doc) AddHeading(headingLevel int, headingText string) {
	d.WordDocument.AddParagraph().AddRun().AddBreak()
	heading1 := d.WordDocument.AddParagraph()
	heading1.Properties().SetHeadingLevel(headingLevel)
	heading1.AddRun().AddText(headingText)
}

// NewDoc creates a new document with default settings
func NewDocument() *Doc {
	wordDocument := document.New()
	return &Doc{WordDocument: *wordDocument}
}

// SaveDocument saves the Word document to the specified output file
func (d *Doc) SaveDocumentToFile(outputFile *string) error {
	return d.WordDocument.SaveToFile(*outputFile)
}

// SaveDocument saves the Word document to the specified output file
func (d *Doc) SaveDocument(buf bytes.Buffer) error {
	return d.WordDocument.Save(&buf)
}
