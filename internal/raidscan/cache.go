package raidscan

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// cacheVersion is bumped whenever the meaning of a cached scope value changes.
const cacheVersion = 1

// maxTrendDays bounds the stored daily history.
const maxTrendDays = 120

// cacheData is the per-team file. Closed sprints never change, so their scope
// is kept; the trend is the only record of past watchlist sizes (Jira cannot
// rebuild it), so a version bump drops the scopes but keeps the trend.
type cacheData struct {
	Version int                `json:"version"`
	Scopes  map[int]ScopePoint `json:"scopes"`
	Trend   []Snapshot         `json:"trend"`
	dirty   bool
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
	return filepath.Join(c.dir, "raid-"+safe+".json")
}

func (c diskCache) load(team string) *cacheData {
	d := &cacheData{Version: cacheVersion, Scopes: map[int]ScopePoint{}}
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
	d.Trend = f.Trend
	if f.Version == cacheVersion && f.Scopes != nil {
		d.Scopes = f.Scopes
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
		return fmt.Errorf("failed to encode RAID cache for team %q: %w", team, err)
	}
	tmp, err := os.CreateTemp(c.dir, "raid-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp RAID cache file for team %q: %w", team, err)
	}
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("failed to write RAID cache for team %q: %w", team, err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("failed to close RAID cache for team %q: %w", team, err)
	}
	if err := os.Rename(tmp.Name(), c.path(team)); err != nil {
		_ = os.Remove(tmp.Name())
		return errors.Join(fmt.Errorf("failed to replace RAID cache file for team %q", team), err)
	}
	d.dirty = false
	return nil
}

// record stores today's snapshot, replacing an earlier one of the same day.
func (d *cacheData) record(s Snapshot) {
	if n := len(d.Trend); n > 0 && d.Trend[n-1].Date == s.Date {
		if d.Trend[n-1] == s {
			return
		}
		d.Trend[n-1] = s
	} else {
		d.Trend = append(d.Trend, s)
		if len(d.Trend) > maxTrendDays {
			d.Trend = d.Trend[len(d.Trend)-maxTrendDays:]
		}
	}
	d.dirty = true
}
