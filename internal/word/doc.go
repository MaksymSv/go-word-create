package word

import (
	"bytes"
	"log"

	"github.com/carmel/gooxml/document"
)

// Table represents a Word document table wrapper
type Doc struct {
	WordDocument document.Document
}

// NewDoc creates a new document with default settings
func NewDocument() *Doc {
	wordDocument := document.New()
	return &Doc{WordDocument: *wordDocument}
}

// SaveDocument saves the Word document to the specified output file
func (d *Doc) SaveDocumentToFile(outputFile *string) error {
	// Save the document
	err := d.WordDocument.SaveToFile(*outputFile)
	if err != nil {
		log.Fatalf("Failed to save document: %v", err)
		return err
	}
	return nil
}

// SaveDocument saves the Word document to the specified output file
func (d *Doc) SaveDocument(buf bytes.Buffer) error {
	// Save the document
	err := d.WordDocument.Save(&buf)
	if err != nil {
		log.Fatalf("Failed to save document: %v", err)
		return err
	}
	return nil
}
