package jiraservice

import (
	"testing"

	jira "github.com/andygrunwald/go-jira"
)

func items(pairs ...string) []jira.ChangelogItems {
	var out []jira.ChangelogItems
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, jira.ChangelogItems{Field: pairs[i], ToString: pairs[i+1]})
	}
	return out
}

func history(entries ...[]jira.ChangelogItems) []jira.ChangelogHistory {
	var out []jira.ChangelogHistory
	for _, e := range entries {
		out = append(out, jira.ChangelogHistory{Items: e})
	}
	return out
}

func assignee(name string) *jira.User { return &jira.User{DisplayName: name} }

func TestResolveImplementer(t *testing.T) {
	tests := []struct {
		name  string
		issue jira.Issue
		want  string
	}{
		// Path A: Changelog == nil (not expanded in the API response)
		{
			name:  "nil_changelog_nil_assignee",
			issue: jira.Issue{Fields: &jira.IssueFields{}},
			want:  "",
		},
		{
			name:  "nil_changelog_with_assignee",
			issue: jira.Issue{Fields: &jira.IssueFields{Assignee: assignee("Alice")}},
			want:  "Alice",
		},

		// Path B: Changelog present but empty
		{
			name:  "empty_histories_nil_assignee",
			issue: jira.Issue{Fields: &jira.IssueFields{}, Changelog: &jira.Changelog{}},
			want:  "",
		},
		{
			name: "empty_histories_with_assignee",
			issue: jira.Issue{
				Fields:    &jira.IssueFields{Assignee: assignee("Alice")},
				Changelog: &jira.Changelog{},
			},
			want: "Alice",
		},

		// Path C: "In Review." found — currentAssignee tracking
		{
			name: "in_review_no_assignee_change_in_history_no_fields_assignee",
			issue: jira.Issue{
				Fields: &jira.IssueFields{},
				Changelog: &jira.Changelog{
					Histories: history(items("status", "In Review.")),
				},
			},
			want: "",
		},
		{
			// fields.Assignee is NOT consulted when "In Review." is found;
			// only currentAssignee (built from changelog) is returned.
			name: "in_review_no_assignee_change_in_history_fields_assignee_set",
			issue: jira.Issue{
				Fields: &jira.IssueFields{Assignee: assignee("Alice")},
				Changelog: &jira.Changelog{
					Histories: history(items("status", "In Review.")),
				},
			},
			want: "",
		},
		{
			name: "in_review_assignee_in_same_entry",
			issue: jira.Issue{
				Fields: &jira.IssueFields{},
				Changelog: &jira.Changelog{
					Histories: history(items("assignee", "Bob", "status", "In Review.")),
				},
			},
			want: "Bob",
		},
		{
			name: "in_review_assignee_in_prior_entry",
			issue: jira.Issue{
				Fields: &jira.IssueFields{},
				Changelog: &jira.Changelog{
					Histories: history(
						items("assignee", "Carol"),
						items("status", "In Review."),
					),
				},
			},
			want: "Carol",
		},
		{
			name: "in_review_multiple_assignee_changes_last_before_trigger_wins",
			issue: jira.Issue{
				Fields: &jira.IssueFields{},
				Changelog: &jira.Changelog{
					Histories: history(
						items("assignee", "Dan"),
						items("assignee", "Eve"),
						items("status", "In Review."),
					),
				},
			},
			want: "Eve",
		},
		{
			name: "in_review_first_occurrence_wins_over_second",
			issue: jira.Issue{
				Fields: &jira.IssueFields{},
				Changelog: &jira.Changelog{
					Histories: history(
						items("status", "In Review."),
						items("assignee", "Frank", "status", "In Review."),
					),
				},
			},
			want: "",
		},

		// Path C: case-insensitivity and exact string matching
		{
			name: "in_review_status_value_case_insensitive",
			issue: jira.Issue{
				Fields: &jira.IssueFields{},
				Changelog: &jira.Changelog{
					Histories: history(items("assignee", "Grace", "status", "IN REVIEW.")),
				},
			},
			want: "Grace",
		},
		{
			name: "in_review_field_names_case_insensitive",
			issue: jira.Issue{
				Fields: &jira.IssueFields{},
				Changelog: &jira.Changelog{
					Histories: history(items("Assignee", "Heidi", "Status", "In Review.")),
				},
			},
			want: "Heidi",
		},
		{
			// "In Review" without trailing period must NOT match; falls through to fields.Assignee.
			name: "in_review_without_trailing_period_no_match",
			issue: jira.Issue{
				Fields: &jira.IssueFields{Assignee: assignee("Ivan")},
				Changelog: &jira.Changelog{
					Histories: history(items("status", "In Review")),
				},
			},
			want: "Ivan",
		},

		// Path D: No "In Review." — fallback to fields.Assignee
		{
			name: "no_in_review_with_current_assignee",
			issue: jira.Issue{
				Fields: &jira.IssueFields{Assignee: assignee("Judy")},
				Changelog: &jira.Changelog{
					Histories: history(
						items("status", "In Progress"),
						items("status", "Ready for QA"),
					),
				},
			},
			want: "Judy",
		},
		{
			name: "no_in_review_nil_current_assignee",
			issue: jira.Issue{
				Fields: &jira.IssueFields{},
				Changelog: &jira.Changelog{
					Histories: history(
						items("status", "In Progress"),
						items("status", "Ready for QA"),
					),
				},
			},
			want: "",
		},

		// Path E: Realistic full-flow scenarios
		{
			// Issue assigned at creation; Jira does not record the initial assignment
			// as a changelog entry — only subsequent changes appear. When status reaches
			// "In Review." but no assignee item was ever logged, currentAssignee stays ""
			// and fields.Assignee is NOT consulted at that point.
			name: "full_flow_assigned_at_creation_no_history_assignee_change",
			issue: jira.Issue{
				Fields: &jira.IssueFields{Assignee: assignee("Alice")},
				Changelog: &jira.Changelog{
					Histories: history(
						items("status", "In Progress"),
						items("status", "In Review."),
						items("status", "Ready for QA"),
					),
				},
			},
			want: "",
		},
		{
			name: "full_flow_assigned_when_moved_to_in_progress",
			issue: jira.Issue{
				Fields: &jira.IssueFields{},
				Changelog: &jira.Changelog{
					Histories: history(
						items("assignee", "Alice", "status", "In Progress"),
						items("status", "In Review."),
					),
				},
			},
			want: "Alice",
		},
		{
			name: "full_flow_assigned_at_same_time_as_in_review",
			issue: jira.Issue{
				Fields: &jira.IssueFields{},
				Changelog: &jira.Changelog{
					Histories: history(
						items("status", "In Progress"),
						items("assignee", "Bob", "status", "In Review."),
					),
				},
			},
			want: "Bob",
		},
		{
			name: "qa_bug_return_first_in_review_wins",
			issue: jira.Issue{
				Fields: &jira.IssueFields{},
				Changelog: &jira.Changelog{
					Histories: history(
						items("assignee", "Alice", "status", "In Progress"),
						items("status", "In Review."), // first — determines result
						items("status", "Ready for QA"),
						items("status", "In QA"),
						items("status", "In Progress"), // bug found, returned
						items("status", "In Review."),  // second — ignored
					),
				},
			},
			want: "Alice",
		},
		{
			name: "qa_bug_return_reassigned_first_in_review_still_wins",
			issue: jira.Issue{
				Fields: &jira.IssueFields{},
				Changelog: &jira.Changelog{
					Histories: history(
						items("assignee", "Alice", "status", "In Progress"),
						items("status", "In Review."), // first — determines result
						items("status", "Ready for QA"),
						items("status", "In QA"),
						items("assignee", "Bob", "status", "In Progress"), // reassigned after bug
						items("status", "In Review."),                     // Bob's review — ignored
					),
				},
			},
			want: "Alice",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveImplementer(tc.issue)
			if got != tc.want {
				t.Errorf("resolveImplementer() = %q, want %q", got, tc.want)
			}
		})
	}
}
