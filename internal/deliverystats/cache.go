package deliverystats

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// cacheVersion is bumped whenever the meaning of cached values changes.
const cacheVersion = 2

// sprintDerived holds the values for a closed sprint that need an expensive
// changelog scan. A closed sprint's history does not change, so they are kept
// on disk and reused.
type sprintDerived struct {
	Start      time.Time      `json:"start"` // sprint start: backlog snapshot time
	From       time.Time      `json:"from"`  // flow window [From, To)
	To         time.Time      `json:"to"`
	Backlog    BacklogPoint   `json:"backlog"`
	LeadCycle  LeadCyclePoint `json:"leadCycle"`
	ComputedAt time.Time      `json:"computedAt"`
}

// sprintVelocity is a closed sprint's committed/completed story points. Jira's
// velocity chart only returns the last 7 sprints, so values are kept as they are
// seen and the stored history grows beyond that window over time.
type sprintVelocity struct {
	Committed float64 `json:"committed"`
	Completed float64 `json:"completed"`
}

// cacheData is the per-team cache file. Fingerprint guards Sprints only (it
// changes with issue types / resolutions); Velocity and Closed are facts about
// the past and are never invalidated.
type cacheData struct {
	Fingerprint string                 `json:"fingerprint"`
	Sprints     map[int]sprintDerived  `json:"sprints"`
	Velocity    map[int]sprintVelocity `json:"velocity"`
	Closed      map[string]time.Time   `json:"closed"` // issue key -> close time of issues closed without a resolution date
	dirty       bool
}

// diskCache stores one JSON file per team below dir. An empty dir disables it.
type diskCache struct {
	dir string
}

func (c diskCache) path(team string) string {
	safe := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r == ' ' {
			return '_'
		}
		return r
	}, team)
	return filepath.Join(c.dir, safe+".json")
}

// load returns the team's cache; a missing or unreadable file yields an empty
// one, and a stale fingerprint drops the derived sprints but keeps velocity.
func (c diskCache) load(team, fingerprint string) *cacheData {
	d := &cacheData{Fingerprint: fingerprint, Sprints: map[int]sprintDerived{}, Velocity: map[int]sprintVelocity{}, Closed: map[string]time.Time{}}
	if c.dir == "" {
		return d
	}
	raw, err := os.ReadFile(c.path(team))
	if err != nil {
		return d
	}
	var f cacheData
	if json.Unmarshal(raw, &f) != nil {
		return d
	}
	if f.Velocity != nil {
		d.Velocity = f.Velocity
	}
	if f.Closed != nil {
		d.Closed = f.Closed
	}
	if f.Fingerprint == fingerprint && f.Sprints != nil {
		d.Sprints = f.Sprints
	}
	return d
}

// save writes the cache atomically (temp file + rename) when it changed.
func (c diskCache) save(team string, d *cacheData) error {
	if c.dir == "" || !d.dirty {
		return nil
	}
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return fmt.Errorf("failed to create cache dir %q: %w", c.dir, err)
	}
	raw, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode cache for team %q: %w", team, err)
	}
	tmp, err := os.CreateTemp(c.dir, "delivery-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp cache file for team %q: %w", team, err)
	}
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("failed to write cache for team %q: %w", team, err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("failed to close cache for team %q: %w", team, err)
	}
	if err := os.Rename(tmp.Name(), c.path(team)); err != nil {
		_ = os.Remove(tmp.Name())
		return errors.Join(fmt.Errorf("failed to replace cache file for team %q", team), err)
	}
	d.dirty = false
	return nil
}

// fresh reports whether a cached entry still describes the sprint as it is now.
func (d sprintDerived) fresh(start, from, to time.Time) bool {
	return d.Start.Equal(start) && d.From.Equal(from) && d.To.Equal(to)
}
