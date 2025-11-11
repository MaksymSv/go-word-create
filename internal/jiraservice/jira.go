package jiraservice

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	jira "github.com/andygrunwald/go-jira"
)

type JiraService struct {
	client    *jira.Client
	epicField string
	spField   string
	url       string
}

type Issue struct {
	Key         string
	Summary     string
	Epic        string
	StoryPoints float64
	Type        string
	Status      string
	URL         string
}

func NewJiraService(baseURL, username, password, epicField, spField string) (*JiraService, error) {
	tp := jira.BasicAuthTransport{
		Username: username,
		Password: password,
	}

	client, err := jira.NewClient(tp.Client(), baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira client: %w", err)
	}

	return &JiraService{
		client:    client,
		epicField: epicField,
		spField:   spField,
		url:       baseURL,
	}, nil
}

func (s *JiraService) GetBoard(boardName string) (*jira.Board, error) {

	// First, find the board ID
	boards, _, err := s.client.Board.GetAllBoards(&jira.BoardListOptions{
		ProjectKeyOrID: "",
		Name:           boardName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get board: %w", err)
	}

	if len(boards.Values) == 0 {
		return nil, fmt.Errorf("board '%s' not found", boardName)
	}

	//boardID := strconv.Itoa(boards.Values[0].ID)
	//log.Printf("Found board '%s' with ID %d", boardName, boardID)

	return &boards.Values[0], nil
}

// LoadEpics loads all Epic issues for the given project key and returns
// a slice of maps with keys "key" and "value" (summary).
func (s *JiraService) LoadEpics(projectKey string) (map[string]string, error) {
	// JQL to find epics in the project
	jql := fmt.Sprintf("project = %s AND issuetype = Epic ORDER BY key", projectKey)

	// Fetch up to 1000 epics (adjust MaxResults if you expect more)
	opts := &jira.SearchOptions{MaxResults: 1000}
	issues, _, err := s.client.Issue.Search(jql, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search epics: %w", err)
	}

	epics := make(map[string]string)
	for _, is := range issues {
		// Use issue key and summary as value
		val := ""
		if is.Fields.Summary != "" {
			val = is.Fields.Summary
		}
		epics[is.Key] = val
	}

	return epics, nil
}

func (s *JiraService) GetAllBoardIssues(projectKey, boardName string, issuesTypesFilter []string) ([]Issue, error) {
	// First, find the board ID
	board, err := s.GetBoard(boardName)
	if err != nil {
		return nil, err
	}

	boardID := strconv.Itoa(board.ID)
	log.Printf("Found board '%s' with ID %s", boardName, boardID)

	// Get all sprints for the board
	sprints, _, err := s.client.Board.GetAllSprints(boardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sprints: %w", err)
	}

	log.Printf("Found %d sprints for board '%s'", len(sprints), boardName)

	// Fetch epic summaries for the collected epic keys
	epicNames, err := s.LoadEpics(projectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load epics: %w", err)
	}

	// Prepare a filter map from issuesTypes (if provided) for O(1) checks
	typeFilter := createFilterMap(issuesTypesFilter)

	allMonthsIssues := []Issue{}
	for _, sprint := range sprints {
		log.Printf("Sprint: ID=%d, Name=%s, State=%s", sprint.ID, sprint.Name, sprint.State)

		sprintIssues, err := s.LoadIssuesFromSprint(sprint.ID, epicNames, typeFilter)
		if err != nil {
			return nil, err
		}

		allMonthsIssues = append(allMonthsIssues, sprintIssues...)
	}

	return allMonthsIssues, nil
}

func (s *JiraService) GetSprintIssues(projectKey, boardName, sprintName string, issuesTypes []string) ([]Issue, error) {
	// First, find the board ID
	board, err := s.GetBoard(boardName)
	if err != nil {
		return nil, err
	}

	boardID := strconv.Itoa(board.ID)
	log.Printf("Found board '%s' with ID %s", boardName, boardID)

	// Get all sprints for the board
	sprints, _, err := s.client.Board.GetAllSprints(boardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sprints: %w", err)
	}

	log.Printf("Found %d sprints for board '%s'", len(sprints), boardName)

	var targetSprint *jira.Sprint
	for _, sprint := range sprints {
		if sprint.Name == sprintName {
			targetSprint = &sprint
			break
		}
	}

	if targetSprint == nil {
		return nil, fmt.Errorf("sprint '%s' not found", sprintName)
	}

	log.Printf("Found sprint '%s' with ID %d", sprintName, targetSprint.ID)

	// Fetch epic summaries for the collected epic keys
	epicNames, err := s.LoadEpics(projectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load epics: %w", err)
	}

	// Prepare a filter map from issuesTypes (if provided) for O(1) checks
	typeFilter := createFilterMap(issuesTypes)

	result, err := s.LoadIssuesFromSprint(targetSprint.ID, epicNames, typeFilter)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *JiraService) LoadIssuesFromSprint(sprintId int, epicNames map[string]string, typeFilter map[string]struct{}) ([]Issue, error) {
	// Get issues in the sprint
	issues, _, err := s.client.Sprint.GetIssuesForSprint(sprintId)
	if err != nil {
		return nil, fmt.Errorf("failed to get sprint issues: %w", err)
	}

	var result []Issue
	for _, issue := range issues {
		issueType := issue.Fields.Type.Name
		epicName := getEpicName(issue, s.epicField, epicNames)
		storyPoints := getStoryPoints(issue, s.spField)

		// If a filter was provided, only include matching types (case-insensitive)
		if len(typeFilter) > 0 {
			if _, ok := typeFilter[strings.ToLower(strings.TrimSpace(issueType))]; !ok {
				// skip this issue because its type is not in the filter list
				continue
			}
		}

		result = append(result, Issue{
			Key:         issue.Key,
			Summary:     issue.Fields.Summary,
			Epic:        epicName,
			StoryPoints: storyPoints,
			Type:        issueType,
			Status:      issue.Fields.Status.Name,
			URL:         fmt.Sprintf("%s/browse/%s", s.url, issue.Key),
		})
	}

	return result, nil
}

func getStoryPoints(issue jira.Issue, spFieldName string) float64 {
	// Extract story points robustly (field may be float64, int, or string)
	storyPoints := 0.0
	if spField, ok := issue.Fields.Unknowns[spFieldName]; ok {
		switch sp := spField.(type) {
		case float64:
			storyPoints = sp
		case int:
			storyPoints = float64(sp)
		case int64:
			storyPoints = float64(sp)
		case string:
			if f, err := strconv.ParseFloat(sp, 64); err == nil {
				storyPoints = f
			}
		case map[string]interface{}:
			// handle nested representations if present
			if v, ok := sp["value"].(float64); ok {
				storyPoints = v
			} else if sstr, ok := sp["value"].(string); ok {
				if f, err := strconv.ParseFloat(sstr, 64); err == nil {
					storyPoints = f
				}
			}
		}
	}
	return storyPoints
}

func getEpicName(issue jira.Issue, epicFieldName string, epicNames map[string]string) string {
	epicName := ""
	if v, ok := issue.Fields.Unknowns[epicFieldName]; ok {
		switch t := v.(type) {
		case string:
			if name, found := epicNames[t]; found {
				epicName = name
			} else {
				epicName = t
			}
		case map[string]interface{}:
			var key string
			if k, ok := t["key"].(string); ok {
				key = k
			} else if v2, ok := t["value"].(string); ok {
				key = v2
			}
			if key != "" {
				if name, found := epicNames[key]; found {
					epicName = name
				} else {
					epicName = key
				}
			}
		}
	}
	return epicName
}

func createFilterMap(issuesTypes []string) map[string]struct{} {
	typeFilter := make(map[string]struct{})
	if len(issuesTypes) > 0 {
		for _, t := range issuesTypes {
			key := strings.ToLower(strings.TrimSpace(t))
			if key != "" {
				typeFilter[key] = struct{}{}
			}
		}
	}

	return typeFilter
}

// GetIssuesInProgressDuringMonth returns issues that were in 'In Progress' status
// during the specified month, regardless of their current status.
// It checks the issue changelog to find when status changed to "In Progress".
func (s *JiraService) GetIssuesInProgressDuringMonth(projectKey string, monthStart, monthEnd time.Time, issuesTypes []string) ([]Issue, error) {
	// Format dates for JQL: YYYY-MM-DD
	startStr := monthStart.Format("2006-01-02")

	// JQL to find issues created or updated during the month in the project
	// We'll then check their changelog for "In Progress" status changes
	jql := fmt.Sprintf(`project = "%s" AND (created >= "%s" OR updated >= "%s")`, projectKey, startStr, startStr)

	opts := &jira.SearchOptions{MaxResults: 1000, Expand: "changelog"}
	jiraIssues, _, err := s.client.Issue.Search(jql, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search issues: %w", err)
	}

	// Load epics for name resolution
	epicNames, err := s.LoadEpics(projectKey)
	if err != nil {
		log.Printf("warning: failed to load epics: %v", err)
		epicNames = make(map[string]string)
	}

	// Create type filter
	typeFilter := createFilterMap(issuesTypes)

	var result []Issue
	for _, jiraIssue := range jiraIssues {
		// Determine issue type
		issueType := jiraIssue.Fields.Type.Name

		// Apply type filter if provided
		if len(typeFilter) > 0 {
			if _, ok := typeFilter[strings.ToLower(strings.TrimSpace(issueType))]; !ok {
				continue
			}
		}

		// Check if this issue was in 'In Progress' status during the target month
		wasInProgressDuringMonth := false
		if jiraIssue.Changelog != nil {
			for _, history := range jiraIssue.Changelog.Histories {
				// Parse the created timestamp
				createdTime, err := time.Parse("2006-01-02T15:04:05.000-0700", history.Created)
				if err != nil {
					// Try alternate format
					createdTime, err = time.Parse(time.RFC3339, history.Created)
					if err != nil {
						log.Printf("warning: could not parse changelog timestamp %s: %v", history.Created, err)
						continue
					}
				}

				// Check if this change happened during the target month
				if createdTime.Before(monthEnd) && !createdTime.Before(monthStart) {
					// Look for status changes to "In Progress"
					for _, item := range history.Items {
						if item.Field == "status" && item.ToString == "In Progress" {
							wasInProgressDuringMonth = true
							break
						}
					}
				}

				if wasInProgressDuringMonth {
					break
				}
			}
		}

		// Skip if not in progress during month
		if !wasInProgressDuringMonth {
			continue
		}

		// Resolve epic name
		epicName := ""
		if v, ok := jiraIssue.Fields.Unknowns[s.epicField]; ok {
			switch t := v.(type) {
			case string:
				if name, found := epicNames[t]; found {
					epicName = name
				} else {
					epicName = t
				}
			case map[string]interface{}:
				var key string
				if k, ok := t["key"].(string); ok {
					key = k
				} else if v2, ok := t["value"].(string); ok {
					key = v2
				}
				if key != "" {
					if name, found := epicNames[key]; found {
						epicName = name
					} else {
						epicName = key
					}
				}
			}
		}

		// Extract story points
		storyPoints := 0.0
		if spField, ok := jiraIssue.Fields.Unknowns[s.spField]; ok {
			switch sp := spField.(type) {
			case float64:
				storyPoints = sp
			case int:
				storyPoints = float64(sp)
			case int64:
				storyPoints = float64(sp)
			case string:
				if f, err := strconv.ParseFloat(sp, 64); err == nil {
					storyPoints = f
				}
			case map[string]interface{}:
				if v, ok := sp["value"].(float64); ok {
					storyPoints = v
				} else if sstr, ok := sp["value"].(string); ok {
					if f, err := strconv.ParseFloat(sstr, 64); err == nil {
						storyPoints = f
					}
				}
			}
		}

		result = append(result, Issue{
			Key:         jiraIssue.Key,
			Summary:     jiraIssue.Fields.Summary,
			Epic:        epicName,
			StoryPoints: storyPoints,
			Type:        issueType,
			Status:      jiraIssue.Fields.Status.Name,
			URL:         fmt.Sprintf("%s/browse/%s", s.url, jiraIssue.Key),
		})
	}

	return result, nil
}
