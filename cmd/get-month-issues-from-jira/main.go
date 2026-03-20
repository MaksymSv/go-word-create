package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

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

// TeamIssues groups closed and open issues for a specific team/component.
type TeamIssues struct {
	Closed []jiraservice.Issue
	Open   []jiraservice.Issue
}

type monthIssueFetcher interface {
	GetIssuesInProgressDuringMonth(projectKey, component string, monthStart, monthEnd time.Time, issuesTypes []string) ([]jiraservice.Issue, error)
}

type reportBuilder interface {
	AddIssuesTable(headingText string, tableContent []jiraservice.Issue)
	SaveDocumentToFile(outputFile *string) error
}

type wordReportBuilder struct {
	doc *word.Doc
}

type runDeps struct {
	loadConfig     func() (*config.Config, error)
	newJiraService func(baseURL, username, password, epicField, spField string) (monthIssueFetcher, error)
	newReport      func() reportBuilder
	stdout         io.Writer
}

var errCLI = errors.New("cli error")

func defaultRunDeps() runDeps {
	return runDeps{
		loadConfig: config.Load,
		newJiraService: func(baseURL, username, password, epicField, spField string) (monthIssueFetcher, error) {
			return jiraservice.NewJiraService(baseURL, username, password, epicField, spField)
		},
		newReport: func() reportBuilder {
			return &wordReportBuilder{doc: word.NewDocument()}
		},
		stdout: os.Stdout,
	}
}

// isClosedStatus checks if a status represents a closed/completed issue.
func isClosedStatus(status string) bool {
	closedStatuses := []string{"Closed", "Done", "Resolved", "Complete", "Completed"}
	statusLower := strings.ToLower(status)
	for _, closedStatus := range closedStatuses {
		if strings.ToLower(closedStatus) == statusLower {
			return true
		}
	}
	return false
}

func main() {
	if err := run(os.Args[1:], defaultRunDeps()); err != nil {
		if errors.Is(err, errCLI) {
			os.Exit(1)
		}
		log.Fatal(err)
	}
}

func logIssuesTable(header string, lines []jiraservice.Issue) {
	writeIssuesTable(os.Stdout, header, lines)
}

func writeIssuesTable(w io.Writer, header string, lines []jiraservice.Issue) {
	fmt.Fprintln(w, header)
	for _, issue := range lines {
		fmt.Fprintf(w, "%-8s|%-12s|%-80s|%-40s|%.1f|%-12s\n",
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

func (b *wordReportBuilder) AddIssuesTable(headingText string, tableContent []jiraservice.Issue) {
	addTableToDocument(b.doc, headingText, tableContent)
}

func (b *wordReportBuilder) SaveDocumentToFile(outputFile *string) error {
	return b.doc.SaveDocumentToFile(outputFile)
}

func parseMonth(month string) (time.Time, time.Time, error) {
	monthStart, err := time.Parse("2006.01", month)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid month format. use YYYY.MM")
	}

	return monthStart, monthStart.AddDate(0, 1, 0), nil
}

func splitIssuesByStatus(issues []jiraservice.Issue) TeamIssues {
	var teamIssues TeamIssues

	for _, issue := range issues {
		if isClosedStatus(issue.Status) {
			teamIssues.Closed = append(teamIssues.Closed, issue)
			continue
		}
		teamIssues.Open = append(teamIssues.Open, issue)
	}

	return teamIssues
}

func buildOutputFilename(outputFile string, monthStart time.Time) string {
	if outputFile == "" {
		return outputFile
	}

	baseFilename := strings.TrimSuffix(outputFile, ".docx")
	return fmt.Sprintf("%s - %s.docx", baseFilename, monthStart.Format("2006-01"))
}

func run(args []string, deps runDeps) error {
	if deps.stdout == nil {
		deps.stdout = io.Discard
	}

	cfg, err := deps.loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	flagSet := flag.NewFlagSet("get-month-issues-from-jira", flag.ContinueOnError)
	flagSet.SetOutput(deps.stdout)

	month := flagSet.String("month", "", "Month in format YYYY.MM (required)")
	outputFile := flagSet.String("output", cfg.OutputFile, "Output file name")
	debugMode := flagSet.Bool("debug", false, "Debug mode: print data without generating Word document")
	if err := flagSet.Parse(args); err != nil {
		return errCLI
	}

	if *month == "" {
		fmt.Fprintln(deps.stdout, "Error: Month is required")
		flagSet.Usage()
		return errCLI
	}

	monthStart, monthEnd, err := parseMonth(*month)
	if err != nil {
		fmt.Fprintln(deps.stdout, "Error: Invalid month format. Use YYYY.MM")
		return errCLI
	}

	log.Printf("Filtering issues for month: %s to %s", monthStart.Format("2006-01-02"), monthEnd.Format("2006-01-02"))

	jiraService, err := deps.newJiraService(cfg.JiraURL, cfg.JiraUsername, cfg.JiraAPIToken, cfg.JiraEpicField, cfg.JiraSPField)
	if err != nil {
		return fmt.Errorf("failed to create Jira service: %w", err)
	}

	teamIssues := make(map[string]TeamIssues)
	totalIssuesCount := 0

	for _, team := range cfg.Teams {
		log.Printf("Fetching issues for team/component '%s'", team)

		issuesForTeam, err := jiraService.GetIssuesInProgressDuringMonth(cfg.ProjectKey, team, monthStart, monthEnd, cfg.IssueTypes)
		if err != nil {
			return fmt.Errorf("failed to get issues in progress for team '%s': %w", team, err)
		}

		totalIssuesCount += len(issuesForTeam)
		teamIssues[team] = splitIssuesByStatus(issuesForTeam)
	}

	if *debugMode {
		fmt.Fprintf(deps.stdout, "Found %d issues in 'In Progress' during %s across %d teams\n", totalIssuesCount, *month, len(cfg.Teams))

		for _, team := range cfg.Teams {
			ti, ok := teamIssues[team]
			if !ok {
				continue
			}

			writeIssuesTable(deps.stdout, fmt.Sprintf("\nClosed Issues for team %s (%d):", team, len(ti.Closed)), ti.Closed)
			writeIssuesTable(deps.stdout, fmt.Sprintf("\nOpen Issues for team %s (%d):", team, len(ti.Open)), ti.Open)
		}

		fmt.Fprintf(deps.stdout, "\nTotal issues across all teams: %d\n", totalIssuesCount)
		return nil
	}

	report := deps.newReport()
	for _, team := range cfg.Teams {
		ti, ok := teamIssues[team]
		if !ok {
			continue
		}

		report.AddIssuesTable(fmt.Sprintf("Closed Issues During %s (%s)", monthStart.Format("January 2006"), team), ti.Closed)
		report.AddIssuesTable(fmt.Sprintf("Issues were in work but not Closed during %s (%s)", monthStart.Format("January 2006"), team), ti.Open)
	}

	*outputFile = buildOutputFilename(*outputFile, monthStart)
	if err := report.SaveDocumentToFile(outputFile); err != nil {
		return fmt.Errorf("failed to save document: %w", err)
	}

	log.Printf("Created document '%s' with %d issues across %d teams\n", *outputFile, totalIssuesCount, len(cfg.Teams))
	return nil
}
