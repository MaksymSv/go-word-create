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

// truncate cuts a string if it's longer than maxLen and adds "..." at the end
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func main() {
	// Load configuration from .env file
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Define command line flags
	sprintName := flag.String("sprint", "", "Sprint name (required)")
	outputFile := flag.String("output", cfg.OutputFile, "Output file name")
	debugMode := flag.Bool("debug", false, "Debug mode: print data without generating Word document")
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
	issues, err := jiraService.GetSprintIssues(cfg.BoardName, *sprintName, []string{"Bug", "Feature", "Task"})
	if err != nil {
		log.Fatalf("Failed to get sprint issues: %v", err)
	}

	if *debugMode {
		// Print debug information
		fmt.Printf("Found %d issues in sprint '%s'\n", len(issues), *sprintName)
		fmt.Println("\nIssues:")
		for _, issue := range issues {
			// Truncate strings that are too long
			fmt.Printf("%-8s|%-12s|%-80s|%-40s|%.1f\n",
				issue.Type, issue.Key, truncate(issue.Summary, 80), truncate(issue.Epic, 40), issue.StoryPoints)
		}
		fmt.Printf("\nTotal issues: %d\n", len(issues))
	} else {
		// Create Word document
		doc := document.New()
		table := wordtable.NewTable(doc)

		// Add header row
		headers := []string{"Type", "Key", "Summary", "Epic", "Story Points"}
		table.AddHeaderRow(headers)

		// Add issue rows
		for _, issue := range issues {
			data := []string{
				issue.Type,
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
}
