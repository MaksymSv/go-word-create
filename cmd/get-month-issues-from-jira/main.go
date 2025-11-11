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

	// Query issues that were in 'In Progress' state during the specified month
	// We use JQL with updated date range to find issues modified during the month
	filtered, err := jiraService.GetIssuesInProgressDuringMonth(cfg.ProjectKey, monthStart, monthEnd, []string{"Bug", "Story", "Task"})
	if err != nil {
		log.Fatalf("Failed to get issues in progress: %v", err)
	}

	// split issues into two lists: Closed and all others
	closedIssues := []jiraservice.Issue{}
	openIssues := []jiraservice.Issue{}

	for _, issue := range filtered {
		if issue.Status == "Closed" {
			closedIssues = append(closedIssues, issue)
		} else {
			openIssues = append(openIssues, issue)
		}
	}

	if *debugMode {
		// Print debug information
		fmt.Printf("Found %d issues in 'In Progress' during %s\n", len(filtered), *month)

		logIssuesTable(fmt.Sprintf("\nClosed Issues (%d):", len(closedIssues)), closedIssues)
		logIssuesTable(fmt.Sprintf("\nOpen Issues (%d):", len(openIssues)), openIssues)

		fmt.Printf("\nTotal issues: %d\n", len(filtered))
	} else {
		// Create Word document
		doc := word.NewDocument()

		addTableToDocument(doc, fmt.Sprintf("Closed Issues During %s", monthStart.Format("January 2006")), closedIssues)
		addTableToDocument(doc, fmt.Sprintf("Issues were in work but not Closed during %s", monthStart.Format("January 2006")), openIssues)

		// output file has format some_file.docx. Insert formatted date "yyyy-mm" before .docx
		if outputFile != nil {
			*outputFile = fmt.Sprintf("%s - %s.docx", (*outputFile)[:len(*outputFile)-5], monthStart.Format("2006-01"))
		}

		// Save the document
		err = doc.SaveDocumentToFile(outputFile)
		if err != nil {
			log.Fatalf("Failed to save document: %v", err)
		}

		fmt.Printf("Created document '%s' with %d issues\n", *outputFile, len(filtered))
	}
}

func logIssuesTable(header string, lines []jiraservice.Issue) {
	fmt.Println(header)
	for _, issue := range lines {
		fmt.Printf("%-8s|%-12s|%-80s|%-40s|%.1f|%-12s\n",
			issue.Type, issue.Key, truncate(issue.Summary, 80), truncate(issue.Epic, 40), issue.StoryPoints, issue.Status)
	}
}

func addTableToDocument(doc *word.Doc, headingText string, tableContent []jiraservice.Issue) {

	// Headers
	headers := []string{"Type", "ID", "Description", "Epic", "SP"}

	doc.AddHeading(1, headingText)

	closedIssuesTable := word.NewTable(&doc.WordDocument)
	closedIssuesTable.AddHeaderRow(headers)

	// Add issue rows
	for _, issue := range tableContent {
		data := []string{
			issue.Type,
			issue.Key,
			issue.Summary,
			issue.Epic,
			strconv.FormatFloat(issue.StoryPoints, 'f', 1, 64),
		}
		closedIssuesTable.AddDataRow(data)
	}
}
