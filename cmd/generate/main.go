package main

import (
	"log"
	"os"

	"go-word-create/internal/wordtable"

	"github.com/carmel/gooxml/document"
)

func main() {
	// Create a new Word document
	doc := document.New()

	// Create a new table with default settings
	table := wordtable.NewTable(doc)

	// Add header row
	headers := []string{"types", "id", "name", "epic", "SP"}
	table.AddHeaderRow(headers)

	// Add a sample data row
	data := []string{"Bug", "123", "Login fails", "User Auth", "5"}
	table.AddDataRow(data)

	// Save the document in the current directory
	fileName := "table.docx"
	err := doc.SaveToFile(fileName)
	if err != nil {
		log.Fatalf("error saving file: %s", err)
	}

	log.Printf("Word document '%s' created successfully in %s\n", fileName, getCurrentDir())
}

// getCurrentDir returns the current working directory
func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}
