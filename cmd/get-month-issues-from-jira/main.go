package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"go-word-create/internal/config"
	"go-word-create/internal/jiraservice"
	"go-word-create/internal/monthreport"
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

	// For each team (component) configured in TEAMS, fetch and split issues.
	teams := make([]monthreport.TeamIssues, 0, len(cfg.Teams))
	totalIssuesCount := 0

	for _, team := range cfg.Teams {
		log.Printf("Fetching issues for team/component '%s'", team.ComponentName)

		issuesForTeam, err := jiraService.GetIssuesInProgressDuringMonth(cfg.ProjectKey, team.ComponentName, monthStart, monthEnd, cfg.JiraIssueTypes)
		if err != nil {
			log.Fatalf("Failed to get issues in progress for team '%s': %v", team.ComponentName, err)
		}

		totalIssuesCount += len(issuesForTeam)
		teams = append(teams, monthreport.Split(team.ComponentName, issuesForTeam))
	}

	if *debugMode {
		// Print debug information grouped by team
		log.Printf("Found %d issues in 'In Progress' during %s across %d teams\n", totalIssuesCount, *month, len(cfg.Teams))

		for _, ti := range teams {
			logIssuesTable(fmt.Sprintf("\nClosed Issues for team %s (%d):", ti.Component, len(ti.Closed)), ti.Closed)
			logIssuesTable(fmt.Sprintf("\nOpen Issues for team %s (%d):", ti.Component, len(ti.Open)), ti.Open)
		}

		log.Printf("\nTotal issues across all teams: %d\n", totalIssuesCount)
		return
	}

	// Create Word document, grouped by team, and save it as "<name> - YYYY-MM.docx"
	doc := monthreport.BuildDocument(monthStart, teams)
	fileName := monthreport.FileName(*outputFile, monthStart)
	if err := doc.SaveDocumentToFile(&fileName); err != nil {
		log.Fatalf("Failed to save document: %v", err)
	}

	log.Printf("Created document '%s' with %d issues across %d teams\n", fileName, totalIssuesCount, len(cfg.Teams))
}

func logIssuesTable(header string, lines []jiraservice.Issue) {
	fmt.Println(header)
	for _, issue := range lines {
		fmt.Printf("%-8s|%-12s|%-80s|%-40s|%.1f|%-12s\n",
			issue.Type, issue.Key, truncate(issue.Summary, 80), truncate(issue.Epic, 40), issue.StoryPoints, issue.Status)
	}
}
