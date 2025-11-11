package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"go-word-create/internal/config"
	"go-word-create/internal/jiraservice"
	"go-word-create/internal/word"
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
	month := flag.String("month", "", "Month in format YYYY.MM (required)")
	outputFile := flag.String("output", cfg.OutputFile, "Output file name")
	debugMode := flag.Bool("debug", false, "Debug mode: print data without generating Word document")
	flag.Parse()

	// Validate required flags
	if *month == "" {
		fmt.Println("Error: Month is required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse month
	monthTime, err := time.Parse("2006.01", *month)
	if err != nil {
		fmt.Printf("Error: Invalid month format. Use YYYY.MM\n")
		os.Exit(1)
	}
	monthStart := monthTime
	monthEnd := monthStart.AddDate(0, 1, 0)
	log.Printf("Filtering issues for month: %s to %s", monthStart.Format("2006-01-02"), monthEnd.Format("2006-01-02"))

	// Create Jira service
	jiraService, err := jiraservice.NewJiraService(cfg.JiraURL, cfg.JiraUsername, cfg.JiraAPIToken, cfg.JiraEpicField, cfg.JiraSPField)
	if err != nil {
		log.Fatalf("Failed to create Jira service: %v", err)
	}

	// Get all issues from board
	issues, err := jiraService.GetAllBoardIssues(cfg.ProjectKey, cfg.BoardName, []string{"Bug", "Feature", "Task"})
	if err != nil {
		log.Fatalf("Failed to get board issues: %v", err)
	}

	filtered := issues

	// Filter issues that were in 'In Progress' state during the specified month
	// var filtered []jiraservice.Issue
	// for _, issue := range issues {
	// 	for _, history := range issue.StatusHistory {
	// 		if history.Status == "In Progress" && !history.ChangedAt.Before(monthStart) && history.ChangedAt.Before(monthEnd) {
	// 			filtered = append(filtered, issue)
	// 			break
	// 		}
	// 	}
	// }

	if *debugMode {
		// Print debug information
		fmt.Printf("Found %d issues in 'In Progress' during %s\n", len(filtered), *month)
		fmt.Println("\nIssues:")
		for _, issue := range filtered {
			fmt.Printf("%-8s|%-12s|%-80s|%-40s|%.1f\n",
				issue.Type, issue.Key, truncate(issue.Summary, 80), truncate(issue.Epic, 40), issue.StoryPoints)
		}
		fmt.Printf("\nTotal issues: %d\n", len(filtered))
	} else {
		// Create Word document
		doc := word.NewDocument()
		table := word.NewTable(&doc.WordDocument)

		// Add header row
		headers := []string{"Type", "Key", "Summary", "Epic", "Story Points"}
		table.AddHeaderRow(headers)

		// Add issue rows
		for _, issue := range filtered {
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
		err = doc.SaveDocumentToFile(outputFile)
		if err != nil {
			log.Fatalf("Failed to save document: %v", err)
		}

		fmt.Printf("Created document '%s' with %d issues\n", *outputFile, len(filtered))
	}
}
