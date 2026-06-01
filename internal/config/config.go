package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	JiraURL            string
	JiraUsername       string
	JiraAPIToken       string
	BoardName          string
	ProjectKey         string
	OutputFile         string
	JiraEpicField      string
	JiraSPField        string
	JiraComponentField string
	Teams              []string
	ReportLabels       []string
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
		"JIRA_BOARD_NAME",
		"JIRA_PROJECT_KEY",
		"TEAMS",
	}

	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			return nil, fmt.Errorf("required environment variable %s is not set", v)
		}
	}

	// Parse TEAMS environment variable into a slice (required).
	rawTeams := os.Getenv("TEAMS")
	var teams []string
	for _, t := range strings.Split(rawTeams, ",") {
		if trimmed := strings.TrimSpace(t); trimmed != "" {
			teams = append(teams, trimmed)
		}
	}

	reportLabels := parseReportLabels(os.Getenv("REPORT_LABELS"))

	config := &Config{
		JiraURL:            os.Getenv("JIRA_URL"),
		JiraUsername:       os.Getenv("JIRA_USERNAME"),
		JiraAPIToken:       os.Getenv("JIRA_API_TOKEN"),
		BoardName:          os.Getenv("JIRA_BOARD_NAME"),
		ProjectKey:         os.Getenv("JIRA_PROJECT_KEY"),
		OutputFile:         getEnvWithDefault("DEFAULT_OUTPUT_FILE", "sprint-issues.docx"),
		JiraEpicField:      getEnvWithDefault("JIRA_EPIC_FIELD", "customfield_14500"),
		JiraSPField:        getEnvWithDefault("JIRA_SP_FIELD", "customfield_10004"),
		JiraComponentField: getEnvWithDefault("JIRA_COMPONENT_FIELD", "components"),
		Teams:              teams,
		ReportLabels:       reportLabels,
	}

	return config, nil
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
