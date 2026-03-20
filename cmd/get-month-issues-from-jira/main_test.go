package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"go-word-create/internal/config"
	"go-word-create/internal/jiraservice"
)

func TestTruncate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string unchanged",
			input:  "short",
			maxLen: 10,
			want:   "short",
		},
		{
			name:   "exact length unchanged",
			input:  "exact",
			maxLen: 5,
			want:   "exact",
		},
		{
			name:   "long string truncated",
			input:  "abcdef",
			maxLen: 5,
			want:   "ab...",
		},
		{
			name:   "utf8 uses runes",
			input:  "helloworld",
			maxLen: 6,
			want:   "hel...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := truncate(tt.input, tt.maxLen); got != tt.want {
				t.Fatalf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestIsClosedStatus(t *testing.T) {
	t.Parallel()

	closedStatuses := []string{"Closed", "closed", "DONE", "Resolved", "Complete", "Completed"}
	for _, status := range closedStatuses {
		if !isClosedStatus(status) {
			t.Fatalf("expected %q to be treated as closed", status)
		}
	}

	openStatuses := []string{"In Progress", "To Do", "Open", ""}
	for _, status := range openStatuses {
		if isClosedStatus(status) {
			t.Fatalf("expected %q not to be treated as closed", status)
		}
	}
}

func TestLogIssuesTable(t *testing.T) {
	issues := []jiraservice.Issue{
		{
			Type:        "Bug",
			Key:         "PROJ-1",
			Summary:     strings.Repeat("a", 90),
			Epic:        strings.Repeat("b", 50),
			StoryPoints: 3.5,
			Status:      "Done",
		},
	}

	output := captureStdout(t, func() {
		logIssuesTable("Header", issues)
	})

	if !strings.Contains(output, "Header") {
		t.Fatalf("expected output to contain header, got %q", output)
	}
	if !strings.Contains(output, "PROJ-1") {
		t.Fatalf("expected output to contain issue key, got %q", output)
	}

	truncatedSummary := truncate(issues[0].Summary, 80)
	if !strings.Contains(output, truncatedSummary) {
		t.Fatalf("expected output to contain truncated summary %q, got %q", truncatedSummary, output)
	}

	truncatedEpic := truncate(issues[0].Epic, 40)
	if !strings.Contains(output, truncatedEpic) {
		t.Fatalf("expected output to contain truncated epic %q, got %q", truncatedEpic, output)
	}
}

func TestParseMonth(t *testing.T) {
	t.Parallel()

	start, end, err := parseMonth("2025.10")
	if err != nil {
		t.Fatalf("parseMonth() returned error: %v", err)
	}

	if got := start.Format("2006-01-02"); got != "2025-10-01" {
		t.Fatalf("month start = %q, want %q", got, "2025-10-01")
	}
	if got := end.Format("2006-01-02"); got != "2025-11-01" {
		t.Fatalf("month end = %q, want %q", got, "2025-11-01")
	}

	if _, _, err := parseMonth("2025-10"); err == nil {
		t.Fatal("expected invalid month format to return an error")
	}
}

func TestSplitIssuesByStatus(t *testing.T) {
	t.Parallel()

	issues := []jiraservice.Issue{
		{Key: "PROJ-1", Status: "Done"},
		{Key: "PROJ-2", Status: "In Progress"},
		{Key: "PROJ-3", Status: "Resolved"},
	}

	got := splitIssuesByStatus(issues)

	if len(got.Closed) != 2 {
		t.Fatalf("len(Closed) = %d, want %d", len(got.Closed), 2)
	}
	if len(got.Open) != 1 {
		t.Fatalf("len(Open) = %d, want %d", len(got.Open), 1)
	}
	if got.Open[0].Key != "PROJ-2" {
		t.Fatalf("Open[0].Key = %q, want %q", got.Open[0].Key, "PROJ-2")
	}
}

func TestBuildOutputFilename(t *testing.T) {
	t.Parallel()

	monthStart := time.Date(2025, time.October, 1, 0, 0, 0, 0, time.UTC)

	if got := buildOutputFilename("report.docx", monthStart); got != "report - 2025-10.docx" {
		t.Fatalf("buildOutputFilename() = %q, want %q", got, "report - 2025-10.docx")
	}
	if got := buildOutputFilename("report", monthStart); got != "report - 2025-10.docx" {
		t.Fatalf("buildOutputFilename() = %q, want %q", got, "report - 2025-10.docx")
	}
	if got := buildOutputFilename("", monthStart); got != "" {
		t.Fatalf("buildOutputFilename() = %q, want empty string", got)
	}
}

func TestRunDebugMode(t *testing.T) {
	cfg := testConfig()
	fetcher := &fakeMonthIssueFetcher{
		issuesByTeam: map[string][]jiraservice.Issue{
			"Team A": {
				{Key: "PROJ-1", Type: "Bug", Summary: "Closed issue", Status: "Done"},
			},
			"Team B": {
				{Key: "PROJ-2", Type: "Task", Summary: "Open issue", Status: "In Progress"},
			},
		},
	}
	report := &fakeReportBuilder{}
	var stdout bytes.Buffer

	err := run([]string{"-month=2025.10", "-debug"}, runDeps{
		loadConfig: func() (*config.Config, error) { return cfg, nil },
		newJiraService: func(baseURL, username, password, epicField, spField string) (monthIssueFetcher, error) {
			return fetcher, nil
		},
		newReport: func() reportBuilder { return report },
		stdout:    &stdout,
	})
	if err != nil {
		t.Fatalf("run() returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Found 2 issues in 'In Progress' during 2025.10 across 2 teams") {
		t.Fatalf("unexpected debug output: %q", output)
	}
	if !strings.Contains(output, "Closed Issues for team Team A (1):") {
		t.Fatalf("expected debug output to contain Team A heading, got %q", output)
	}
	if !strings.Contains(output, "Open Issues for team Team B (1):") {
		t.Fatalf("expected debug output to contain Team B heading, got %q", output)
	}
	if len(report.headings) != 0 {
		t.Fatalf("expected debug mode not to build report, got headings %#v", report.headings)
	}
}

func TestRunDocumentModeBuildsTablesInTeamOrder(t *testing.T) {
	cfg := testConfig()
	fetcher := &fakeMonthIssueFetcher{
		issuesByTeam: map[string][]jiraservice.Issue{
			"Team A": {
				{Key: "PROJ-1", Status: "Done"},
				{Key: "PROJ-2", Status: "In Progress"},
			},
			"Team B": {
				{Key: "PROJ-3", Status: "Resolved"},
			},
		},
	}
	report := &fakeReportBuilder{}

	err := run([]string{"-month=2025.10"}, runDeps{
		loadConfig: func() (*config.Config, error) { return cfg, nil },
		newJiraService: func(baseURL, username, password, epicField, spField string) (monthIssueFetcher, error) {
			return fetcher, nil
		},
		newReport: func() reportBuilder { return report },
		stdout:    &bytes.Buffer{},
	})
	if err != nil {
		t.Fatalf("run() returned error: %v", err)
	}

	wantHeadings := []string{
		"Closed Issues During October 2025 (Team A)",
		"Issues were in work but not Closed during October 2025 (Team A)",
		"Closed Issues During October 2025 (Team B)",
		"Issues were in work but not Closed during October 2025 (Team B)",
	}
	if !equalStrings(report.headings, wantHeadings) {
		t.Fatalf("headings = %#v, want %#v", report.headings, wantHeadings)
	}
	if report.savedPath != "report - 2025-10.docx" {
		t.Fatalf("savedPath = %q, want %q", report.savedPath, "report - 2025-10.docx")
	}
	if !equalStrings(fetcher.calls, []string{"Team A", "Team B"}) {
		t.Fatalf("team fetch order = %#v, want %#v", fetcher.calls, []string{"Team A", "Team B"})
	}
}

func TestRunPropagatesFetcherError(t *testing.T) {
	cfg := testConfig()
	fetcher := &fakeMonthIssueFetcher{
		errByTeam: map[string]error{
			"Team A": errors.New("boom"),
		},
	}

	err := run([]string{"-month=2025.10"}, runDeps{
		loadConfig: func() (*config.Config, error) { return cfg, nil },
		newJiraService: func(baseURL, username, password, epicField, spField string) (monthIssueFetcher, error) {
			return fetcher, nil
		},
		newReport: func() reportBuilder { return &fakeReportBuilder{} },
		stdout:    &bytes.Buffer{},
	})
	if err == nil {
		t.Fatal("expected run() to return an error")
	}
	if !strings.Contains(err.Error(), "Team A") {
		t.Fatalf("expected error to mention team name, got %v", err)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}

	os.Stdout = writer
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		t.Fatalf("failed to read captured output: %v", err)
	}

	return buf.String()
}

type fakeMonthIssueFetcher struct {
	issuesByTeam map[string][]jiraservice.Issue
	errByTeam    map[string]error
	calls        []string
}

func (f *fakeMonthIssueFetcher) GetIssuesInProgressDuringMonth(projectKey, component string, monthStart, monthEnd time.Time, issuesTypes []string) ([]jiraservice.Issue, error) {
	f.calls = append(f.calls, component)
	if err := f.errByTeam[component]; err != nil {
		return nil, err
	}
	return f.issuesByTeam[component], nil
}

type fakeReportBuilder struct {
	headings  []string
	tableData [][]jiraservice.Issue
	savedPath string
}

func (f *fakeReportBuilder) AddIssuesTable(headingText string, tableContent []jiraservice.Issue) {
	f.headings = append(f.headings, headingText)
	f.tableData = append(f.tableData, tableContent)
}

func (f *fakeReportBuilder) SaveDocumentToFile(outputFile *string) error {
	f.savedPath = *outputFile
	return nil
}

func testConfig() *config.Config {
	return &config.Config{
		JiraURL:       "https://jira.example.com",
		JiraUsername:  "user@example.com",
		JiraAPIToken:  "token",
		ProjectKey:    "PROJ",
		OutputFile:    "report.docx",
		JiraEpicField: "epic",
		JiraSPField:   "story_points",
		Teams:         []string{"Team A", "Team B"},
		IssueTypes:    []string{"Bug", "Task"},
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
