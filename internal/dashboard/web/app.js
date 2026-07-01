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
  selectedStatuses: new Set(),
  configuredLabels: [],
};

// ── Theme ──────────────────────────────────────────────────────────────────

function applyTheme(theme) {
  document.documentElement.setAttribute('data-theme', theme);
  const btn = document.getElementById('theme-toggle');
  if (btn) btn.textContent = theme === 'dark' ? '⚪' : '☾';
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
  state.selectedStatuses.clear();
  state.configuredLabels = [];

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
  state.selectedStatuses.clear();
  state.configuredLabels = [];

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

function renderStatusPills(statuses) {
  const filterBar = document.getElementById('status-filter-bar');
  if (!filterBar) return;

  // Clear existing pills
  filterBar.innerHTML = '';

  // Add status label
  const statusLabel = document.createElement('span');
  statusLabel.textContent = 'Status:';
  filterBar.appendChild(statusLabel);

  // Create pills for each unique status
  const sortedStatuses = [...new Set(statuses)].sort();
  sortedStatuses.forEach(status => {
    const pill = document.createElement('button');
    pill.className = 'pill';
    pill.textContent = status;
    pill.dataset.status = status;

    // Check if this status is currently selected
    if (state.selectedStatuses.has(status)) {
      pill.classList.add('active');
    }

    pill.addEventListener('click', onStatusPillClick);
    filterBar.appendChild(pill);
  });
}

function onStatusPillClick(e) {
  const status = e.currentTarget.dataset.status;

  if (state.selectedStatuses.has(status)) {
    state.selectedStatuses.delete(status);
    e.currentTarget.classList.remove('active');
  } else {
    state.selectedStatuses.add(status);
    e.currentTarget.classList.add('active');
  }

  // Re-filter the table
  applyFilter();
}

function applyFilter() {
  const content = document.getElementById('content');
  const table = content.querySelector('table');

  if (!table) return;

  // Get all issue rows (excluding header)
  const rows = table.querySelectorAll('tbody tr');

  // If no statuses are selected, show all rows
  if (state.selectedStatuses.size === 0) {
    rows.forEach(row => {
      row.classList.remove('hidden');
    });
  } else {
    // Filter rows by status
    rows.forEach(row => {
      const statusCell = row.cells[5]; // Status column (index 5)
      const statusText = statusCell.textContent;

      if (state.selectedStatuses.has(statusText)) {
        row.classList.remove('hidden');
      } else {
        row.classList.add('hidden');
      }
    });
  }

  // Update summary row
  const filteredIssues = getFilteredIssues();
  renderSummaryRow(filteredIssues, state.configuredLabels);
}

function getFilteredIssues() {
  const content = document.getElementById('content');
  const table = content.querySelector('table');

  if (!table) return [];

  // Get the data from the original issues array
  const allIssues = JSON.parse(localStorage.getItem('dashboardIssues') || '[]');

  // Filter issues based on current status filter
  if (state.selectedStatuses.size === 0) {
    return allIssues;
  }

  return allIssues.filter(issue => state.selectedStatuses.has(issue.status));
}

function renderSummaryRow(filteredIssues, configuredLabels) {
  const content = document.getElementById('content');
  const table = content.querySelector('table');

  if (!table) return;

  // Remove existing summary row if present
  const existingSummary = table.querySelector('.summary-row');
  if (existingSummary) {
    existingSummary.remove();
  }

  // If no filtered issues, don't render summary row
  if (!filteredIssues || filteredIssues.length === 0) {
    return;
  }

  // Create summary row
  const tfoot = document.createElement('tfoot');
  const summaryRow = tfoot.insertRow();
  summaryRow.className = 'summary-row';

  // Create the first cell with "Summary"
  const summaryCell = summaryRow.insertCell();
  summaryCell.colSpan = 4;
  summaryCell.textContent = 'Summary';

  // Create cells for each label
  const labelsCell = summaryRow.insertCell();
  labelsCell.colSpan = configuredLabels.length;

  const labelDivs = document.createElement('div');
  labelDivs.style.display = 'flex';
  labelDivs.style.flexWrap = 'wrap';
  labelDivs.style.gap = '8px';

  // For each configured label, calculate count and total SP
  const labelTotals = {};

  filteredIssues.forEach(issue => {
    if (issue.activeLabels) {
      issue.activeLabels.forEach(label => {
        if (!labelTotals[label]) {
          labelTotals[label] = { count: 0, totalSP: 0 };
        }
        labelTotals[label].count += 1;
        labelTotals[label].totalSP += issue.storyPoints || 0;
      });
    }
  });

  // Create label summary elements
  configuredLabels.forEach(label => {
    const total = labelTotals[label] || { count: 0, totalSP: 0 };
    const labelDiv = document.createElement('div');
    labelDiv.textContent = `${label}: ${total.count} / ${total.totalSP}`;
    labelDiv.style.fontSize = '12px';
    labelDiv.style.fontWeight = '600';
    labelDivs.appendChild(labelDiv);
  });

  labelsCell.appendChild(labelDivs);

  // Append the summary row to the table
  table.appendChild(tfoot);
}

function renderTable(data) {
  const { configuredLabels, issues } = data;
  const content = document.getElementById('content');

  if (!issues || issues.length === 0) {
    content.innerHTML = '<div class="empty-state">No issues found for this sprint.</div>';
    return;
  }

  // Store the original issues and labels for filtering/summary
  state.configuredLabels = configuredLabels || [];
  localStorage.setItem('dashboardIssues', JSON.stringify(issues));

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
  });

  content.innerHTML = '';
  content.appendChild(table);

  // Render status pills
  const statuses = issues.map(issue => issue.status);
  renderStatusPills(statuses);

  // Render summary row for all issues (no filter active yet)
  renderSummaryRow(issues, state.configuredLabels);
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
