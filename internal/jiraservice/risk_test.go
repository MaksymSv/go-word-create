package jiraservice

import "testing"

func est(initial, current float64) reportIssue {
	r := reportIssue{Key: "", EstimateStatistic: &reportEstStat{}, CurrentEstimate: &reportEstStat{}}
	r.EstimateStatistic.StatFieldValue.Value = initial
	r.CurrentEstimate.StatFieldValue.Value = current
	return r
}

func keyed(key string, r reportIssue) reportIssue {
	r.Key = key
	return r
}

func TestReduceSprintReport(t *testing.T) {
	completed := []reportIssue{keyed("A-1", est(3, 5)), keyed("A-2", est(2, 2)), keyed("A-9", est(4, 4))} // A-9 was added
	notCompleted := []reportIssue{keyed("A-3", est(5, 5)), keyed("A-8", est(1, 2))}                       // A-8 was added
	punted := []reportIssue{keyed("A-4", est(8, 8)), keyed("A-7", est(3, 3))}                             // A-7 was added, then removed
	other := []reportIssue{keyed("A-5", est(2, 2))}
	added := map[string]bool{"A-9": true, "A-8": true, "A-7": true}

	got := reduceSprintReport(42, completed, notCompleted, punted, other, added)

	want := SprintScope{
		SprintID:      42,
		Committed:     3 + 2 + 5 + 8 + 2, // everything not added, punted ones included
		Added:         4 + 1,             // A-9, A-8 stay; A-7 left again
		Removed:       8,                 // A-4 only: A-7 was never committed
		Reestimated:   (5 - 3) + (2 - 1), // A-1 and A-8
		Final:         5 + 2 + 4 + 5 + 2 + 2,
		Completed:     5 + 2 + 4,
		CarriedOver:   5 + 2 + 2,
		IssuesAdded:   2,
		IssuesRemoved: 2,
		IssuesDone:    3,
		IssuesCarried: 3,
	}
	if got.SprintID != want.SprintID || got.Committed != want.Committed || got.Added != want.Added || got.Removed != want.Removed ||
		got.Reestimated != want.Reestimated || got.Final != want.Final || got.Completed != want.Completed || got.CarriedOver != want.CarriedOver ||
		got.IssuesAdded != want.IssuesAdded || got.IssuesRemoved != want.IssuesRemoved || got.IssuesDone != want.IssuesDone || got.IssuesCarried != want.IssuesCarried {
		t.Errorf("scope = %+v, want %+v", *got, want)
	}
	if got.Final != got.Committed-got.Removed+got.Added+got.Reestimated {
		t.Errorf("Final %v != Committed-Removed+Added+Reestimated = %v", got.Final, got.Committed-got.Removed+got.Added+got.Reestimated)
	}
}

func TestReduceSprintReportMissingEstimates(t *testing.T) {
	unestimated := reportIssue{Key: "A-1"}
	currentMissing := reportIssue{Key: "A-2", EstimateStatistic: &reportEstStat{}}
	currentMissing.EstimateStatistic.StatFieldValue.Value = 3
	got := reduceSprintReport(1, []reportIssue{unestimated, currentMissing}, nil, nil, nil, nil)
	if got.Committed != 3 || got.Final != 3 || got.Reestimated != 0 {
		t.Errorf("scope = %+v, want committed 3, final 3, reestimated 0", *got)
	}
}
