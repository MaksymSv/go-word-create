package jiraservice

import (
	"testing"
	"time"

	jira "github.com/andygrunwald/go-jira"
)

func TestGetStoryPoints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		value  interface{}
		want   float64
		hasKey bool
	}{
		{name: "missing field", hasKey: false, want: 0},
		{name: "float64", hasKey: true, value: 3.5, want: 3.5},
		{name: "int", hasKey: true, value: 2, want: 2},
		{name: "int64", hasKey: true, value: int64(7), want: 7},
		{name: "string", hasKey: true, value: "5.5", want: 5.5},
		{name: "invalid string", hasKey: true, value: "abc", want: 0},
		{name: "map float value", hasKey: true, value: map[string]interface{}{"value": 8.0}, want: 8},
		{name: "map string value", hasKey: true, value: map[string]interface{}{"value": "13.5"}, want: 13.5},
		{name: "unsupported type", hasKey: true, value: true, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			issue := testIssueWithUnknown("story_points", tt.value, tt.hasKey)
			if got := getStoryPoints(issue, "story_points"); got != tt.want {
				t.Fatalf("getStoryPoints() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEpicName(t *testing.T) {
	t.Parallel()

	epicNames := map[string]string{
		"EPIC-1": "Epic One",
		"EPIC-2": "Epic Two",
	}

	tests := []struct {
		name   string
		value  interface{}
		want   string
		hasKey bool
	}{
		{name: "missing field", hasKey: false, want: ""},
		{name: "string resolved", hasKey: true, value: "EPIC-1", want: "Epic One"},
		{name: "string fallback", hasKey: true, value: "EPIC-9", want: "EPIC-9"},
		{name: "map key resolved", hasKey: true, value: map[string]interface{}{"key": "EPIC-2"}, want: "Epic Two"},
		{name: "map value fallback", hasKey: true, value: map[string]interface{}{"value": "EPIC-9"}, want: "EPIC-9"},
		{name: "malformed map", hasKey: true, value: map[string]interface{}{"key": 123}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			issue := testIssueWithUnknown("epic_link", tt.value, tt.hasKey)
			if got := getEpicName(issue, "epic_link", epicNames); got != tt.want {
				t.Fatalf("getEpicName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCreateFilterMap(t *testing.T) {
	t.Parallel()

	got := createFilterMap([]string{" Bug ", "Story", "", "story", " Task "})

	if len(got) != 3 {
		t.Fatalf("len(createFilterMap()) = %d, want %d", len(got), 3)
	}

	for _, key := range []string{"bug", "story", "task"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("expected filter map to contain %q", key)
		}
	}
}

func TestMatchesTypeFilter(t *testing.T) {
	t.Parallel()

	filter := createFilterMap([]string{" Bug ", "Story"})

	if !matchesTypeFilter("bug", filter) {
		t.Fatal("expected bug to match filter")
	}
	if !matchesTypeFilter(" Story ", filter) {
		t.Fatal("expected story to match filter")
	}
	if matchesTypeFilter("Task", filter) {
		t.Fatal("expected task not to match filter")
	}
	if !matchesTypeFilter("Anything", nil) {
		t.Fatal("expected nil filter to match everything")
	}
}

func TestParseChangelogTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "jira timestamp format",
			input: "2025-10-15T09:30:00.000+0000",
			want:  "2025-10-15T09:30:00Z",
		},
		{
			name:  "rfc3339 fallback",
			input: "2025-10-16T12:00:00Z",
			want:  "2025-10-16T12:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseChangelogTime(tt.input)
			if err != nil {
				t.Fatalf("parseChangelogTime() returned error: %v", err)
			}
			if got.Format(time.RFC3339) != tt.want {
				t.Fatalf("parseChangelogTime() = %q, want %q", got.Format(time.RFC3339), tt.want)
			}
		})
	}

	if _, err := parseChangelogTime("invalid"); err == nil {
		t.Fatal("expected invalid changelog timestamp to return an error")
	}
}

func TestWasInProgressDuringMonth(t *testing.T) {
	t.Parallel()

	monthStart := time.Date(2025, time.October, 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)

	if !wasInProgressDuringMonth([]jira.ChangelogHistory{
		{
			Created: "2025-10-15T09:30:00.000+0000",
			Items: []jira.ChangelogItems{
				{Field: "status", ToString: "In Progress"},
			},
		},
	}, monthStart, monthEnd) {
		t.Fatal("expected in-month status change to match")
	}

	if wasInProgressDuringMonth([]jira.ChangelogHistory{
		{
			Created: "2025-09-30T23:59:59.000+0000",
			Items: []jira.ChangelogItems{
				{Field: "status", ToString: "In Progress"},
			},
		},
	}, monthStart, monthEnd) {
		t.Fatal("expected out-of-month change not to match")
	}

	if wasInProgressDuringMonth([]jira.ChangelogHistory{
		{
			Created: "2025-10-15T09:30:00.000+0000",
			Items: []jira.ChangelogItems{
				{Field: "status", ToString: "Done"},
			},
		},
	}, monthStart, monthEnd) {
		t.Fatal("expected non-In Progress status change not to match")
	}

	if wasInProgressDuringMonth([]jira.ChangelogHistory{
		{
			Created: "invalid",
			Items: []jira.ChangelogItems{
				{Field: "status", ToString: "In Progress"},
			},
		},
	}, monthStart, monthEnd) {
		t.Fatal("expected invalid timestamp not to match")
	}

	if wasInProgressDuringMonth([]jira.ChangelogHistory{
		{
			Created: monthEnd.Format(time.RFC3339),
			Items: []jira.ChangelogItems{
				{Field: "status", ToString: "In Progress"},
			},
		},
	}, monthStart, monthEnd) {
		t.Fatal("expected monthEnd boundary not to match")
	}
}

func TestMatchesComponent(t *testing.T) {
	t.Parallel()

	components := []*jira.Component{
		{Name: "Processing"},
		nil,
		{Name: "Discovery"},
	}

	if !matchesComponent(components, " processing ") {
		t.Fatal("expected component match to be case-insensitive and trimmed")
	}
	if matchesComponent(components, "Stabletek") {
		t.Fatal("expected non-matching component to return false")
	}
	if !matchesComponent(components, "") {
		t.Fatal("expected empty component filter to match all")
	}
}

func TestMapJiraIssueToIssue(t *testing.T) {
	t.Parallel()

	service := &JiraService{
		epicField: "epic_link",
		spField:   "story_points",
		url:       "https://jira.example.com",
	}
	jiraIssue := jira.Issue{
		Key: "PROJ-1",
		Fields: &jira.IssueFields{
			Summary: "Issue summary",
			Type: jira.IssueType{
				Name: "Bug",
			},
			Status: &jira.Status{
				Name: "Done",
			},
			Unknowns: map[string]interface{}{
				"epic_link":    "EPIC-1",
				"story_points": "5.5",
			},
		},
	}

	got := service.mapJiraIssueToIssue(jiraIssue, jiraIssue.Fields.Type.Name, map[string]string{"EPIC-1": "Epic One"})

	if got.Key != "PROJ-1" || got.Summary != "Issue summary" || got.Epic != "Epic One" || got.StoryPoints != 5.5 || got.Type != "Bug" || got.Status != "Done" {
		t.Fatalf("unexpected mapped issue: %#v", got)
	}
	if got.URL != "https://jira.example.com/browse/PROJ-1" {
		t.Fatalf("URL = %q, want %q", got.URL, "https://jira.example.com/browse/PROJ-1")
	}
}

func testIssueWithUnknown(fieldName string, value interface{}, hasKey bool) jira.Issue {
	unknowns := map[string]interface{}{}
	if hasKey {
		unknowns[fieldName] = value
	}

	return jira.Issue{
		Fields: &jira.IssueFields{
			Unknowns: unknowns,
		},
	}
}
