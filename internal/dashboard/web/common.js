'use strict';

// Helpers shared by the delivery and RAID pages: theme, API calls, team tabs,
// sprint-count pills and the card shell. Chart drawing lives in charts.js.
const Dash = (() => {
  const THEME_KEY = 'sprint-dashboard-theme';
  const $ = id => document.getElementById(id);

  function store(key, value) {
    try { localStorage.setItem(key, value); } catch (e) { /* storage may be unavailable */ }
  }
  function recall(key) {
    try { return localStorage.getItem(key); } catch (e) { return null; }
  }

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

  // One tab per team in #team-section; onSelect receives the component name.
  function renderTeams(teams, onSelect) {
    const section = $('team-section');
    section.textContent = '';
    teams.forEach(team => {
      const btn = document.createElement('button');
      btn.className = 'btn-team';
      btn.textContent = team.componentName;
      btn.dataset.component = team.componentName;
      btn.setAttribute('role', 'tab');
      btn.addEventListener('click', () => onSelect(team.componentName));
      section.appendChild(btn);
    });
  }

  function markActiveTeam(active) {
    document.querySelectorAll('#team-section .btn-team').forEach(b => {
      const on = b.dataset.component === active;
      b.classList.toggle('active', on);
      b.setAttribute('aria-selected', on ? 'true' : 'false');
    });
  }

  // Sprint-count pills in #sprint-count-section; onPick receives the chosen count.
  function renderSprintCounts({ defaultCount, maxSprints, current, onPick }) {
    const section = $('sprint-count-section');
    section.textContent = '';
    const label = document.createElement('span');
    label.textContent = 'Sprints:';
    section.appendChild(label);
    const options = new Set([3, 6, 9, 12, defaultCount, current]);
    [...options].filter(n => n >= 1 && n <= maxSprints).sort((a, b) => a - b).forEach(n => {
      const pill = document.createElement('button');
      pill.className = 'pill' + (n === current ? ' active' : '');
      pill.textContent = n;
      pill.dataset.count = n;
      pill.addEventListener('click', () => { if (n !== current) onPick(n); });
      section.appendChild(pill);
    });
  }

  // "PC26.17" -> "26.17" when every sprint shares the same letter prefix (full names stay in tooltips).
  function shortSprintNames(names) {
    const prefix = names.map(n => (n.match(/^[A-Za-z]+/) || [''])[0]);
    return prefix[0] && prefix.every(p => p === prefix[0]) ? names.map(n => n.slice(prefix[0].length)) : names;
  }

  // addCard appends a card to parent. def = { key, title, sub, note, wide, spec | build }.
  // A chart card gives spec() (see Charts.render) and gets a Table toggle; any other
  // card gives build(body), which fills the card body itself. errors[def.key] replaces
  // the body with the failure message.
  function addCard(parent, errors, def) {
    const card = document.createElement('section');
    card.className = 'card' + (def.wide ? ' wide' : '');

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
      return card;
    }

    if (def.build) {
      def.build(body);
    } else {
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
    }
    if (def.note) {
      const note = document.createElement('div');
      note.className = 'card-note';
      note.textContent = def.note;
      card.appendChild(note);
    }
    return card;
  }

  return { $, store, recall, initTheme, showError, clearError, api, renderTeams, markActiveTeam, renderSprintCounts, shortSprintNames, addCard };
})();
