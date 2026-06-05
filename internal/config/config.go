package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// TeamEntry holds the Jira component name (used to filter issues) and the
// Jira board display name (used for sprint queries) for one configured team.
type TeamEntry struct {
	ComponentName string
	BoardName     string
}

type Config struct {
	JiraURL            string
	JiraUsername       string
	JiraAPIToken       string
	ProjectKey         string
	OutputFile         string
	JiraEpicField      string
	JiraSPField        string
	JiraComponentField string
	Teams              []TeamEntry
	ReportLabels       []string
	JiraIssueTypes     []string
}

// Load reads the configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	// Check required environment variables
	requiredVars := []string{
		"JIRA_URL",
		"JIRA_USERNAME",
		"JIRA_API_TOKEN",
		"JIRA_PROJECT_KEY",
		"TEAMS",
	}

	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			return nil, fmt.Errorf("required environment variable %s is not set", v)
		}
	}

	teams := parseTeams(os.Getenv("TEAMS"))
	reportLabels := parseReportLabels(os.Getenv("REPORT_LABELS"))
	jiraIssueTypes := parseIssueTypes(os.Getenv("ISSUE_TYPES"))

	config := &Config{
		JiraURL:            os.Getenv("JIRA_URL"),
		JiraUsername:       os.Getenv("JIRA_USERNAME"),
		JiraAPIToken:       os.Getenv("JIRA_API_TOKEN"),
		ProjectKey:         os.Getenv("JIRA_PROJECT_KEY"),
		OutputFile:         getEnvWithDefault("DEFAULT_OUTPUT_FILE", "sprint-issues.docx"),
		JiraEpicField:      getEnvWithDefault("JIRA_EPIC_FIELD", "customfield_14500"),
		JiraSPField:        getEnvWithDefault("JIRA_SP_FIELD", "customfield_10004"),
		JiraComponentField: getEnvWithDefault("JIRA_COMPONENT_FIELD", "components"),
		Teams:              teams,
		ReportLabels:       reportLabels,
		JiraIssueTypes:     jiraIssueTypes,
	}

	return config, nil
}

func parseIssueTypes(s string) []string {
	if s == "" {
		return []string{"Bug", "Story", "Task"}
	}
	var types []string
	for _, t := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(t); trimmed != "" {
			types = append(types, trimmed)
		}
	}
	if len(types) == 0 {
		return []string{"Bug", "Story", "Task"}
	}
	return types
}

// parseTeams parses TEAMS env var entries of the form "COMP|\"Board Name\",..."
// into TeamEntry values. When no pipe separator is present the component name
// is used as the board name (backwards-compatible fallback).
func parseTeams(raw string) []TeamEntry {
	var teams []TeamEntry
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "|", 2)
		componentName := strings.TrimSpace(parts[0])
		var boardName string
		if len(parts) == 2 {
			boardName = strings.TrimSpace(parts[1])
			boardName = strings.Trim(boardName, `"`)
		} else {
			boardName = componentName
		}
		teams = append(teams, TeamEntry{ComponentName: componentName, BoardName: boardName})
	}
	return teams
}

// getEnvWithDefault returns environment variable value or default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseReportLabels parses REPORT_LABELS env var; falls back to the default AI label set.
func parseReportLabels(raw string) []string {
	if raw != "" {
		var labels []string
		for _, l := range strings.Split(raw, ",") {
			if trimmed := strings.TrimSpace(l); trimmed != "" {
				labels = append(labels, trimmed)
			}
		}
		if len(labels) > 0 {
			return labels
		}
	}
	return []string{"ai-assisted", "ai-assisted-ba", "ai-assisted-dev", "ai-assisted-qa"}
}

func (c *Config) GetTeamByComponent(componentName string) TeamEntry {
	for _, team := range c.Teams {
		if team.ComponentName == componentName {
			return team
		}
	}
	log.Printf("Warning: no team configured for component %q\n", componentName)
	return TeamEntry{}
}

func (c *Config) GetAllBoardNames() []string {
	boardNames := make([]string, len(c.Teams))
	for i, team := range c.Teams {
		boardNames[i] = team.BoardName
	}
	return boardNames
}
