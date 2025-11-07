package jiraservice

import (
	"fmt"

	jira "github.com/andygrunwald/go-jira"
)

type JiraService struct {
	client *jira.Client
}

type Issue struct {
	Key         string
	Summary     string
	Epic        string
	StoryPoints float64
}

func NewJiraService(baseURL, username, password string) (*JiraService, error) {
	tp := jira.BasicAuthTransport{
		Username: username,
		Password: password,
	}

	client, err := jira.NewClient(tp.Client(), baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira client: %w", err)
	}

	return &JiraService{client: client}, nil
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

	//boardID := boards.Values[0].ID

	// Get all sprints for the board
	sprints, _, err := s.client.Board.GetAllSprints(boardName)
	if err != nil {
		return nil, fmt.Errorf("failed to get sprints: %w", err)
	}

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

	// Get issues in the sprint
	issues, _, err := s.client.Sprint.GetIssuesForSprint(targetSprint.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sprint issues: %w", err)
	}

	var result []Issue
	for _, issue := range issues {
		epic := ""
		if epicField, ok := issue.Fields.Unknowns["customfield_10014"].(string); ok {
			epic = epicField
		}

		storyPoints := 0.0
		if spField, ok := issue.Fields.Unknowns["customfield_10026"].(float64); ok {
			storyPoints = spField
		}

		result = append(result, Issue{
			Key:         issue.Key,
			Summary:     issue.Fields.Summary,
			Epic:        epic,
			StoryPoints: storyPoints,
		})
	}

	return result, nil
}
