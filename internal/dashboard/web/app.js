'use strict';

const ICON_MAP = {
  'Bug':      '/static/icons/bug.svg',
  'Story':    '/static/icons/story.svg',
  'Task':     '/static/icons/task.svg',
  'Sub-task': '/static/icons/subtask.svg',
  'Subtask':  '/static/icons/subtask.svg',
  'Epic':     '/static/icons/epic.svg',
};
const DEFAULT_ICON = '/static/icons/issue.svg';

const state = {
  selectedTeam: null,
  selectedSprint: null,
};

// ── Theme ──────────────────────────────────────────────────────────────────

function applyTheme(theme) {
  document.documentElement.setAttribute('data-theme', theme);
  const btn = document.getElementById('theme-toggle');
  if (btn) btn.textContent = theme === 'dark' ? '☀️' : '🌙';
}

function initTheme() {
  const saved = localStorage.getItem('sprint-dashboard-theme') || 'light';
  applyTheme(saved);
  const btn = document.getElementById('theme-toggle');
  if (btn) {
    btn.addEventListener('click', () => {
      const current = document.documentElement.getAttribute('data-theme') || 'light';
      const next = current === 'dark' ? 'light' : 'dark';
      applyTheme(next);
      localStorage.setItem('sprint-dashboard-theme', next);
    });
  }
}

// ── Error banner ───────────────────────────────────────────────────────────

function showError(msg) {
  const banner = document.getElementById('error-banner');
  if (!banner) return;
  banner.textContent = msg;
  banner.removeAttribute('hidden');
}

function clearError() {
  const banner = document.getElementById('error-banner');
  if (banner) banner.setAttribute('hidden', '');
}

// ── Teams ──────────────────────────────────────────────────────────────────

async function fetchTeams() {
  try {
    const res = await fetch('/api/teams');
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const teams = await res.json();
    renderTeamButtons(teams);
  } catch (err) {
    showError(`Failed to load teams: ${err.message}`);
  }
}

function renderTeamButtons(teams) {
  const section = document.getElementById('team-section');
  section.innerHTML = '';
  teams.forEach((team, idx) => {
    const btn = document.createElement('button');
    btn.className = 'btn-team';
    btn.textContent = team.componentName;
    btn.dataset.component = team.componentName;
    btn.addEventListener('click', () => onTeamClick(team.componentName));
    section.appendChild(btn);
    if (idx === 0) btn.click();
  });
}

function onTeamClick(component) {
  if (state.selectedTeam === component) return;
  state.selectedTeam = component;
  state.selectedSprint = null;

  document.querySelectorAll('.btn-team').forEach(b => {
    b.classList.toggle('active', b.dataset.component === component);
  });

  document.getElementById('sprint-section').innerHTML = '';
  document.getElementById('content').innerHTML =
    '<div class="empty-state">Select a sprint to load issues.</div>';
  clearError();

  fetchSprints(component);
}

// ── Sprints ────────────────────────────────────────────────────────────────

async function fetchSprints(component) {
  try {
    const res = await fetch(`/api/teams/${encodeURIComponent(component)}/sprints`);
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error(body.error || `HTTP ${res.status}`);
    }
    const sprints = await res.json();
    renderSprintButtons(sprints);
  } catch (err) {
    showError(`Failed to load sprints for "${component}": ${err.message}`);
  }
}

function renderSprintButtons(sprints) {
  const section = document.getElementById('sprint-section');
  section.innerHTML = '';
  if (!sprints || sprints.length === 0) {
    section.innerHTML = '<span style="font-size:12px;color:var(--color-text-muted)">No sprints found</span>';
    return;
  }
  sprints.forEach(sprint => {
    const btn = document.createElement('button');
    btn.className = 'btn-sprint';
    btn.textContent = sprint.name;
    btn.dataset.sprintId = sprint.id;
    btn.addEventListener('click', () => onSprintClick(sprint.id));
    section.appendChild(btn);
  });
}

function onSprintClick(sprintId) {
  state.selectedSprint = sprintId;

  document.querySelectorAll('.btn-sprint').forEach(b => {
    b.classList.toggle('active', Number(b.dataset.sprintId) === sprintId);
  });

  document.getElementById('content').innerHTML =
    '<div class="loading-state">Loading issues…</div>';
  clearError();

  fetchIssues(sprintId);
}

// ── Issues ─────────────────────────────────────────────────────────────────

async function fetchIssues(sprintId) {
  try {
    const res = await fetch(`/api/sprints/${sprintId}/issues`);
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error(body.error || `HTTP ${res.status}`);
    }
    const data = await res.json();
    renderTable(data);
  } catch (err) {
    showError(`Failed to load issues: ${err.message}`);
    document.getElementById('content').innerHTML = '';
  }
}

function renderTable(data) {
  const { configuredLabels, issues } = data;
  const content = document.getElementById('content');

  if (!issues || issues.length === 0) {
    content.innerHTML = '<div class="empty-state">No issues found for this sprint.</div>';
    return;
  }

  const table = document.createElement('table');

  // Header
  const thead = table.createTHead();
  const headerRow = thead.insertRow();
  ['Type', 'Key', 'Summary', 'Epic', 'Implementer', 'SP', 'Status', 'Labels'].forEach(col => {
    const th = document.createElement('th');
    th.textContent = col;
    headerRow.appendChild(th);
  });

  // Body
  const tbody = table.createTBody();
  issues.forEach(issue => {
    const tr = tbody.insertRow();

    // Type icon
    const typeCell = tr.insertCell();
    const img = document.createElement('img');
    img.className = 'issue-icon';
    img.src = ICON_MAP[issue.type] || DEFAULT_ICON;
    img.alt = issue.type;
    img.title = issue.type;
    typeCell.appendChild(img);

    // Key (link)
    const keyCell = tr.insertCell();
    const a = document.createElement('a');
    a.className = 'issue-key-link';
    a.href = issue.url;
    a.target = '_blank';
    a.rel = 'noopener noreferrer';
    a.textContent = issue.key;
    keyCell.appendChild(a);

    // Summary
    tr.insertCell().textContent = issue.summary;

    // Epic
    tr.insertCell().textContent = issue.epic || '—';

    // Implementer
    tr.insertCell().textContent = issue.implementer || '—';

    // Story Points
    const spCell = tr.insertCell();
    spCell.className = 'sp-value';
    spCell.textContent = issue.storyPoints > 0 ? issue.storyPoints : '—';

    // Status
    tr.insertCell().textContent = issue.status;

    // Labels
    const labelsCell = tr.insertCell();
    const labelsDiv = document.createElement('div');
    labelsDiv.className = 'labels-cell';

    const activeSet = new Set(issue.activeLabels || []);
    (configuredLabels || []).forEach(label => {
      const btn = document.createElement('button');
      btn.className = 'btn-label' + (activeSet.has(label) ? ' active' : '');
      btn.textContent = label;
      btn.dataset.key = issue.key;
      btn.dataset.label = label;
      btn.addEventListener('click', onLabelClick);
      labelsDiv.appendChild(btn);
    });

    labelsCell.appendChild(labelsDiv);
    tr.appendChild(labelsCell);
  });

  content.innerHTML = '';
  content.appendChild(table);
}

// ── Label toggle ───────────────────────────────────────────────────────────

async function onLabelClick(e) {
  const btn = e.currentTarget;
  const { key, label } = btn.dataset;
  const wasActive = btn.classList.contains('active');
  const action = wasActive ? 'remove' : 'add';

  btn.classList.add('pending');
  btn.disabled = true;

  try {
    const res = await fetch(`/api/issues/${encodeURIComponent(key)}/labels`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action, label }),
    });
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error(body.error || `HTTP ${res.status}`);
    }
    btn.classList.remove('pending');
    btn.classList.toggle('active', !wasActive);
  } catch (err) {
    btn.classList.remove('pending');
    btn.classList.toggle('active', wasActive);
    showError(`Failed to ${action} label "${label}" on ${key}: ${err.message}`);
  } finally {
    btn.disabled = false;
  }
}

// ── Bootstrap ──────────────────────────────────────────────────────────────

document.addEventListener('DOMContentLoaded', () => {
  initTheme();
  fetchTeams();
});
