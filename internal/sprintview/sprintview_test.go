package sprintview

import (
	"errors"
	"reflect"
	"testing"

	"go-word-create/internal/jiraservice"
)

type fakeSource struct {
	epicsErr error
	issues   []jiraservice.Issue
	issuesEr error
}

func (f fakeSource) LoadEpics(string) (map[string]string, error) { return nil, f.epicsErr }

func (f fakeSource) GetIssuesFromSprintWithChangelog(int, map[string]string, map[string]struct{}) ([]jiraservice.Issue, error) {
	return f.issues, f.issuesEr
}

func TestBuildKeepsOnlyConfiguredLabels(t *testing.T) {
	src := fakeSource{issues: []jiraservice.Issue{
		{Key: "A-1", Labels: []string{"ai-assisted", "unrelated"}},
		{Key: "A-2"},
	}}
	got, err := Build(src, "A", []string{"ai-assisted", "ai-assisted-dev"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Issues[0].ActiveLabels, []string{"ai-assisted"}) {
		t.Errorf("active labels = %v", got.Issues[0].ActiveLabels)
	}
	if got.Issues[1].ActiveLabels == nil || len(got.Issues[1].ActiveLabels) != 0 {
		t.Errorf("issue without labels must have an empty, non-nil slice: %#v", got.Issues[1].ActiveLabels)
	}
	if len(got.ConfiguredLabels) != 2 {
		t.Errorf("configured = %v", got.ConfiguredLabels)
	}
}

func TestBuildToleratesEpicFailureButNotIssueFailure(t *testing.T) {
	if _, err := Build(fakeSource{epicsErr: errors.New("x")}, "A", nil, 1); err != nil {
		t.Errorf("epic failure should be tolerated: %v", err)
	}
	if _, err := Build(fakeSource{issuesEr: errors.New("x")}, "A", nil, 1); err == nil {
		t.Error("issue failure must be returned")
	}
}
