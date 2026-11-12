package raidscan

import (
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"go-word-create/internal/config"
	"go-word-create/internal/jiraservice"
)

// Source is the Jira access the service needs; *jiraservice.JiraService implements it.
type Source interface {
	ActiveSprint(boardName string) (*jiraservice.SprintInfo, error)
	ClosedSprints(boardName string, limit int) ([]jiraservice.SprintInfo, error)
	SprintScope(boardName string, sprintID int) (*jiraservice.SprintScope, error)
	SearchRiskIssues(jql string, withChangelog bool) ([]jiraservice.RiskIssue, error)
	SearchKeys(jql string) (map[string]struct{}, error)
}

// Team is one configured team: the Jira component filters issues, the board resolves sprints.
type Team struct {
	Component string
	Board     string
}

// Options configures a Service; zero values fall back to the defaults in New.
type Options struct {
	ProjectKey         string
	IssueTypes         []string
	BlockedStatuses    []string
	BlockedLabels      []string
	CriticalPriorities []string
	AgingDays          int // in progress for this long (or more) is aging
	StaleDays          int // in progress without an update for this long is stale
	RolloverSprints    int // part of this many sprints (or more) counts as rolled over
	LateStartPct       int // sprint items still to do after this share of the sprint
	DriftPct           int // sprint scope change that the page colors as a warning
	SprintCount        int // default number of closed sprints in the scope chart
	CacheDir           string
	CacheTTL           time.Duration
	Now                func() time.Time // defaults to time.Now
}

// MaxSprints bounds the sprint count a caller may request.
const MaxSprints = 24

type memEntry struct {
	report *Report
	at     time.Time
}

// Service builds RAID reports. Closed-sprint scope and the daily trend are
// persisted; the watchlist is read live and kept in memory for CacheTTL.
type Service struct {
	src  Source
	opt  Options
	disk diskCache

	mu    sync.Mutex
	locks map[string]*sync.Mutex
	mem   map[string]memEntry
}

// OptionsFromConfig maps the .env-driven configuration to service options.
func OptionsFromConfig(cfg *config.Config) Options {
	return Options{
		ProjectKey:         cfg.ProjectKey,
		IssueTypes:         cfg.JiraIssueTypes,
		BlockedStatuses:    cfg.RaidBlockedStatuses,
		BlockedLabels:      cfg.RaidBlockedLabels,
		CriticalPriorities: cfg.RaidCriticalPriorities,
		AgingDays:          cfg.RaidAgingDays,
		StaleDays:          cfg.RaidStaleDays,
		RolloverSprints:    cfg.RaidRolloverSprints,
		LateStartPct:       cfg.RaidLateStartPct,
		DriftPct:           cfg.RaidDriftPct,
		SprintCount:        cfg.DeliverySprintCount,
		CacheDir:           cfg.DeliveryCacheDir,
		CacheTTL:           cfg.DeliveryCacheTTL,
	}
}

// TeamsFromConfig lists the configured teams in .env order.
func TeamsFromConfig(cfg *config.Config) []Team {
	teams := make([]Team, len(cfg.Teams))
	for i, t := range cfg.Teams {
		teams[i] = Team{Component: t.ComponentName, Board: t.BoardName}
	}
	return teams
}

// New creates a Service, filling defaults for unset options.
func New(src Source, opt Options) *Service {
	def := func(v *int, d int) {
		if *v <= 0 {
			*v = d
		}
	}
	def(&opt.AgingDays, 10)
	def(&opt.StaleDays, 5)
	def(&opt.RolloverSprints, 3)
	def(&opt.LateStartPct, 60)
	def(&opt.DriftPct, 20)
	def(&opt.SprintCount, 6)
	if len(opt.BlockedStatuses) == 0 {
		opt.BlockedStatuses = []string{"Blocked", "On Hold"}
	}
	if len(opt.BlockedLabels) == 0 {
		opt.BlockedLabels = []string{"blocked"}
	}
	if len(opt.CriticalPriorities) == 0 {
		opt.CriticalPriorities = []string{"Blocker", "Critical", "Highest"}
	}
	if len(opt.IssueTypes) == 0 {
		opt.IssueTypes = []string{"Bug", "Story", "Task"}
	}
	if opt.Now == nil {
		opt.Now = time.Now
	}
	return &Service{src: src, opt: opt, disk: diskCache{dir: opt.CacheDir}, locks: map[string]*sync.Mutex{}, mem: map[string]memEntry{}}
}

// DriftPct is the scope change the page colors as a warning.
func (s *Service) DriftPct() int { return s.opt.DriftPct }

func (s *Service) teamLock(key string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, ok := s.locks[key]
	if !ok {
		l = &sync.Mutex{}
		s.locks[key] = l
	}
	return l
}

// Report returns the RAID data for the team with scope for its last `sprints`
// closed sprints plus the active one (<= 0 means the configured default).
// refresh bypasses the in-memory copy but still reuses persisted closed-sprint scope.
func (s *Service) Report(team Team, sprints int, refresh bool) *Report {
	if sprints <= 0 {
		sprints = s.opt.SprintCount
	}
	if sprints > MaxSprints {
		sprints = MaxSprints
	}
	key := fmt.Sprintf("%s|%d", team.Component, sprints)

	lock := s.teamLock(team.Component)
	lock.Lock()
	defer lock.Unlock()

	if !refresh {
		s.mu.Lock()
		e, ok := s.mem[key]
		s.mu.Unlock()
		if ok && s.opt.Now().Sub(e.at) < s.opt.CacheTTL {
			return e.report
		}
	}
	rep := s.build(team, sprints)
	s.mu.Lock()
	s.mem[key] = memEntry{report: rep, at: s.opt.Now()}
	s.mu.Unlock()
	return rep
}

func (s *Service) thresholds() Thresholds {
	return Thresholds{
		BlockedStatuses: s.opt.BlockedStatuses, BlockedLabels: s.opt.BlockedLabels, CriticalPriorities: s.opt.CriticalPriorities,
		AgingDays: s.opt.AgingDays, StaleDays: s.opt.StaleDays, RolloverSprints: s.opt.RolloverSprints,
		LateStartPct: s.opt.LateStartPct, DriftPct: s.opt.DriftPct,
	}
}

// build never fails as a whole: a part that cannot be read is left empty and
// explained in Errors (or Warnings when the rest stays correct).
func (s *Service) build(team Team, n int) *Report {
	now := s.opt.Now()
	rep := &Report{
		Team: team.Component, Board: team.Board, GeneratedAt: now, SprintCount: n,
		Thresholds: s.thresholds(), Errors: map[string]string{},
		Items: []Item{}, Scope: []ScopePoint{}, Trend: []Snapshot{},
		AgingBuckets: agingBuckets(nil, now),
	}
	cache := s.disk.load(team.Component)

	active, err := s.src.ActiveSprint(team.Board)
	if err != nil {
		msg := fmt.Sprintf("team %q: %v", team.Component, err)
		rep.Errors["scope"], rep.Errors["pulse"] = msg, msg
		active = nil
	}

	counts, watchOK := s.scanWatchlist(rep, team, active, now)
	s.fillScope(rep, team, active, n, cache)
	s.fillPulse(rep, active, counts, now)
	if watchOK {
		cache.record(Snapshot{
			Date: now.Format("2006-01-02"), Open: rep.Summary.Open, Flagged: rep.Summary.Flagged,
			Blocked: rep.Summary.BySignal[SigBlocked], Aging: rep.Summary.BySignal[SigAging],
		})
	}
	rep.Trend = append(rep.Trend, cache.Trend...)
	if err := s.disk.save(team.Component, cache); err != nil {
		log.Printf("raidscan: %v (results are served but not persisted)", err)
	}
	if len(rep.Errors) == 0 {
		rep.Errors = nil
	}
	return rep
}

type sprintCounts struct{ notStarted, inProgress int }

func (s *Service) jqlBase(team Team) string {
	return fmt.Sprintf(`project = %s AND component = "%s"`, s.opt.ProjectKey, team.Component)
}

// scanWatchlist reads the open work of the team, with its changelog, and fills
// Items, Summary and AgingBuckets. The whole open backlog is read because
// blocked work, dependencies and critical bugs matter outside the sprint too.
func (s *Service) scanWatchlist(rep *Report, team Team, active *jiraservice.SprintInfo, now time.Time) (sprintCounts, bool) {
	var counts sprintCounts
	sprintKnown := false // the running sprint exists and its issues were listed
	warn := func(format string, args ...any) { rep.Warnings = append(rep.Warnings, fmt.Sprintf(format, args...)) }

	r := &rules{opt: s.opt, now: now, blockedStatuses: lowerSet(s.opt.BlockedStatuses), blockedLabels: lowerSet(s.opt.BlockedLabels),
		critical: lowerSet(s.opt.CriticalPriorities), flagged: map[string]struct{}{}, inSprint: map[string]struct{}{}, sprintElapsedPct: -1}

	base := s.jqlBase(team)
	if active != nil {
		r.activeSprintID = active.ID
		if span := active.End.Sub(active.Start); span > 0 {
			r.sprintElapsedPct = math.Min(100, math.Max(0, float64(now.Sub(active.Start))/float64(span)*100))
		}
		keys, err := s.src.SearchKeys(fmt.Sprintf(`%s AND sprint = %d`, base, active.ID))
		if err != nil {
			warn("team %q: the issues of the active sprint could not be listed (%v); sprint-based signals are incomplete", team.Component, err)
		} else {
			r.inSprint = keys
			sprintKnown = true
		}
	}
	if keys, err := s.src.SearchKeys(base + ` AND statusCategory != Done AND flagged is not EMPTY`); err != nil {
		warn("team %q: flagged issues could not be read (%v); the Flagged impediment signal is skipped", team.Component, err)
	} else {
		r.flagged = keys
	}

	quoted := make([]string, len(s.opt.IssueTypes))
	for i, t := range s.opt.IssueTypes {
		quoted[i] = `"` + t + `"`
	}
	jql := fmt.Sprintf(`%s AND issuetype in (%s) AND statusCategory != Done`, base, strings.Join(quoted, ","))
	started := time.Now()
	issues, err := s.src.SearchRiskIssues(jql, true)
	if err != nil {
		msg := fmt.Sprintf("team %q: %v", team.Component, err)
		rep.Errors["watchlist"], rep.Errors["aging"] = msg, msg
		return counts, false
	}
	log.Printf("raidscan: team %q scanned %d open issues with changelog in %s", team.Component, len(issues), time.Since(started).Round(time.Millisecond))

	items := make([]Item, 0, len(issues))
	sprintOpen := 0
	for _, is := range issues {
		if _, in := r.inSprint[is.Key]; in {
			sprintOpen++
			switch strings.ToLower(is.StatusCategory) {
			case "new":
				counts.notStarted++
			case "indeterminate":
				counts.inProgress++
			}
		}
		if it := r.evaluate(is); len(it.Signals) > 0 {
			items = append(items, it)
		}
	}
	rank(items)
	rep.Items = items
	rep.Summary = summarize(len(issues), items)
	if sprintKnown {
		var inSprint []Item
		for _, it := range items {
			if it.InSprint {
				inSprint = append(inSprint, it)
			}
		}
		sum := summarize(sprintOpen, inSprint)
		rep.SprintSummary = &sum
	}
	rep.AgingBuckets = agingBuckets(issues, now)
	return counts, true
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

func scopePoint(id int, name string, active bool, sc *jiraservice.SprintScope) ScopePoint {
	p := ScopePoint{
		ID: id, Sprint: name, Active: active,
		Committed: sc.Committed, Added: sc.Added, Removed: sc.Removed, Reestimated: sc.Reestimated,
		Final: sc.Final, Completed: sc.Completed, CarriedOver: sc.CarriedOver,
		IssuesAdded: sc.IssuesAdded, IssuesRemoved: sc.IssuesRemoved, IssuesCarried: sc.IssuesCarried,
	}
	if sc.Committed > 0 {
		p.DriftPct = ptr(round1((sc.Final - sc.Committed) / sc.Committed * 100))
	}
	return p
}

// fillScope fills Scope with the last n closed sprints (from disk once known)
// followed by the active sprint (always live).
func (s *Service) fillScope(rep *Report, team Team, active *jiraservice.SprintInfo, n int, cache *cacheData) {
	if _, failed := rep.Errors["scope"]; failed {
		return
	}
	var firstErr error
	closed, err := s.src.ClosedSprints(team.Board, n)
	if err != nil {
		rep.Errors["scope"] = fmt.Sprintf("team %q (board %q): %v", team.Component, team.Board, err)
		return
	}
	for _, sp := range closed {
		if p, ok := cache.Scopes[sp.ID]; ok {
			rep.Scope = append(rep.Scope, p)
			continue
		}
		sc, err := s.src.SprintScope(team.Board, sp.ID)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("sprint %q: %w", sp.Name, err)
			}
			continue
		}
		p := scopePoint(sp.ID, sp.Name, false, sc)
		cache.Scopes[sp.ID] = p
		cache.dirty = true
		rep.Scope = append(rep.Scope, p)
	}
	if active != nil {
		sc, err := s.src.SprintScope(team.Board, active.ID)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("sprint %q: %w", active.Name, err)
			}
		} else {
			rep.Scope = append(rep.Scope, scopePoint(active.ID, active.Name, true, sc))
		}
	}
	if firstErr != nil {
		msg := fmt.Sprintf("team %q: %v", team.Component, firstErr)
		if len(rep.Scope) == 0 {
			rep.Errors["scope"] = msg
		} else {
			rep.Warnings = append(rep.Warnings, "some sprints are missing from the scope chart: "+msg)
		}
	}
}

func (s *Service) fillPulse(rep *Report, active *jiraservice.SprintInfo, counts sprintCounts, now time.Time) {
	if active == nil {
		return
	}
	if len(rep.Scope) == 0 || !rep.Scope[len(rep.Scope)-1].Active {
		if _, failed := rep.Errors["pulse"]; !failed {
			rep.Errors["pulse"] = fmt.Sprintf("the sprint report of %q could not be read", active.Name)
		}
		return
	}
	sc := rep.Scope[len(rep.Scope)-1]
	p := &SprintPulse{
		Name: active.Name, StartDate: active.Start, EndDate: active.End, Scope: sc,
		NotStarted: counts.notStarted, InProgress: counts.inProgress,
	}
	if span := active.End.Sub(active.Start); span > 0 {
		p.ElapsedPct = round1(math.Min(100, math.Max(0, float64(now.Sub(active.Start))/float64(span)*100)))
		p.DaysLeft = round1(math.Max(0, active.End.Sub(now).Hours()/24))
	}
	if sc.Final > 0 {
		p.DonePct = ptr(round1(sc.Completed / sc.Final * 100))
	}
	rep.Pulse = p
}
