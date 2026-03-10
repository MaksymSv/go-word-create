package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"go-word-create/internal/config"
	"go-word-create/internal/jiraservice"
	"go-word-create/internal/word"
)

// truncate cuts a string if it's longer than maxLen and adds "..." at the end
// Uses rune-based indexing to properly handle multi-byte UTF-8 characters
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
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

	log.Printf("Fetching issues for sprint '%s'", *sprintName)

	// Create Jira service
	jiraService, err := jiraservice.NewJiraService(cfg.JiraURL, cfg.JiraUsername, cfg.JiraAPIToken, cfg.JiraEpicField, cfg.JiraSPField)
	if err != nil {
		log.Fatalf("Failed to create Jira service: %v", err)
	}

	// Get issues from sprint (use configured issue types)
	issues, err := jiraService.GetSprintIssues(cfg.ProjectKey, cfg.BoardName, *sprintName, cfg.IssueTypes)
	if err != nil {
		log.Fatalf("Failed to get sprint issues: %v", err)
	}

	if *debugMode {
		// Print debug information
		log.Printf("Found %d issues in sprint '%s'\n", len(issues), *sprintName)
		logIssuesTable("\nIssues:", issues)
		log.Printf("\nTotal issues: %d\n", len(issues))
	} else {
		// Create Word document with heading
		doc := word.NewDocument()
		addTableToDocument(doc, fmt.Sprintf("Sprint %s - Issues", *sprintName), issues)

		// Append sprint name to output file (e.g. output.docx -> output - Sprint-16.docx)
		if *outputFile != "" {
			baseFilename := strings.TrimSuffix(*outputFile, ".docx")
			sprintForFile := strings.ReplaceAll(*sprintName, " ", "-")
			*outputFile = fmt.Sprintf("%s - %s.docx", baseFilename, sprintForFile)
		}

		// Save the document
		err = doc.SaveDocumentToFile(outputFile)
		if err != nil {
			log.Fatalf("Failed to save document: %v", err)
		}

		log.Printf("Created document '%s' with %d issues\n", *outputFile, len(issues))
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
	headers := []string{"Type", "ID", "Description", "Epic", "SP"}

	doc.AddHeading(1, headingText)
	table := word.NewTable(&doc.WordDocument)
	table.AddHeaderRow(headers)

	for _, issue := range tableContent {
		data := []string{
			issue.Type,
			issue.Key,
			issue.Summary,
			issue.Epic,
			strconv.FormatFloat(issue.StoryPoints, 'f', 1, 64),
		}
		table.AddDataRow(data)
	}
}
