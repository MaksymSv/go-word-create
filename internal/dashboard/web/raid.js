'use strict';

const VIEW_KEY = 'raid-dashboard-view';
const SLOW_LOAD_MS = 4000;
const PAGE_SIZE = 15;
const { $, store, recall, api, showError, clearError, addCard, shortSprintNames } = Dash;

const state = {
  teams: [],
  settings: { driftPct: 20, jiraUrl: '' },
  view: recall(VIEW_KEY) === 'sprint' ? 'sprint' : 'all', // 'all' open work or the running 'sprint' only
  team: null,
  loadToken: 0,
  report: null,
  filter: { cat: null, signal: null },
  showAll: false,
};

const CATEGORIES = [
  { code: 'R', name: 'Risks', hint: 'Aging, stale, late-start, rolled-over and overdue work' },
  { code: 'A', name: 'Assumptions', hint: 'Unestimated sprint work, in-progress work nobody owns' },
  { code: 'I', name: 'Issues', hint: 'Blocked work and critical bugs' },
  { code: 'D', name: 'Dependencies', hint: 'Work waiting on unfinished linked issues' },
];

const SIGNALS = [
  { code: 'blocked', cat: 'I', name: 'Blocked' },
  { code: 'critical-bug', cat: 'I', name: 'Critical bug' },
  { code: 'dependency', cat: 'D', name: 'Waiting on' },
  { code: 'aging', cat: 'R', name: 'Aging' },
  { code: 'stale', cat: 'R', name: 'Stale' },
  { code: 'late-start', cat: 'R', name: 'Late start' },
  { code: 'rolled-over', cat: 'R', name: 'Rolled over' },
  { code: 'overdue', cat: 'R', name: 'Overdue' },
  { code: 'unestimated', cat: 'A', name: 'No estimate' },
  { code: 'unassigned', cat: 'A', name: 'No assignee' },
];

const SEVERITY_MARK = { high: '▲', medium: '●', low: '○' };
const AGING_COLORS = ['var(--s3)', 'var(--s3)', 'var(--s4)', 'var(--s2)', 'var(--s8)'];

const fmt = Charts.fmtNum;
const withUnit = unit => v => (v === null || v === undefined ? '—' : `${fmt(v)}${unit}`);
const signed = v => (v > 0 ? '+' : v < 0 ? '−' : '') + fmt(Math.abs(v));
const plural = (n, one, many) => `${n} ${n === 1 ? one : many}`;

function el(tag, cls, text) {
  const node = document.createElement(tag);
  if (cls) node.className = cls;
  if (text !== undefined) node.textContent = text;
  return node;
}

// ── Teams and sprint count ───────────────────────────────────────────────────

function selectTeam(component) {
  if (state.team === component) return;
  state.team = component;
  Dash.markActiveTeam(component);
  history.replaceState(null, '', '#' + encodeURIComponent(component));
  load(false);
}

// ── Loading ──────────────────────────────────────────────────────────────────

async function load(refresh) {
  const token = ++state.loadToken;
  const team = state.team;
  const refreshBtn = $('refresh');
  refreshBtn.disabled = true;
  clearError();
  $('meta').textContent = '';
  $('warnings').textContent = '';
  $('tiles').textContent = '';
  $('view-switch').textContent = '';
  $('rules').hidden = true;
  state.filter = { cat: null, signal: null };
  state.showAll = false;
  const cards = $('cards');
  cards.style.display = 'block';
  cards.innerHTML = '<div class="loading-state">Scanning Jira for risks…</div>';
  const slow = setTimeout(() => {
    if (token === state.loadToken) {
      cards.firstChild.textContent = 'Reading the open backlog and its history from Jira. The first load of a team can take up to a minute; later loads reuse the stored sprint history.';
    }
  }, SLOW_LOAD_MS);

  try {
    const report = await api(`/api/raid/${encodeURIComponent(team)}${refresh ? '?refresh=1' : ''}`);
    if (token !== state.loadToken) return;
    state.report = report;
    renderReport(report);
  } catch (err) {
    if (token !== state.loadToken) return;
    showError(`Failed to scan "${team}": ${err.message}`);
    cards.textContent = '';
  } finally {
    clearTimeout(slow);
    if (token === state.loadToken) refreshBtn.disabled = false;
  }
}

// ── Report ───────────────────────────────────────────────────────────────────

// What the tiles and the watchlist show: all open work, or only the running
// sprint's issues when that view is chosen and the report has one.
function scan(rep) {
  if (state.view === 'sprint' && rep.sprintSummary) {
    return { sprint: true, summary: rep.sprintSummary, items: rep.items.filter(it => it.inSprint) };
  }
  return { sprint: false, summary: rep.summary, items: rep.items };
}

function sprintName(rep) {
  const running = (rep.scope || []).find(p => p.active);
  return running ? running.sprint : 'running sprint';
}

function renderMeta(rep) {
  const when = new Date(rep.generatedAt).toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' });
  const { sprint, summary } = scan(rep);
  const where = sprint ? `in sprint ${sprintName(rep)}` : 'in total';
  $('meta').textContent = `${rep.team} · board “${rep.board}” · ${summary.flagged} of ${plural(summary.open, 'open issue', 'open issues')} ${where} need attention · updated ${when}`;
}

// Switch between all open work and the running sprint; it filters the tiles and the watchlist.
function renderSwitch(rep) {
  const box = $('view-switch');
  box.textContent = '';
  if (rep.errors && rep.errors.watchlist) return;
  box.appendChild(el('span', null, 'Show:'));
  const option = (view, label, summary) => {
    const pill = el('button', 'pill' + (scan(rep).sprint === (view === 'sprint') ? ' active' : ''), summary ? `${label} · ${summary.flagged} flagged` : label);
    pill.type = 'button';
    pill.setAttribute('aria-pressed', String(pill.classList.contains('active')));
    if (!summary) {
      pill.disabled = true;
      pill.title = 'No sprint is running, or its issues could not be listed';
    } else {
      pill.addEventListener('click', () => {
        state.view = view;
        store(VIEW_KEY, view);
        state.filter = { cat: null, signal: null };
        state.showAll = false;
        renderMeta(rep);
        renderSwitch(rep);
        renderTiles(rep, rep.errors || {});
        renderWatchlist();
      });
    }
    box.appendChild(pill);
  };
  option('all', 'All open work', rep.summary);
  option('sprint', `Current sprint ${sprintName(rep)}`, rep.sprintSummary);
}

function renderReport(rep) {
  renderMeta(rep);
  renderSwitch(rep);

  const warnings = $('warnings');
  warnings.textContent = '';
  (rep.warnings || []).forEach(w => warnings.appendChild(el('div', 'warning-note', w)));
  renderRules(rep.thresholds);

  const errors = rep.errors || {};
  renderTiles(rep, errors);

  const cards = $('cards');
  cards.textContent = '';
  cards.style.display = '';

  if (rep.pulse || errors.pulse) {
    addCard(cards, errors, {
      key: 'pulse', wide: true,
      title: rep.pulse ? `Sprint pulse: ${rep.pulse.name}` : 'Sprint pulse',
      sub: 'Progress against the clock in the running sprint',
      build: body => renderPulse(body, rep.pulse),
    });
  }

  addCard(cards, errors, {
    key: 'watchlist', wide: true,
    title: 'Watchlist',
    sub: 'Open issues with at least one signal, most severe first. Hover a chip for the reason.',
    build: body => { watchBody = body; renderWatchlist(); },
  });

  addCard(cards, errors, {
    key: 'aging',
    title: 'Work in progress by age',
    sub: `In-progress issues by days since work started (aging limit: ${rep.thresholds.agingDays} days)`,
    spec: () => ({
      categories: rep.agingBuckets.map(b => b.label),
      emptyText: 'Nothing is in progress.',
      series: [{ name: 'In-progress issues', kind: 'bar', color: 'var(--s1)', colors: AGING_COLORS, values: rep.agingBuckets.map(b => b.count) }],
    }),
  });

  const scope = rep.scope || [];
  const names = scope.map(p => p.sprint);
  const cats = shortSprintNames(names).map((n, i) => (scope[i].active ? `${n}*` : n));
  const titles = scope.map(p => (p.active ? `${p.sprint} (running)` : p.sprint));
  const scopeExtra = i => {
    const p = scope[i];
    return [
      ['Committed at start', `${fmt(p.committed)} SP`],
      ['Re-estimated', `${signed(p.reestimated)} SP`],
      ['Scope now', `${fmt(p.final)} SP`],
      ['Not completed', `${fmt(p.carriedOver)} SP in ${plural(p.issuesCarried, 'issue', 'issues')}`],
    ];
  };

  addCard(cards, errors, {
    key: 'scope',
    title: 'Scope change per sprint',
    sub: 'Story points added to and removed from a sprint after it started. * marks the running sprint.',
    spec: () => ({
      categories: cats, titles,
      series: [
        { name: 'Added', kind: 'bar', color: 'var(--s2)', values: scope.map(p => p.added) },
        { name: 'Removed', kind: 'bar', color: 'var(--s1)', values: scope.map(p => p.removed) },
      ],
      extra: scopeExtra,
    }),
  });

  const limit = rep.thresholds.driftPct;
  const driftColor = v => (v === null ? 'var(--s1)' : Math.abs(v) >= limit ? 'var(--s8)' : Math.abs(v) >= limit / 2 ? 'var(--s4)' : 'var(--s1)');
  addCard(cards, errors, {
    key: 'scope',
    title: 'Scope drift',
    sub: 'Scope now vs. committed at sprint start, in %',
    note: `Bars turn amber at half the limit and red at ${limit}% (RAID_SCOPE_DRIFT_PCT). Adding work mid-sprint and re-estimating both count.`,
    spec: () => ({
      categories: cats, titles,
      yFmt: v => v + '%',
      refLine: { value: limit },
      series: [{
        name: 'Scope drift', kind: 'bar', color: 'var(--s1)', colors: scope.map(p => driftColor(p.driftPct)),
        fmt: withUnit('%'), labelFmt: v => Math.round(v) + '%', values: scope.map(p => p.driftPct),
      }],
      extra: i => [['Committed → now', `${fmt(scope[i].committed)} → ${fmt(scope[i].final)} SP`], ...scopeExtra(i).slice(1, 2)],
    }),
  });

  const trend = rep.trend || [];
  const day = iso => new Date(iso + 'T00:00:00').toLocaleDateString('en-GB', { day: 'numeric', month: 'short' });
  const trendCard = {
    key: 'trend',
    title: 'Watchlist trend',
    sub: 'Daily snapshot of the number of flagged, blocked and aging issues',
  };
  if (trend.length < 2) {
    addCard(cards, errors, {
      ...trendCard,
      build: body => body.appendChild(el('div', 'chart-empty', 'The trend starts today. Jira cannot tell what was blocked last week, so one snapshot is stored per day and the chart fills in from the second day.')),
    });
  } else {
    addCard(cards, errors, {
      ...trendCard,
      spec: () => ({
        categories: trend.map(p => day(p.date)),
        titles: trend.map(p => day(p.date)),
        series: [
          { name: 'Flagged', kind: 'line', color: 'var(--s1)', values: trend.map(p => p.flagged) },
          { name: 'Blocked', kind: 'line', color: 'var(--s8)', values: trend.map(p => p.blocked) },
          { name: 'Aging', kind: 'line', color: 'var(--s4)', values: trend.map(p => p.aging) },
        ],
        extra: i => [['Open issues', String(trend[i].open)]],
      }),
    });
  }
}

function renderRules(t) {
  const list = $('rules-list');
  list.textContent = '';
  [
    `Blocked: status ${t.blockedStatuses.join(' / ')}, label ${t.blockedLabels.join(' / ')}, or flagged as an impediment in Jira.`,
    `Critical bug: an open bug with priority ${t.criticalPriorities.join(' / ')}.`,
    `Waiting on: an unfinished issue linked with “is blocked by” or “depends on”.`,
    `Aging: in progress for ${t.agingDays} days or more (high at ${2 * t.agingDays}). Stale: in progress without an update for ${t.staleDays} days.`,
    `Late start: still to do in the running sprint once ${t.lateStartPct}% of it has passed.`,
    `Rolled over: in a sprint or in progress after ${t.rolloverSprints} sprints. Overdue: past its due date.`,
    `Assumptions: sprint work (not bugs) without story points, and in-progress work without an assignee.`,
    `Scope drift: story points added, removed or re-estimated after the sprint started, as in Jira’s sprint report.`,
  ].forEach(line => list.appendChild(el('li', null, line)));
  $('rules').hidden = false;
}

// ── Tiles ────────────────────────────────────────────────────────────────────

const STATUS_WORDS = { bad: 'Needs action', warn: 'Watch', ok: 'Clear' };

function renderTiles(rep, errors) {
  const tiles = $('tiles');
  tiles.textContent = '';
  if (errors.watchlist) {
    const err = el('div', 'card-error', `Could not scan the backlog: ${errors.watchlist}`);
    err.style.gridColumn = '1 / -1';
    tiles.appendChild(err);
  } else {
    const { summary } = scan(rep);
    CATEGORIES.forEach(c => {
      const total = summary.byCategory[c.code] || 0;
      const high = summary.highByCategory[c.code] || 0;
      const status = high > 0 ? 'bad' : total > 0 ? 'warn' : 'ok';
      const btn = el('button', `tile ${status}`);
      btn.type = 'button';
      btn.dataset.cat = c.code;
      btn.setAttribute('aria-pressed', 'false');
      btn.append(
        el('span', 'tile-label', c.name),
        el('span', 'tile-value', String(total)),
        el('span', 'tile-status', high > 0 ? `${STATUS_WORDS[status]} · ${high} high` : STATUS_WORDS[status]),
        el('span', 'tile-hint', c.hint),
      );
      btn.addEventListener('click', () => {
        state.filter = { cat: state.filter.cat === c.code ? null : c.code, signal: null };
        state.showAll = false;
        syncTiles();
        renderWatchlist();
      });
      tiles.appendChild(btn);
    });
  }
  tiles.appendChild(driftTile(rep, errors));
}

function syncTiles() {
  document.querySelectorAll('#tiles button.tile').forEach(b => b.setAttribute('aria-pressed', String(b.dataset.cat === state.filter.cat)));
}

// The drift tile shows the running sprint, or the latest closed one between sprints.
function driftTile(rep, errors) {
  const tile = el('div', 'tile ok');
  tile.appendChild(el('span', 'tile-label', 'Scope drift'));
  const scope = rep.scope || [];
  const p = scope[scope.length - 1];
  if (errors.scope || !p || p.driftPct === null) {
    tile.append(el('span', 'tile-value', '—'), el('span', 'tile-hint', errors.scope ? 'Sprint report unavailable' : 'No committed scope to compare'));
    return tile;
  }
  const limit = rep.thresholds.driftPct;
  const a = Math.abs(p.driftPct);
  tile.className = `tile ${a >= limit ? 'bad' : a >= limit / 2 ? 'warn' : 'ok'}`;
  tile.append(
    el('span', 'tile-value', `${p.driftPct > 0 ? '+' : p.driftPct < 0 ? '−' : ''}${fmt(a)}%`),
    el('span', 'tile-status', STATUS_WORDS[a >= limit ? 'bad' : a >= limit / 2 ? 'warn' : 'ok']),
    el('span', 'tile-hint', `${p.sprint}${p.active ? ' (running)' : ''}: ${fmt(p.committed)} → ${fmt(p.final)} SP, limit ${limit}%`),
  );
  return tile;
}

// ── Sprint pulse ─────────────────────────────────────────────────────────────

function renderPulse(body, p) {
  const wrap = el('div', 'pulse');
  const totalDays = Math.max(1, Math.round((new Date(p.endDate) - new Date(p.startDate)) / 86400000));
  const day = Math.min(totalDays, Math.floor(p.elapsedPct / 100 * totalDays) + 1);

  const row = (label, pct, cls, value, marker) => {
    const r = el('div', 'pulse-row');
    const track = el('div', 'track');
    const fill = el('div', `fill ${cls}`);
    fill.style.width = `${Math.min(100, Math.max(0, pct))}%`;
    track.appendChild(fill);
    if (marker !== undefined) {
      const m = el('div', 'mark');
      m.style.left = `${Math.min(100, marker)}%`;
      m.title = `Time elapsed: ${fmt(marker)}%`;
      track.appendChild(m);
    }
    r.append(el('span', 'pulse-label', label), track, el('span', 'pulse-val', value));
    return r;
  };
  wrap.appendChild(row('Time elapsed', p.elapsedPct, 'time', `Day ${day} of ${totalDays} · ${fmt(p.daysLeft)} days left`));
  const done = p.donePct === null ? 0 : p.donePct;
  wrap.appendChild(row('Scope completed', done, 'done', `${fmt(done)}% of ${fmt(p.scope.final)} SP`, p.elapsedPct));

  const gap = Math.round(done - p.elapsedPct);
  wrap.appendChild(el('div', 'pulse-gap',
    gap < 0 ? `${-gap} points behind the clock` : gap > 0 ? `${gap} points ahead of the clock` : 'On pace with the clock'));

  const facts = el('div', 'pulse-facts');
  const fact = (label, value) => {
    const s = el('span');
    s.append(`${label} `, el('b', null, value));
    facts.appendChild(s);
  };
  fact('Committed', `${fmt(p.scope.committed)} SP`);
  fact('Added', `${fmt(p.scope.added)} SP`);
  fact('Removed', `${fmt(p.scope.removed)} SP`);
  fact('Re-estimated', `${signed(p.scope.reestimated)} SP`);
  fact('Not started', String(p.notStarted));
  fact('In progress', String(p.inProgress));
  wrap.appendChild(facts);
  body.appendChild(wrap);
}

// ── Watchlist ────────────────────────────────────────────────────────────────

let watchBody = null;

function matches(item) {
  const { cat, signal } = state.filter;
  if (signal) return item.signals.some(s => s.code === signal);
  if (cat) return item.signals.some(s => s.category === cat);
  return true;
}

function chip(signal) {
  const c = el('span', `chip sev-${signal.severity}`);
  c.title = signal.detail;
  c.append(el('span', 'chip-cat', signal.category), el('span', 'sev-mark', SEVERITY_MARK[signal.severity]), signal.label);
  return c;
}

function renderWatchlist() {
  const rep = state.report;
  const body = watchBody;
  if (!rep || !body) return;
  body.textContent = '';

  const { sprint, summary, items } = scan(rep);
  if (!items.length) {
    body.appendChild(el('div', 'chart-empty', sprint
      ? `Nothing in sprint ${sprintName(rep)} is blocked, aging or at risk.`
      : 'Nothing on the watchlist. No open issue of this team is blocked, aging or at risk.'));
    return;
  }

  const filters = el('div', 'filters');
  const add = (label, active, onClick) => {
    const b = el('button', 'filter-chip' + (active ? ' active' : ''), label);
    b.type = 'button';
    b.setAttribute('aria-pressed', String(active));
    b.addEventListener('click', () => { onClick(); state.showAll = false; syncTiles(); renderWatchlist(); });
    filters.appendChild(b);
  };
  add(`All (${items.length})`, !state.filter.cat && !state.filter.signal, () => { state.filter = { cat: null, signal: null }; });
  SIGNALS.filter(s => (summary.bySignal[s.code] || 0) > 0 && (!state.filter.cat || s.cat === state.filter.cat)).forEach(s => {
    add(`${s.name} (${summary.bySignal[s.code]})`, state.filter.signal === s.code,
      () => { state.filter = { cat: state.filter.signal === s.code ? state.filter.cat : null, signal: state.filter.signal === s.code ? null : s.code }; });
  });
  body.appendChild(filters);

  const rows = items.filter(matches);
  const shown = state.showAll ? rows : rows.slice(0, PAGE_SIZE);

  const scroll = el('div', 'table-scroll');
  const table = el('table', 'watchlist');
  const head = table.createTHead().insertRow();
  [['Issue', ''], ['Summary', ''], ['Status', ''], ['Assignee', ''], ['Age', 'num'], ['SP', 'num'], ['Signals', '']].forEach(([name, cls]) => {
    const th = el('th', cls, name);
    if (name === 'Age') th.title = 'Days since work started (in progress) or since the issue was created (not started)';
    head.appendChild(th);
  });
  const tbody = table.createTBody();
  shown.forEach(it => {
    const tr = tbody.insertRow();
    const key = tr.insertCell();
    key.className = 'key';
    if (state.settings.jiraUrl) {
      const a = el('a', null, it.key);
      a.href = `${state.settings.jiraUrl}/browse/${encodeURIComponent(it.key)}`;
      a.target = '_blank';
      a.rel = 'noopener noreferrer';
      key.appendChild(a);
    } else {
      key.textContent = it.key;
    }
    if (it.inSprint) key.appendChild(el('span', 'in-sprint', 'sprint'));

    const summary = tr.insertCell();
    summary.className = 'summary';
    const text = el('span', null, it.summary);
    text.title = `${it.type}${it.priority ? ' · ' + it.priority : ''}: ${it.summary}`;
    summary.appendChild(text);

    tr.insertCell().textContent = it.status;
    tr.insertCell().textContent = it.assignee || '—';
    const age = tr.insertCell();
    age.className = 'num';
    age.textContent = it.ageDays === null ? '—' : `${Math.round(it.ageDays)} d`;
    const sp = tr.insertCell();
    sp.className = 'num';
    sp.textContent = it.storyPoints > 0 ? fmt(it.storyPoints) : '—';

    const signals = tr.insertCell();
    const chips = el('div', 'chips');
    it.signals.forEach(s => chips.appendChild(chip(s)));
    signals.appendChild(chips);
  });
  scroll.appendChild(table);
  body.appendChild(scroll);

  if (!rows.length) {
    body.appendChild(el('div', 'chart-empty', 'No issue matches this filter.'));
  } else if (rows.length > shown.length) {
    const more = el('button', 'card-toggle show-more', `Show all ${rows.length}`);
    more.type = 'button';
    more.addEventListener('click', () => { state.showAll = true; renderWatchlist(); });
    body.appendChild(more);
  }
}

// ── Bootstrap ────────────────────────────────────────────────────────────────

async function init() {
  Dash.initTheme();
  $('refresh').addEventListener('click', () => load(true));
  try {
    const [teams, settings] = await Promise.all([api('/api/teams'), api('/api/raid/settings')]);
    state.teams = teams;
    state.settings = settings;
  } catch (err) {
    showError(`Failed to load dashboard configuration: ${err.message}`);
    return;
  }
  Dash.renderTeams(state.teams, selectTeam);
  if (!state.teams.length) {
    $('cards').innerHTML = '<div class="empty-state">No teams are configured (TEAMS in .env).</div>';
    return;
  }
  const hash = decodeURIComponent(location.hash.replace(/^#/, ''));
  selectTeam(state.teams.some(t => t.componentName === hash) ? hash : state.teams[0].componentName);
}

document.addEventListener('DOMContentLoaded', init);
