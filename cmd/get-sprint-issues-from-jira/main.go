package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"go-word-create/internal/config"
	"go-word-create/internal/issuecsv"
	"go-word-create/internal/jiraservice"
	"go-word-create/internal/word"
)

const (
	formatDocx = "docx"
	formatCSV  = "csv"
)

func writeCSV(path string, issues []jiraservice.Issue) (err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
	return issuecsv.Write(f, issues)
}

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
	sprintName := flag.String("sprint", "", "Sprint name, or comma-separated sprint names, e.g. \"ST26.20,PC26.20\" (required)")
	outputFile := flag.String("output", cfg.OutputFile, "Output file name")
	format := flag.String("format", formatDocx, "Output format: docx (one document for all sprints) or csv (one file per sprint)")
	outputDir := flag.String("output-dir", ".", "Directory for CSV files (csv format only)")
	debugMode := flag.Bool("debug", false, "Debug mode: print data without generating Word document")
	flag.Parse()

	// Validate required flags
	if *sprintName == "" {
		fmt.Println("Error: Sprint name is required")
		flag.Usage()
		os.Exit(1)
	}

	if *format != formatDocx && *format != formatCSV {
		fmt.Printf("Error: -format must be %q or %q, got %q\n", formatDocx, formatCSV, *format)
		os.Exit(1)
	}

	// Create Jira service
	jiraService, err := jiraservice.NewJiraService(cfg.JiraURL, cfg.JiraUsername, cfg.JiraAPIToken, cfg.JiraEpicField, cfg.JiraSPField)
	if err != nil {
		log.Fatalf("Failed to create Jira service: %v", err)
	}

	sprintNames, err := jiraservice.ParseSprintNames(*sprintName)
	if err != nil {
		log.Fatalf("Failed to parse -sprint: %v", err)
	}

	// Get issues from every sprint
	var issues []jiraservice.Issue
	for _, sprint := range sprintNames {
		sprintIssues, err := jiraService.GetSprintIssues(cfg.ProjectKey, sprint, cfg.GetAllBoardNames(), cfg.JiraIssueTypes)
		if err != nil {
			log.Fatalf("Failed to get issues for sprint %q: %v", sprint, err)
		}

		if *format == formatCSV && !*debugMode {
			path := filepath.Join(*outputDir, issuecsv.FileName(sprint, time.Now()))
			if err := writeCSV(path, sprintIssues); err != nil {
				log.Fatalf("Failed to write CSV for sprint %q: %v", sprint, err)
			}
			fmt.Printf("Created CSV '%s' with %d issues\n", path, len(sprintIssues))
		}

		if *debugMode {
			if len(sprintNames) > 1 {
				fmt.Printf("\n=== Sprint: %s ===\n", sprint)
			}
			fmt.Printf("Found %d issues in sprint '%s'\n", len(sprintIssues), sprint)
			fmt.Println("\nIssues:")
			for _, issue := range sprintIssues {
				// Truncate strings that are too long
				fmt.Printf("%-8s|%-12s|%-80s|%-40s|%.1f|%s\n",
					issue.Type, issue.Key, truncate(issue.Summary, 80), truncate(issue.Epic, 40), issue.StoryPoints, issue.Status)
			}
			fmt.Printf("\nTotal issues: %d\n", len(sprintIssues))
		}

		issues = append(issues, sprintIssues...)
	}

	if !*debugMode && *format == formatDocx {
		// Create Word document
		doc := word.NewDocument()
		cfg := word.DefaultConfig()
		cfg.ColumnHeaders = []string{"Type", "Key", "Summary", "Epic", "Story Points", "Status"}
		cfg.ColumnWidths = []float64{5.5, 9.0, 38.0, 29.0, 5.5, 13.0}
		table := word.NewTable(&doc.WordDocument, cfg)
		table.AddHeaderRow()

		// Add issue rows
		for _, issue := range issues {
			data := []string{
				issue.Type,
				issue.Key,
				issue.Summary,
				issue.Epic,
				strconv.FormatFloat(issue.StoryPoints, 'f', 1, 64),
				issue.Status,
			}
			table.AddDataRow(data)
		}

		// Save the document
		err = doc.SaveDocumentToFile(outputFile)
		if err != nil {
			log.Fatalf("Failed to save document: %v", err)
		}

		fmt.Printf("Created document '%s' with %d issues\n", *outputFile, len(issues))
	}
}
