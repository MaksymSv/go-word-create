package server

import (
	"bytes"
	"net/http"

	"go-word-create/internal/wordtable"

	"github.com/carmel/gooxml/document"
)

// Handler handles HTTP requests for document generation
type Handler struct{}

// NewHandler creates a new Handler instance
func NewHandler() *Handler {
	return &Handler{}
}

// GetDocument handles the GET request to generate and return a Word document
func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	// Create a new Word document
	doc := document.New()

	// Create a new table
	table := wordtable.NewTable(doc)

	// Add header row
	headers := []string{"types", "id", "name", "epic", "SP"}
	table.AddHeaderRow(headers)

	// Add sample data rows
	data := [][]string{
		{"Bug", "123", "Login fails", "User Auth", "5"},
		{"Feature", "124", "Add pagination", "User List", "3"},
		{"Task", "125", "Update docs", "Documentation", "1"},
	}

	for _, row := range data {
		table.AddDataRow(row)
	}

	// Create a buffer to store the document
	var buf bytes.Buffer
	err := doc.Save(&buf)
	if err != nil {
		http.Error(w, "Error generating document", http.StatusInternalServerError)
		return
	}

	// Set headers for file download
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Content-Disposition", `attachment; filename="report.docx"`)
	w.Header().Set("Content-Length", string(buf.Len()))

	// Write the document to the response
	_, err = w.Write(buf.Bytes())
	if err != nil {
		http.Error(w, "Error sending document", http.StatusInternalServerError)
		return
	}
}
