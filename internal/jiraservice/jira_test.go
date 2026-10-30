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

// assignChange is an assignee change item; an empty string means unassigned.
func assignChange(from, to string) jira.ChangelogItems {
	return jira.ChangelogItems{Field: "assignee", FromString: from, ToString: to}
}

// entry groups the items of one changelog history entry.
func entry(it ...jira.ChangelogItems) []jira.ChangelogItems { return it }

// statusChange is a status change item moving the issue to the given status.
func statusChange(to string) jira.ChangelogItems {
	return jira.ChangelogItems{Field: "status", ToString: to}
}

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
			// The assignee never changed, so the current assignee is also the
			// creation-time assignee and counts at the "In Review." move.
			name: "in_review_no_assignee_change_in_history_fields_assignee_set",
			issue: jira.Issue{
				Fields: &jira.IssueFields{Assignee: assignee("Alice")},
				Changelog: &jira.Changelog{
					Histories: history(items("status", "In Review.")),
				},
			},
			want: "Alice",
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
			// as a changelog entry. With no assignee change at all, the current
			// assignee is the creation-time assignee and counts at "In Progress".
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
			want: "Alice",
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

		// User story 1: assignee before the first move to "In Progress"
		{
			name: "us1_reassigned_after_start_keeps_original",
			issue: jira.Issue{
				Fields: &jira.IssueFields{Assignee: assignee("Bob")},
				Changelog: &jira.Changelog{
					Histories: history(
						entry(statusChange("In Progress")),
						entry(assignChange("Alice", "Bob")),
						entry(statusChange("In Review.")),
					),
				},
			},
			want: "Alice",
		},
		{
			name: "us1_reassigned_before_start_takes_latest",
			issue: jira.Issue{
				Fields: &jira.IssueFields{Assignee: assignee("Bob")},
				Changelog: &jira.Changelog{
					Histories: history(
						entry(assignChange("Alice", "Bob")),
						entry(statusChange("In Progress")),
					),
				},
			},
			want: "Bob",
		},
		{
			name: "us1_assigned_at_creation_never_changed",
			issue: jira.Issue{
				Fields: &jira.IssueFields{Assignee: assignee("Alice")},
				Changelog: &jira.Changelog{
					Histories: history(entry(statusChange("In Progress"))),
				},
			},
			want: "Alice",
		},
		{
			name: "us1_assignee_set_in_same_entry_as_move_counts",
			issue: jira.Issue{
				Fields: &jira.IssueFields{Assignee: assignee("Carol")},
				Changelog: &jira.Changelog{
					Histories: history(entry(assignChange("", "Carol"), statusChange("In Progress"))),
				},
			},
			want: "Carol",
		},
		{
			name: "us1_first_of_several_in_progress_moves_wins",
			issue: jira.Issue{
				Fields: &jira.IssueFields{Assignee: assignee("Bob")},
				Changelog: &jira.Changelog{
					Histories: history(
						entry(assignChange("", "Alice"), statusChange("In Progress")),
						entry(statusChange("To Do")),
						entry(assignChange("Alice", "Bob"), statusChange("In Progress")),
					),
				},
			},
			want: "Alice",
		},
		{
			name: "us1_status_name_case_insensitive",
			issue: jira.Issue{
				Fields: &jira.IssueFields{Assignee: assignee("Dave")},
				Changelog: &jira.Changelog{
					Histories: history(entry(assignChange("", "Dave"), statusChange("IN PROGRESS"))),
				},
			},
			want: "Dave",
		},
		{
			// A different status that merely starts with "In Progress" must not trigger
			// the rule; the current assignee is returned instead of the earlier Eve.
			name: "us1_similar_status_name_does_not_match",
			issue: jira.Issue{
				Fields: &jira.IssueFields{Assignee: assignee("Frank")},
				Changelog: &jira.Changelog{
					Histories: history(
						entry(statusChange("In Progress (Dev)")),
						entry(assignChange("Eve", "Frank")),
					),
				},
			},
			want: "Frank",
		},

		// User story 2: fall back to the assignee before the first "In Review." move
		{
			name: "us2_unassigned_at_start_assigned_before_review",
			issue: jira.Issue{
				Fields: &jira.IssueFields{Assignee: assignee("Carl")},
				Changelog: &jira.Changelog{
					Histories: history(
						entry(statusChange("In Progress")),
						entry(assignChange("", "Cara")),
						entry(statusChange("In Review.")),
						entry(assignChange("Cara", "Carl")),
					),
				},
			},
			want: "Cara",
		},
		{
			name: "us2_skipped_in_progress",
			issue: jira.Issue{
				Fields: &jira.IssueFields{Assignee: assignee("Dina")},
				Changelog: &jira.Changelog{
					Histories: history(
						entry(assignChange("", "Cara")),
						entry(statusChange("In Review.")),
						entry(assignChange("Cara", "Dina")),
					),
				},
			},
			want: "Cara",
		},
		{
			name: "us2_in_progress_assignee_wins_over_in_review_assignee",
			issue: jira.Issue{
				Fields: &jira.IssueFields{Assignee: assignee("Eli")},
				Changelog: &jira.Changelog{
					Histories: history(
						entry(assignChange("", "Alice")),
						entry(statusChange("In Progress")),
						entry(assignChange("Alice", "Eli")),
						entry(statusChange("In Review.")),
					),
				},
			},
			want: "Alice",
		},
		{
			name: "us2_assignee_removed_before_start_falls_back",
			issue: jira.Issue{
				Fields: &jira.IssueFields{Assignee: assignee("Dina")},
				Changelog: &jira.Changelog{
					Histories: history(
						entry(assignChange("Alice", "")),
						entry(statusChange("In Progress")),
						entry(assignChange("", "Dina")),
						entry(statusChange("In Review.")),
					),
				},
			},
			want: "Dina",
		},
		{
			// Only the first move to "In Progress" counts, even when it had no assignee.
			name: "us2_later_in_progress_move_does_not_replace_empty_first",
			issue: jira.Issue{
				Fields: &jira.IssueFields{Assignee: assignee("Fay")},
				Changelog: &jira.Changelog{
					Histories: history(
						entry(statusChange("In Progress")),
						entry(assignChange("", "Fay")),
						entry(statusChange("In Progress")),
						entry(statusChange("In Review.")),
					),
				},
			},
			want: "Fay",
		},
		{
			name: "us2_first_in_review_move_wins",
			issue: jira.Issue{
				Fields: &jira.IssueFields{Assignee: assignee("Gus")},
				Changelog: &jira.Changelog{
					Histories: history(
						entry(assignChange("", "Cara")),
						entry(statusChange("In Review.")),
						entry(assignChange("Cara", "Gus")),
						entry(statusChange("In Review.")),
					),
				},
			},
			want: "Cara",
		},
		{
			// Same entry: the assignee set together with the "In Review." move counts.
			name: "us2_assignee_set_in_same_entry_as_review_move",
			issue: jira.Issue{
				Fields: &jira.IssueFields{},
				Changelog: &jira.Changelog{
					Histories: history(entry(assignChange("", "Hank"), statusChange("In Review."))),
				},
			},
			want: "Hank",
		},

		// User story 3: neither moment applies, so use the current assignee, then empty
		{
			name: "us3_never_reached_either_status_uses_current_assignee",
			issue: jira.Issue{
				Fields: &jira.IssueFields{Assignee: assignee("Ivy")},
				Changelog: &jira.Changelog{
					Histories: history(entry(statusChange("Closed"))),
				},
			},
			want: "Ivy",
		},
		{
			name: "us3_assignee_changed_but_never_started_uses_current_assignee",
			issue: jira.Issue{
				Fields: &jira.IssueFields{Assignee: assignee("Bob")},
				Changelog: &jira.Changelog{
					Histories: history(entry(assignChange("Alice", "Bob"))),
				},
			},
			want: "Bob",
		},
		{
			name: "us3_no_assignee_anywhere_is_empty",
			issue: jira.Issue{
				Fields: &jira.IssueFields{},
				Changelog: &jira.Changelog{
					Histories: history(
						entry(statusChange("In Progress")),
						entry(statusChange("In Review.")),
						entry(statusChange("Closed")),
					),
				},
			},
			want: "",
		},
		{
			name:  "us3_nil_fields_and_nil_changelog_is_empty",
			issue: jira.Issue{},
			want:  "",
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
