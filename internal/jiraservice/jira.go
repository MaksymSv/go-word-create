package jiraservice

import (
	"fmt"
	"log"
	"strconv"

	jira "github.com/andygrunwald/go-jira"
)

type JiraService struct {
	client    *jira.Client
	epicField string
	spField   string
}

type Issue struct {
	Key         string
	Summary     string
	Epic        string
	StoryPoints float64
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

	return &JiraService{client: client, epicField: epicField, spField: spField}, nil
}

func (s *JiraService) GetSprintIssues(boardName, sprintName string) ([]Issue, error) {
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

	boardID := strconv.Itoa(boards.Values[0].ID)

	log.Printf("Found board '%s' with ID %d", boardName, boards.Values[0].ID)

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

	// Get issues in the sprint
	issues, _, err := s.client.Sprint.GetIssuesForSprint(targetSprint.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sprint issues: %w", err)
	}

	// Collect unique epic keys referenced by the sprint issues
	epicKeys := make(map[string]struct{})
	for _, issue := range issues {
		if v, ok := issue.Fields.Unknowns[s.epicField]; ok {
			switch t := v.(type) {
			case string:
				if t != "" {
					epicKeys[t] = struct{}{}
				}
			case map[string]interface{}:
				// sometimes the field can be an object containing a key or value
				if key, ok := t["key"].(string); ok && key != "" {
					epicKeys[key] = struct{}{}
				} else if val, ok := t["value"].(string); ok && val != "" {
					epicKeys[val] = struct{}{}
				}
			}
		}
	}

	// Fetch epic summaries for the collected epic keys
	epicNames := make(map[string]string)
	for key := range epicKeys {
		epicIssue, _, err := s.client.Issue.Get(key, nil)
		if err != nil {
			log.Printf("warning: failed to fetch epic %s: %v", key, err)
			epicNames[key] = ""
			continue
		}
		epicNames[key] = epicIssue.Fields.Summary
	}

	// Build result using epic name lookup
	var result []Issue
	for _, issue := range issues {
		epicName := ""
		if v, ok := issue.Fields.Unknowns[s.epicField]; ok {
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

		// Extract story points robustly (field may be float64, int, or string)
		storyPoints := 0.0
		if spField, ok := issue.Fields.Unknowns[s.spField]; ok {
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

		result = append(result, Issue{
			Key:         issue.Key,
			Summary:     issue.Fields.Summary,
			Epic:        epicName,
			StoryPoints: storyPoints,
		})
	}

	return result, nil
}
