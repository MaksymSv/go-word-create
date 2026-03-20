package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadRequiresEnvironmentVariables(t *testing.T) {
	requiredVars := []string{
		"JIRA_URL",
		"JIRA_USERNAME",
		"JIRA_API_TOKEN",
		"JIRA_PROJECT_KEY",
		"TEAMS",
	}

	for _, missingVar := range requiredVars {
		t.Run(missingVar, func(t *testing.T) {
			withTempWorkingDirectory(t)
			setBaseEnv(t)
			t.Setenv(missingVar, "")

			_, err := Load()
			if err == nil {
				t.Fatalf("expected missing %s to return an error", missingVar)
			}
			if !strings.Contains(err.Error(), missingVar) {
				t.Fatalf("expected error to mention %s, got %v", missingVar, err)
			}
		})
	}
}

func TestLoadParsesTeamsIssueTypesAndDefaults(t *testing.T) {
	withTempWorkingDirectory(t)
	setBaseEnv(t)
	t.Setenv("TEAMS", " Team A,Team B , , Team C ")
	t.Setenv("ISSUE_TYPES", " Bug, Story , ,Task ")
	t.Setenv("DEFAULT_OUTPUT_FILE", "report.docx")
	t.Setenv("JIRA_EPIC_FIELD", "customfield_epic")
	t.Setenv("JIRA_SP_FIELD", "customfield_sp")
	t.Setenv("JIRA_COMPONENT_FIELD", "custom_components")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	wantTeams := []string{"Team A", "Team B", "Team C"}
	if !equalSlices(cfg.Teams, wantTeams) {
		t.Fatalf("Teams = %#v, want %#v", cfg.Teams, wantTeams)
	}

	wantIssueTypes := []string{"Bug", "Story", "Task"}
	if !equalSlices(cfg.IssueTypes, wantIssueTypes) {
		t.Fatalf("IssueTypes = %#v, want %#v", cfg.IssueTypes, wantIssueTypes)
	}

	if cfg.OutputFile != "report.docx" {
		t.Fatalf("OutputFile = %q, want %q", cfg.OutputFile, "report.docx")
	}
	if cfg.JiraEpicField != "customfield_epic" {
		t.Fatalf("JiraEpicField = %q, want %q", cfg.JiraEpicField, "customfield_epic")
	}
	if cfg.JiraSPField != "customfield_sp" {
		t.Fatalf("JiraSPField = %q, want %q", cfg.JiraSPField, "customfield_sp")
	}
	if cfg.JiraComponentField != "custom_components" {
		t.Fatalf("JiraComponentField = %q, want %q", cfg.JiraComponentField, "custom_components")
	}
}

func TestLoadRejectsEmptyTeams(t *testing.T) {
	withTempWorkingDirectory(t)
	setBaseEnv(t)
	t.Setenv("TEAMS", " , , ")

	_, err := Load()
	if err == nil {
		t.Fatal("expected empty TEAMS to return an error")
	}
	if !strings.Contains(err.Error(), "no teams configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadUsesDefaultIssueTypesAndOptionalDefaults(t *testing.T) {
	withTempWorkingDirectory(t)
	setBaseEnv(t)
	t.Setenv("ISSUE_TYPES", " , ")
	t.Setenv("DEFAULT_OUTPUT_FILE", "")
	t.Setenv("JIRA_EPIC_FIELD", "")
	t.Setenv("JIRA_SP_FIELD", "")
	t.Setenv("JIRA_COMPONENT_FIELD", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	wantIssueTypes := []string{"Bug", "Story", "Task"}
	if !equalSlices(cfg.IssueTypes, wantIssueTypes) {
		t.Fatalf("IssueTypes = %#v, want %#v", cfg.IssueTypes, wantIssueTypes)
	}

	if cfg.OutputFile != "sprint-issues.docx" {
		t.Fatalf("OutputFile = %q, want %q", cfg.OutputFile, "sprint-issues.docx")
	}
	if cfg.JiraEpicField != "customfield_14500" {
		t.Fatalf("JiraEpicField = %q, want %q", cfg.JiraEpicField, "customfield_14500")
	}
	if cfg.JiraSPField != "customfield_10004" {
		t.Fatalf("JiraSPField = %q, want %q", cfg.JiraSPField, "customfield_10004")
	}
	if cfg.JiraComponentField != "components" {
		t.Fatalf("JiraComponentField = %q, want %q", cfg.JiraComponentField, "components")
	}
}

func TestGetEnvWithDefault(t *testing.T) {
	t.Setenv("TEST_KEY", "configured")
	if got := getEnvWithDefault("TEST_KEY", "fallback"); got != "configured" {
		t.Fatalf("getEnvWithDefault() = %q, want %q", got, "configured")
	}

	t.Setenv("TEST_KEY", "")
	if got := getEnvWithDefault("TEST_KEY", "fallback"); got != "fallback" {
		t.Fatalf("getEnvWithDefault() = %q, want %q", got, "fallback")
	}
}

func setBaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("JIRA_URL", "https://jira.example.com")
	t.Setenv("JIRA_USERNAME", "user@example.com")
	t.Setenv("JIRA_API_TOKEN", "token")
	t.Setenv("JIRA_PROJECT_KEY", "PROJ")
	t.Setenv("TEAMS", "Team A")
	t.Setenv("ISSUE_TYPES", "")
	t.Setenv("DEFAULT_OUTPUT_FILE", "")
	t.Setenv("JIRA_EPIC_FIELD", "")
	t.Setenv("JIRA_SP_FIELD", "")
	t.Setenv("JIRA_COMPONENT_FIELD", "")
}

func withTempWorkingDirectory(t *testing.T) {
	t.Helper()

	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(currentDir); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})
}

func equalSlices(got, want []string) bool {
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
