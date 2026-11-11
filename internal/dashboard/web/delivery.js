'use strict';

const THEME_KEY = 'sprint-dashboard-theme';
const SPRINTS_KEY = 'delivery-dashboard-sprints';
const SLOW_LOAD_MS = 4000;

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

const $ = id => document.getElementById(id);

function store(key, value) {
  try { localStorage.setItem(key, value); } catch (e) { /* storage may be unavailable */ }
}
function recall(key) {
  try { return localStorage.getItem(key); } catch (e) { return null; }
}

// ── Theme ────────────────────────────────────────────────────────────────────

function applyTheme(theme) {
  document.documentElement.setAttribute('data-theme', theme);
  $('theme-toggle').textContent = theme === 'dark' ? '⚪' : '☾';
}

function initTheme() {
  applyTheme(recall(THEME_KEY) || 'light');
  $('theme-toggle').addEventListener('click', () => {
    const next = document.documentElement.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
    applyTheme(next);
    store(THEME_KEY, next);
  });
}

// ── Helpers ──────────────────────────────────────────────────────────────────

function showError(msg) {
  const banner = $('error-banner');
  banner.textContent = msg;
  banner.removeAttribute('hidden');
}
function clearError() { $('error-banner').setAttribute('hidden', ''); }

async function api(path) {
  const res = await fetch(path);
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `HTTP ${res.status}`);
  }
  return res.json();
}

const fmt = Charts.fmtNum;
const dayShort = iso => new Date(iso + 'T00:00:00').toLocaleDateString('en-GB', { day: 'numeric', month: 'short' });
const dayLong = iso => new Date(iso + 'T00:00:00').toLocaleDateString('en-GB', { weekday: 'short', day: 'numeric', month: 'short', year: 'numeric' });
// "PC26.17" -> "26.17" when every sprint shares the same letter prefix (full names stay in tooltips).
function shortSprintNames(names) {
  const prefix = names.map(n => (n.match(/^[A-Za-z]+/) || [''])[0]);
  return prefix[0] && prefix.every(p => p === prefix[0]) ? names.map(n => n.slice(prefix[0].length)) : names;
}
const withUnit = unit => v => (v === null || v === undefined ? '—' : `${fmt(v)}${unit}`);

// ── Teams (tabs) ─────────────────────────────────────────────────────────────

function renderTeams() {
  const section = $('team-section');
  section.textContent = '';
  state.teams.forEach(team => {
    const btn = document.createElement('button');
    btn.className = 'btn-team';
    btn.textContent = team.componentName;
    btn.dataset.component = team.componentName;
    btn.setAttribute('role', 'tab');
    btn.addEventListener('click', () => selectTeam(team.componentName));
    section.appendChild(btn);
  });
}

function markActiveTeam() {
  document.querySelectorAll('#team-section .btn-team').forEach(b => {
    const active = b.dataset.component === state.team;
    b.classList.toggle('active', active);
    b.setAttribute('aria-selected', active ? 'true' : 'false');
  });
}

function selectTeam(component) {
  if (state.team === component) return;
  state.team = component;
  markActiveTeam();
  history.replaceState(null, '', '#' + encodeURIComponent(component));
  load(false);
}

// ── Sprint count ─────────────────────────────────────────────────────────────

function renderSprintCounts() {
  const section = $('sprint-count-section');
  section.textContent = '';
  const label = document.createElement('span');
  label.textContent = 'Sprints:';
  section.appendChild(label);
  const options = new Set([3, 6, 9, 12, state.settings.sprintCount, state.sprints]);
  [...options].filter(n => n >= 1 && n <= state.settings.maxSprints).sort((a, b) => a - b).forEach(n => {
    const pill = document.createElement('button');
    pill.className = 'pill' + (n === state.sprints ? ' active' : '');
    pill.textContent = n;
    pill.dataset.count = n;
    pill.addEventListener('click', () => {
      if (state.sprints === n) return;
      state.sprints = n;
      store(SPRINTS_KEY, String(n));
      renderSprintCounts();
      load(false);
    });
    section.appendChild(pill);
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

function addCard(parent, errors, def) {
  const card = document.createElement('section');
  card.className = 'card';

  const head = document.createElement('div');
  head.className = 'card-head';
  const titles = document.createElement('div');
  const h2 = document.createElement('h2');
  h2.textContent = def.title;
  const sub = document.createElement('div');
  sub.className = 'card-sub';
  sub.textContent = def.sub;
  titles.append(h2, sub);
  head.appendChild(titles);
  card.appendChild(head);

  const body = document.createElement('div');
  card.appendChild(body);
  parent.appendChild(card);

  if (errors[def.key]) {
    const err = document.createElement('div');
    err.className = 'card-error';
    err.textContent = `Could not compute this chart: ${errors[def.key]}`;
    body.appendChild(err);
    return;
  }

  const spec = def.spec();
  spec.ariaLabel = `${def.title}. ${def.sub}.`;
  spec.title = def.title;
  const chart = Charts.render(body, spec);
  if (chart) {
    const toggle = document.createElement('button');
    toggle.className = 'card-toggle';
    toggle.textContent = 'Table';
    toggle.addEventListener('click', () => { toggle.textContent = chart.toggleTable() ? 'Chart' : 'Table'; });
    head.appendChild(toggle);
  }
  if (def.note) {
    const note = document.createElement('div');
    note.className = 'card-note';
    note.textContent = def.note;
    card.appendChild(note);
  }
}

// ── Bootstrap ────────────────────────────────────────────────────────────────

async function init() {
  initTheme();
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
  renderTeams();
  renderSprintCounts();
  if (!state.teams.length) {
    $('cards').innerHTML = '<div class="empty-state">No teams are configured (TEAMS in .env).</div>';
    return;
  }
  const hash = decodeURIComponent(location.hash.replace(/^#/, ''));
  selectTeam(state.teams.some(t => t.componentName === hash) ? hash : state.teams[0].componentName);
}

document.addEventListener('DOMContentLoaded', init);
