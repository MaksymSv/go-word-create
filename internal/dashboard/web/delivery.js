'use strict';

const SPRINTS_KEY = 'delivery-dashboard-sprints';
const SLOW_LOAD_MS = 4000;
const { $, store, recall, api, showError, clearError, addCard, shortSprintNames } = Dash;

const state = {
  teams: [],
  settings: { sprintCount: 6, maxSprints: 24, bugDays: 30 },
  team: null,
  sprints: 6,
  loadToken: 0,
};

// Bug priorities keep one color across teams and sprints.
const PRIORITY_COLORS = {
  blocker: 'var(--s8)', critical: 'var(--s2)', major: 'var(--s5)', high: 'var(--s4)',
  medium: 'var(--s1)', low: 'var(--s3)', minor: 'var(--s7)',
  unprioritized: 'var(--s-neutral)', none: 'var(--s-neutral)',
};

const fmt = Charts.fmtNum;
const dayShort = iso => new Date(iso + 'T00:00:00').toLocaleDateString('en-GB', { day: 'numeric', month: 'short' });
const dayLong = iso => new Date(iso + 'T00:00:00').toLocaleDateString('en-GB', { weekday: 'short', day: 'numeric', month: 'short', year: 'numeric' });
const withUnit = unit => v => (v === null || v === undefined ? '—' : `${fmt(v)}${unit}`);

// ── Teams and sprint count ───────────────────────────────────────────────────

function selectTeam(component) {
  if (state.team === component) return;
  state.team = component;
  Dash.markActiveTeam(component);
  history.replaceState(null, '', '#' + encodeURIComponent(component));
  load(false);
}

function renderSprintCounts() {
  Dash.renderSprintCounts({
    defaultCount: state.settings.sprintCount,
    maxSprints: state.settings.maxSprints,
    current: state.sprints,
    onPick: n => {
      state.sprints = n;
      store(SPRINTS_KEY, String(n));
      renderSprintCounts();
      load(false);
    },
  });
}

// ── Loading ──────────────────────────────────────────────────────────────────

async function load(refresh) {
  const token = ++state.loadToken;
  const team = state.team;
  const refreshBtn = $('refresh');
  refreshBtn.disabled = true;
  clearError();
  $('meta').textContent = '';
  const cards = $('cards');
  cards.style.display = 'block';
  cards.innerHTML = '<div class="loading-state">Loading delivery metrics…</div>';
  const slow = setTimeout(() => {
    if (token === state.loadToken) {
      cards.firstChild.textContent = 'Reading issue history from Jira. The first load of a team can take up to a minute; later loads use the local cache.';
    }
  }, SLOW_LOAD_MS);

  try {
    const url = `/api/delivery/${encodeURIComponent(team)}?sprints=${state.sprints}${refresh ? '&refresh=1' : ''}`;
    const report = await api(url);
    if (token !== state.loadToken) return;
    renderReport(report);
  } catch (err) {
    if (token !== state.loadToken) return;
    showError(`Failed to load delivery metrics for "${team}": ${err.message}`);
    cards.textContent = '';
  } finally {
    clearTimeout(slow);
    if (token === state.loadToken) refreshBtn.disabled = false;
  }
}

// ── Report → cards ───────────────────────────────────────────────────────────

function renderReport(rep) {
  const names = rep.sprints.map(s => s.name);
  const range = names.length ? `${names[0]} – ${names[names.length - 1]}` : 'no closed sprints';
  const when = new Date(rep.generatedAt).toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' });
  $('meta').textContent = `${rep.team} · board “${rep.board}” · closed sprints ${range} · updated ${when}`;

  const cards = $('cards');
  cards.textContent = '';
  cards.style.display = '';
  if (!names.length) {
    cards.style.display = 'block';
    cards.innerHTML = '<div class="empty-state">This board has no closed sprints yet.</div>';
    return;
  }

  const errors = rep.errors || {};
  const sprintNames = shortSprintNames(names);
  const velocityGap = (rep.velocity || []).some(p => p.completed === null);

  addCard(cards, errors, {
    key: 'velocity',
    title: 'Velocity: committed vs. completed',
    sub: 'Story points per sprint, with the average of completed over the last 3 sprints',
    note: velocityGap ? 'Jira’s velocity chart only covers the last 7 sprints; older sprints appear here once they have been seen.' : '',
    spec: () => ({
      categories: sprintNames,
      titles: names,
      series: [
        { name: 'Committed', kind: 'bar', color: 'var(--s1)', values: rep.velocity.map(p => p.committed) },
        { name: 'Completed', kind: 'bar', color: 'var(--s2)', values: rep.velocity.map(p => p.completed) },
        { name: 'Avg last 3 (completed)', kind: 'line', dash: true, color: 'var(--color-text)', values: rep.velocity.map(p => p.avgLast3) },
      ],
    }),
  });

  addCard(cards, errors, {
    key: 'commitmentRate',
    title: 'Commitment rate',
    sub: 'Completed ÷ committed story points, in %',
    spec: () => ({
      categories: sprintNames,
      titles: names,
      yFmt: v => v + '%',
      refLine: { value: 100 },
      series: [{ name: 'Commitment rate', kind: 'bar', color: 'var(--s1)', fmt: withUnit('%'), labelFmt: v => Math.round(v) + '%', values: rep.commitmentRate.map(p => p.ratePct) }],
      extra: i => {
        const v = rep.velocity && rep.velocity[i];
        return v && v.committed !== null ? [['Committed', `${fmt(v.committed)} SP`], ['Completed', `${fmt(v.completed)} SP`]] : [];
      },
    }),
  });

  addCard(cards, errors, {
    key: 'backlogHealth',
    title: 'Backlog health',
    sub: 'Story points of open issues outside any sprint when the sprint started',
    spec: () => ({
      categories: sprintNames,
      titles: names,
      series: [{ name: 'Backlog (SP)', kind: 'bar', color: 'var(--s1)', values: rep.backlogHealth.map(p => p.storyPoints) }],
      extra: i => {
        const p = rep.backlogHealth[i];
        return [
          ['Issues', String(p.issues)],
          ['Without estimate', String(p.unestimated)],
          ['Runway', p.runwaySprints === null ? '—' : `${fmt(p.runwaySprints)} sprints`],
        ];
      },
    }),
  });

  addCard(cards, errors, {
    key: 'leadCycle',
    title: 'Lead and cycle time',
    sub: 'Average days for issues resolved in the sprint',
    spec: () => ({
      categories: sprintNames,
      titles: names,
      series: [
        { name: 'Lead time (created → resolved)', kind: 'bar', color: 'var(--s1)', fmt: withUnit(' d'), labelFmt: v => String(Math.round(v)), values: rep.leadCycle.map(p => p.leadDays) },
        { name: 'Cycle time (first In Progress → resolved)', kind: 'bar', color: 'var(--s2)', fmt: withUnit(' d'), labelFmt: v => String(Math.round(v)), values: rep.leadCycle.map(p => p.cycleDays) },
      ],
      extra: i => [['Issues resolved', String(rep.leadCycle[i].issues)], ['With In Progress date', String(rep.leadCycle[i].cycleIssues)]],
    }),
  });

  addCard(cards, errors, {
    key: 'openBugsDaily',
    title: `Open bugs, last ${rep.bugDays} days`,
    sub: 'Bugs not yet closed (status category not Done) at the end of each day',
    spec: () => ({
      categories: rep.openBugsDaily.map(p => dayShort(p.date)),
      titles: rep.openBugsDaily.map(p => dayLong(p.date)),
      series: [{ name: 'Open bugs', kind: 'line', color: 'var(--s8)', values: rep.openBugsDaily.map(p => p.open) }],
    }),
  });

  addCard(cards, errors, {
    key: 'bugDynamics',
    title: 'Bug dynamics',
    sub: 'Bugs logged vs. resolved per sprint',
    spec: () => ({
      categories: sprintNames,
      titles: names,
      series: [
        { name: 'Logged', kind: 'bar', color: 'var(--s2)', values: rep.bugDynamics.map(p => p.logged) },
        { name: 'Fixed', kind: 'bar', color: 'var(--s3)', values: rep.bugDynamics.map(p => p.fixed) },
      ],
      extra: i => [['Net', (rep.bugDynamics[i].logged - rep.bugDynamics[i].fixed > 0 ? '+' : '') + (rep.bugDynamics[i].logged - rep.bugDynamics[i].fixed)]],
    }),
  });

  addCard(cards, errors, {
    key: 'openBugsByPriority',
    title: 'Open bugs by priority',
    sub: 'Bugs not yet closed (status category not Done) at the end of each sprint',
    spec: () => {
      const by = rep.openBugsByPriority;
      return {
        categories: sprintNames,
      titles: names,
        endLabels: true,
        series: by.priorities.map(p => ({
          name: p,
          kind: 'line',
          color: PRIORITY_COLORS[p.toLowerCase()] || 'var(--s6)',
          values: by.points.map(pt => pt.counts[p] || 0),
        })),
        extra: i => [['Total open', String(Object.values(by.points[i].counts).reduce((a, b) => a + b, 0))]],
      };
    },
  });
}

// ── Bootstrap ────────────────────────────────────────────────────────────────

async function init() {
  Dash.initTheme();
  $('refresh').addEventListener('click', () => load(true));
  try {
    const [teams, settings] = await Promise.all([api('/api/teams'), api('/api/delivery/settings')]);
    state.teams = teams;
    state.settings = settings;
  } catch (err) {
    showError(`Failed to load dashboard configuration: ${err.message}`);
    return;
  }
  const saved = parseInt(recall(SPRINTS_KEY), 10);
  state.sprints = saved >= 1 && saved <= state.settings.maxSprints ? saved : state.settings.sprintCount;
  Dash.renderTeams(state.teams, selectTeam);
  renderSprintCounts();
  if (!state.teams.length) {
    $('cards').innerHTML = '<div class="empty-state">No teams are configured (TEAMS in .env).</div>';
    return;
  }
  const hash = decodeURIComponent(location.hash.replace(/^#/, ''));
  selectTeam(state.teams.some(t => t.componentName === hash) ? hash : state.teams[0].componentName);
}

document.addEventListener('DOMContentLoaded', init);
