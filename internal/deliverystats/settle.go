package deliverystats

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"go-word-create/internal/jiraservice"
)

// closureBatch is the number of issue keys fetched per changelog query.
const closureBatch = 50

// closureFromChanges returns when the issue last moved into its current
// (Done-category) status.
func closureFromChanges(is jiraservice.HistoryIssue) (time.Time, bool) {
	for i := len(is.Changes) - 1; i >= 0; i-- {
		c := is.Changes[i]
		if c.Field == "status" && strings.EqualFold(strings.TrimSpace(c.ToString), strings.TrimSpace(is.Status)) {
			return c.At, true
		}
	}
	return time.Time{}, false
}

// settleResolution makes Resolved reflect whether the issue is really finished.
// Jira teams often close issues without setting a resolution, so the resolution
// alone cannot tell open from closed. The status category decides:
//   - not Done: the issue is open now (Resolved is cleared, even when a resolution is still set after a reopen);
//   - Done with a resolution date: kept;
//   - Done without one: the time it entered its current status, from the changelog.
//
// Issues without a changelog are looked up once in a separate query and the
// result is remembered in cache.Closed (a closure time never changes).
// Issues with an unknown status category are left untouched.
func (s *Service) settleResolution(issues []jiraservice.HistoryIssue, cache *cacheData) error {
	var need []int
	for i := range issues {
		is := &issues[i]
		switch cat := strings.ToLower(is.StatusCategory); {
		case cat == "":
			continue
		case cat != "done":
			is.Resolved = time.Time{}
		case !is.Resolved.IsZero():
			continue
		default:
			if t, ok := closureFromChanges(*is); ok {
				is.Resolved = t
			} else if t, ok := cache.Closed[is.Key]; ok {
				is.Resolved = t
			} else {
				need = append(need, i)
			}
		}
	}
	if len(need) == 0 {
		return nil
	}

	byKey := make(map[string]int, len(need))
	keys := make([]string, 0, len(need))
	for _, i := range need {
		byKey[issues[i].Key] = i
		keys = append(keys, issues[i].Key)
	}
	sort.Strings(keys)
	for from := 0; from < len(keys); from += closureBatch {
		to := from + closureBatch
		if to > len(keys) {
			to = len(keys)
		}
		jql := fmt.Sprintf("key in (%s)", strings.Join(keys[from:to], ","))
		withLog, err := s.src.SearchHistoryIssues(jql, true)
		if err != nil {
			return fmt.Errorf("failed to read the close dates of %d issue(s) closed without a resolution: %w", len(keys), err)
		}
		for _, w := range withLog {
			i, ok := byKey[w.Key]
			if !ok {
				continue
			}
			t, found := closureFromChanges(w)
			if !found {
				// No status transition recorded: treat it as closed when it was created.
				t = issues[i].Created
				log.Printf("deliverystats: %s is %q (done) but has no status transition; using its creation time as close time", w.Key, w.Status)
			}
			issues[i].Resolved = t
			cache.Closed[w.Key] = t
			cache.dirty = true
		}
	}
	return nil
}
