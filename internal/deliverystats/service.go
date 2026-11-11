package deliverystats

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"go-word-create/internal/jiraservice"
)

// Source is the Jira access the service needs; *jiraservice.JiraService implements it.
type Source interface {
	ClosedSprints(boardName string, limit int) ([]jiraservice.SprintInfo, error)
	BoardVelocity(boardName string) (map[int]jiraservice.SprintVelocity, error)
	SearchHistoryIssues(jql string, withChangelog bool) ([]jiraservice.HistoryIssue, error)
}

// Team is one configured team: the Jira component filters issues, the board resolves sprints.
type Team struct {
	Component string
	Board     string
}

// Options configures a Service.
type Options struct {
	ProjectKey           string
	IssueTypes           []string // issue types counted for backlog and lead/cycle time
	CompletedResolutions []string // resolutions that mean "work was delivered"
	SprintCount          int      // default number of sprints shown
	BugDays              int      // length of the daily open-bugs series
	CacheDir             string   // closed-sprint results; empty disables the disk cache
	CacheTTL             time.Duration
	Now                  func() time.Time // defaults to time.Now
}

// MaxSprints bounds the sprint count a caller may request.
const MaxSprints = 24

// window is the interval [from, to) whose events count towards one sprint.
type window struct{ from, to time.Time }

// velocityWindow is the number of sprints in the trailing velocity average.
const velocityWindow = 3

type memEntry struct {
	report *Report
	at     time.Time
}

// Service builds delivery reports. Closed-sprint values that need a changelog
// scan are persisted; everything else is read live and kept in memory for CacheTTL.
type Service struct {
	src         Source
	opt         Options
	types       map[string]struct{}
	completed   map[string]struct{}
	disk        diskCache
	fingerprint string

	mu    sync.Mutex
	locks map[string]*sync.Mutex
	mem   map[string]memEntry
}

// New creates a Service, filling defaults for unset options.
func New(src Source, opt Options) *Service {
	if opt.SprintCount <= 0 {
		opt.SprintCount = 6
	}
	if opt.BugDays <= 0 {
		opt.BugDays = 30
	}
	if opt.Now == nil {
		opt.Now = time.Now
	}
	return &Service{
		src:         src,
		opt:         opt,
		types:       lowerSet(opt.IssueTypes),
		completed:   lowerSet(opt.CompletedResolutions),
		disk:        diskCache{dir: opt.CacheDir},
		fingerprint: fmt.Sprintf("v%d|%s|%s", cacheVersion, strings.ToLower(strings.Join(opt.IssueTypes, ",")), strings.ToLower(strings.Join(opt.CompletedResolutions, ","))),
		locks:       map[string]*sync.Mutex{},
		mem:         map[string]memEntry{},
	}
}

// DefaultSprintCount is the number of sprints shown when the caller does not choose.
func (s *Service) DefaultSprintCount() int { return s.opt.SprintCount }

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

// Report returns the dashboard data for the team over its last `sprints` closed
// sprints (<= 0 means the configured default). refresh bypasses the in-memory
// copy but still reuses persisted closed-sprint values.
func (s *Service) Report(team Team, sprints int, refresh bool) (*Report, error) {
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
			return e.report, nil
		}
	}

	rep, err := s.build(team, sprints)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.mem[key] = memEntry{report: rep, at: s.opt.Now()}
	s.mu.Unlock()
	return rep, nil
}

func (s *Service) build(team Team, n int) (*Report, error) {
	now := s.opt.Now()
	// Two extra sprints feed the trailing velocity average of the first shown ones.
	closed, err := s.src.ClosedSprints(team.Board, n+velocityWindow-1)
	if err != nil {
		return nil, fmt.Errorf("failed to list closed sprints for team %q (board %q): %w", team.Component, team.Board, err)
	}
	skip := 0
	if len(closed) > n {
		skip = len(closed) - n
	}
	shown := closed[skip:]
	windows := make([]window, len(shown))
	for i := range shown {
		windows[i].from, windows[i].to = sprintWindow(closed, skip+i)
	}

	rep := &Report{
		Team: team.Component, Board: team.Board, GeneratedAt: now,
		SprintCount: n, BugDays: s.opt.BugDays, Errors: map[string]string{},
		Sprints: make([]SprintRef, len(shown)),
	}
	for i, sp := range shown {
		rep.Sprints[i] = SprintRef{ID: sp.ID, Name: sp.Name, StartDate: sp.Start, EndDate: effectiveEnd(sp)}
	}

	cache := s.disk.load(team.Component, s.fingerprint)
	avgs := s.fillVelocity(rep, team, closed, skip, cache)
	s.fillBugs(rep, team, shown, windows, now, cache)
	s.fillDerived(rep, team, shown, windows, avgs, cache)
	if err := s.disk.save(team.Component, cache); err != nil {
		log.Printf("deliverystats: %v (results are served but not persisted)", err)
	}

	if len(rep.Errors) == 0 {
		rep.Errors = nil
	}
	return rep, nil
}

// fillVelocity fills Velocity and CommitmentRate and returns the trailing
// average per shown sprint (0 when unknown). Velocity read live is merged into
// the persisted history so sprints that left Jira's 7-sprint window stay known.
func (s *Service) fillVelocity(rep *Report, team Team, closed []jiraservice.SprintInfo, skip int, cache *cacheData) []float64 {
	live, err := s.src.BoardVelocity(team.Board)
	if err != nil {
		if len(cache.Velocity) == 0 {
			rep.Errors["velocity"] = fmt.Sprintf("team %q: %v", team.Component, err)
			rep.Errors["commitmentRate"] = rep.Errors["velocity"]
			return nil
		}
		log.Printf("deliverystats: team %q: %v (using stored velocity history)", team.Component, err)
	}
	for _, sp := range closed {
		if v, ok := live[sp.ID]; ok {
			if old, seen := cache.Velocity[sp.ID]; !seen || old.Committed != v.Committed || old.Completed != v.Completed {
				cache.Velocity[sp.ID] = sprintVelocity{Committed: v.Committed, Completed: v.Completed}
				cache.dirty = true
			}
		}
	}

	// Trailing average over the known completed values among the last three sprints.
	avgAt := func(i int) (float64, bool) {
		sum, n := 0.0, 0
		for j := i; j > i-velocityWindow && j >= 0; j-- {
			if v, ok := cache.Velocity[closed[j].ID]; ok {
				sum += v.Completed
				n++
			}
		}
		if n == 0 {
			return 0, false
		}
		return sum / float64(n), true
	}

	avgs := make([]float64, 0, len(closed)-skip)
	for i := skip; i < len(closed); i++ {
		sp := closed[i]
		vp := VelocityPoint{Sprint: sp.Name}
		cp := CommitmentPoint{Sprint: sp.Name}
		avg, hasAvg := avgAt(i)
		avgs = append(avgs, avg)
		if hasAvg {
			vp.AvgLast3 = ptr(round1(avg))
		}
		if v, ok := cache.Velocity[sp.ID]; ok {
			vp.Committed, vp.Completed = ptr(v.Committed), ptr(v.Completed)
			if v.Committed > 0 {
				cp.RatePct = ptr(round1(v.Completed / v.Committed * 100))
			}
		}
		rep.Velocity = append(rep.Velocity, vp)
		rep.CommitmentRate = append(rep.CommitmentRate, cp)
	}
	return avgs
}

func (s *Service) fillBugs(rep *Report, team Team, shown []jiraservice.SprintInfo, wins []window, now time.Time, cache *cacheData) {
	jql := fmt.Sprintf(`project = %s AND component = "%s" AND issuetype = Bug`, s.opt.ProjectKey, team.Component)
	bugs, err := s.src.SearchHistoryIssues(jql, false)
	if err == nil {
		err = s.settleResolution(bugs, cache)
	}
	if err != nil {
		msg := fmt.Sprintf("team %q: %v", team.Component, err)
		rep.Errors["openBugsDaily"], rep.Errors["bugDynamics"], rep.Errors["openBugsByPriority"] = msg, msg, msg
		return
	}

	for i := s.opt.BugDays - 1; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		at := endOfDay(day)
		if at.After(now) {
			at = now
		}
		rep.OpenBugsDaily = append(rep.OpenBugsDaily, DayPoint{Date: day.Format("2006-01-02"), Open: countOpen(bugs, at)})
	}

	byPrio := &OpenBugsByPriority{}
	seen := map[string]struct{}{}
	for i, sp := range shown {
		logged, fixed := loggedFixed(bugs, s.completed, wins[i].from, wins[i].to)
		rep.BugDynamics = append(rep.BugDynamics, BugDynamicsPoint{Sprint: sp.Name, Logged: logged, Fixed: fixed})
		counts := openByPriority(bugs, wins[i].to)
		byPrio.Points = append(byPrio.Points, PriorityPoint{Sprint: sp.Name, Counts: counts})
		for p := range counts {
			seen[p] = struct{}{}
		}
	}
	names := make([]string, 0, len(seen))
	for p := range seen {
		names = append(names, p)
	}
	byPrio.Priorities = sortPriorities(names)
	rep.OpenBugsByPriority = byPrio
}

// fillDerived fills BacklogHealth and LeadCycle, scanning issue changelogs
// only for closed sprints that are not in the disk cache yet.
func (s *Service) fillDerived(rep *Report, team Team, shown []jiraservice.SprintInfo, wins []window, avgs []float64, cache *cacheData) {
	cached := cache.Sprints
	var missing []int // indexes into shown
	for i, sp := range shown {
		if d, ok := cached[sp.ID]; !ok || !d.fresh(sp.Start, wins[i].from, wins[i].to) {
			missing = append(missing, i)
		}
	}

	if len(missing) > 0 {
		computed, err := s.computeDerived(team, shown, wins, missing, cache)
		if err != nil {
			msg := fmt.Sprintf("team %q: %v", team.Component, err)
			rep.Errors["backlogHealth"], rep.Errors["leadCycle"] = msg, msg
			return
		}
		for id, d := range computed {
			cached[id] = d
		}
		cache.dirty = true
	}

	for i, sp := range shown {
		d := cached[sp.ID]
		bp := d.Backlog
		bp.Sprint = sp.Name
		if avgs != nil && avgs[i] > 0 {
			bp.RunwaySprints = ptr(round1(bp.StoryPoints / avgs[i]))
		}
		lc := d.LeadCycle
		lc.Sprint = sp.Name
		rep.BacklogHealth = append(rep.BacklogHealth, bp)
		rep.LeadCycle = append(rep.LeadCycle, lc)
	}
}

func (s *Service) computeDerived(team Team, shown []jiraservice.SprintInfo, wins []window, missing []int, cache *cacheData) (map[int]sprintDerived, error) {
	oldest := shown[missing[0]].Start
	for _, i := range missing {
		if shown[i].Start.Before(oldest) {
			oldest = shown[i].Start
		}
	}
	quoted := make([]string, len(s.opt.IssueTypes))
	for i, t := range s.opt.IssueTypes {
		quoted[i] = `"` + t + `"`
	}
	// An issue open at any sprint start >= oldest is not Done now, or finished after it.
	// Issues closed without a resolution date are caught by `updated`; one day of slack
	// absorbs the date-only JQL comparison and time zones.
	since := oldest.AddDate(0, 0, -1).Format("2006-01-02")
	jql := fmt.Sprintf(`project = %s AND component = "%s" AND issuetype in (%s) AND (statusCategory != Done OR resolutiondate >= "%s" OR updated >= "%s")`,
		s.opt.ProjectKey, team.Component, strings.Join(quoted, ","), since, since)
	started := time.Now()
	issues, err := s.src.SearchHistoryIssues(jql, true)
	if err != nil {
		return nil, fmt.Errorf("failed to load issue history for %d sprint(s) starting %s: %w", len(missing), oldest.Format("2006-01-02"), err)
	}
	if err := s.settleResolution(issues, cache); err != nil {
		return nil, err
	}
	log.Printf("deliverystats: team %q scanned %d issues with changelog for %d sprint(s) in %s", team.Component, len(issues), len(missing), time.Since(started).Round(time.Millisecond))

	out := make(map[int]sprintDerived, len(missing))
	for _, i := range missing {
		sp := shown[i]
		spPts, n, unest := backlogAt(issues, s.types, sp.Start)
		lead, cycle, leadN, cycleN := leadCycle(issues, s.types, s.completed, wins[i].from, wins[i].to)
		out[sp.ID] = sprintDerived{
			Start: sp.Start, From: wins[i].from, To: wins[i].to, ComputedAt: s.opt.Now(),
			Backlog:   BacklogPoint{StoryPoints: spPts, Issues: n, Unestimated: unest},
			LeadCycle: LeadCyclePoint{LeadDays: lead, CycleDays: cycle, Issues: leadN, CycleIssues: cycleN},
		}
	}
	return out, nil
}
