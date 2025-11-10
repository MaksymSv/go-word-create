package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"go-word-create/internal/config"
	"go-word-create/internal/jiraservice"
	"go-word-create/internal/wordtable"

	"github.com/carmel/gooxml/document"
)

func main() {
	// Load configuration from .env file
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Define command line flags
	sprintName := flag.String("sprint", "", "Sprint name (required)")
	outputFile := flag.String("output", cfg.OutputFile, "Output file name")
	flag.Parse()

	// Validate required flags
	if *sprintName == "" {
		fmt.Println("Error: Sprint name is required")
		flag.Usage()
		os.Exit(1)
	}

	// Create Jira service
	jiraService, err := jiraservice.NewJiraService(cfg.JiraURL, cfg.JiraUsername, cfg.JiraAPIToken, cfg.JiraEpicField, cfg.JiraSPField)
	if err != nil {
		log.Fatalf("Failed to create Jira service: %v", err)
	}

	// Get issues from sprint
	issues, err := jiraService.GetSprintIssues(cfg.BoardName, *sprintName)
	if err != nil {
		log.Fatalf("Failed to get sprint issues: %v", err)
	}

	// Create Word document
	doc := document.New()
	table := wordtable.NewTable(doc)

	// Add header row
	headers := []string{"Key", "Summary", "Epic", "Story Points"}
	table.AddHeaderRow(headers)

	// Add issue rows
	for _, issue := range issues {
		data := []string{
			issue.Key,
			issue.Summary,
			issue.Epic,
			strconv.FormatFloat(issue.StoryPoints, 'f', 1, 64),
		}
		table.AddDataRow(data)
	}

	// Save the document
	err = doc.SaveToFile(*outputFile)
	if err != nil {
		log.Fatalf("Failed to save document: %v", err)
	}

	fmt.Printf("Created document '%s' with %d issues\n", *outputFile, len(issues))
}
