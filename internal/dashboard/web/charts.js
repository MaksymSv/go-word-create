'use strict';

// Minimal SVG chart renderer (columns and lines on one shared y axis) for the
// delivery dashboard. No dependencies. Colors come from CSS custom properties.
const Charts = (() => {
  const NS = 'http://www.w3.org/2000/svg';
  const H = 250;
  const PAD = { top: 12, right: 14, bottom: 30, left: 44 };
  const BAR_RADIUS = 4;
  const BAR_GAP = 2;
  const MAX_BAR = 34;

  function svg(tag, attrs, parent) {
    const node = document.createElementNS(NS, tag);
    for (const [k, v] of Object.entries(attrs || {})) {
      if (k === 'style') node.setAttribute('style', v);
      else node.setAttribute(k, v);
    }
    if (parent) parent.appendChild(node);
    return node;
  }

  function html(tag, cls, text) {
    const node = document.createElement(tag);
    if (cls) node.className = cls;
    if (text !== undefined) node.textContent = text;
    return node;
  }

  function fmtNum(v) {
    if (v === null || v === undefined || Number.isNaN(v)) return '—';
    const r = Math.round(v * 10) / 10;
    return Number.isInteger(r) ? String(r) : r.toFixed(1);
  }

  // Rounds [min,max] outward to a "nice" step so ticks land on round numbers.
  function niceScale(min, max, ticks) {
    if (max === min) max = min + 1;
    const raw = (max - min) / ticks;
    const mag = Math.pow(10, Math.floor(Math.log10(raw)));
    const norm = raw / mag;
    const step = (norm <= 1 ? 1 : norm <= 2 ? 2 : norm <= 5 ? 5 : 10) * mag;
    return { min: Math.floor(min / step) * step, max: Math.ceil(max / step) * step, step };
  }

  let tip = null;
  function tooltip() {
    if (!tip) {
      tip = html('div', 'chart-tooltip');
      tip.setAttribute('role', 'status');
      document.body.appendChild(tip);
    }
    return tip;
  }
  function hideTip() { if (tip) tip.classList.remove('visible'); }

  function showTip(spec, i, evt) {
    const t = tooltip();
    t.textContent = '';
    t.appendChild(html('div', 'tt-title', spec.titles ? spec.titles[i] : spec.categories[i]));
    spec.series.forEach(s => {
      const row = html('div', 'tt-row');
      const sw = html('span', 'tt-swatch' + (s.kind === 'line' ? ' line' : ''));
      sw.style.background = s.color;
      row.appendChild(sw);
      row.appendChild(html('span', 'tt-name', s.name));
      row.appendChild(html('span', 'tt-val', (s.fmt || fmtNum)(s.values[i])));
      t.appendChild(row);
    });
    (spec.extra ? spec.extra(i) : []).forEach(([label, value]) => {
      const row = html('div', 'tt-row extra');
      row.appendChild(html('span', 'tt-name', label));
      row.appendChild(html('span', 'tt-val', value));
      t.appendChild(row);
    });
    t.classList.add('visible');
    const pad = 14;
    let x = evt.clientX + pad;
    let y = evt.clientY + pad;
    const r = t.getBoundingClientRect();
    if (x + r.width > window.innerWidth - 8) x = evt.clientX - r.width - pad;
    if (y + r.height > window.innerHeight - 8) y = evt.clientY - r.height - pad;
    t.style.left = Math.max(8, x) + 'px';
    t.style.top = Math.max(8, y) + 'px';
  }

  function legend(spec) {
    if (spec.series.length < 2) return null;
    const box = html('div', 'chart-legend');
    spec.series.forEach(s => {
      const item = html('span', 'legend-item');
      const sw = html('span', 'legend-swatch' + (s.kind === 'line' ? ' line' : '') + (s.dash ? ' dash' : ''));
      sw.style.background = s.dash ? 'transparent' : s.color;
      sw.style.borderColor = s.color;
      item.appendChild(sw);
      item.appendChild(html('span', null, s.name));
      box.appendChild(item);
    });
    return box;
  }

  function tableView(spec) {
    const table = html('table', 'chart-table');
    const head = table.createTHead().insertRow();
    [spec.categoryLabel || '', ...spec.series.map(s => s.name)].forEach(c => head.appendChild(html('th', null, c)));
    const body = table.createTBody();
    spec.categories.forEach((c, i) => {
      const tr = body.insertRow();
      tr.insertCell().textContent = spec.titles ? spec.titles[i] : c;
      spec.series.forEach(s => { tr.insertCell().textContent = (s.fmt || fmtNum)(s.values[i]); });
    });
    return table;
  }

  function hasData(spec) {
    return spec.series.some(s => s.values.some(v => v !== null && v !== undefined));
  }

  function draw(host, spec, width) {
    host.textContent = '';
    const n = spec.categories.length;
    const lineOnly = spec.series.every(s => s.kind === 'line');
    const rightPad = spec.endLabels ? 96 : PAD.right;
    const iw = Math.max(40, width - PAD.left - rightPad);
    const band = iw / n;

    // Value labels above bars: horizontal 11px, else horizontal 9px, else rotated 9px.
    const bars = spec.series.filter(s => s.kind === 'bar');
    const labelOf = (s, v) => (s.labelFmt || s.fmt || fmtNum)(v);
    const groupW = bars.length ? Math.min(band * 0.72, MAX_BAR * bars.length + BAR_GAP * (bars.length - 1)) : 0;
    const bw = bars.length ? (groupW - BAR_GAP * (bars.length - 1)) / bars.length : 0;
    const longest = Math.max(1, ...bars.flatMap(s => s.values.filter(v => v !== null && v !== undefined).map(v => labelOf(s, v).length)));
    const slot = bars.length > 1 ? bw + BAR_GAP + 2 : band - 6;
    const textW = size => longest * size * 0.58;
    let label = { size: 11, vertical: false };
    if (textW(11) + 3 > slot) label = textW(9) + 6 <= slot ? { size: 9, vertical: false } : { size: 9, vertical: true };
    const padTop = bars.length && label.vertical ? PAD.top + Math.ceil(textW(9)) + 6 : PAD.top;
    const ih = H - padTop - PAD.bottom;

    // y domain: bars always start at zero; line-only charts may zoom in when the data is flat.
    const vals = spec.series.flatMap(s => s.values).filter(v => v !== null && v !== undefined);
    if (spec.refLine) vals.push(spec.refLine.value);
    let lo = Math.min(0, ...vals);
    let hi = Math.max(...vals, 1);
    if (lineOnly && spec.zoom !== false && lo === 0) {
      const dmin = Math.min(...vals);
      if (dmin > 0.5 * hi) { const pad = (hi - dmin) * 0.25 || hi * 0.1; lo = Math.max(0, dmin - pad); hi += pad; }
    }
    const sc = niceScale(lo, hi, 4);
    const y = v => padTop + ih - ((v - sc.min) / (sc.max - sc.min)) * ih;
    const cx = i => PAD.left + band * (i + 0.5);

    const root = svg('svg', { width, height: H, viewBox: `0 0 ${width} ${H}`, role: 'img', 'aria-label': spec.ariaLabel || spec.title || 'chart' }, host);

    // grid + y axis labels
    for (let v = sc.min; v <= sc.max + sc.step / 2; v += sc.step) {
      svg('line', { x1: PAD.left, x2: PAD.left + iw, y1: y(v), y2: y(v), class: v === sc.min ? 'axis-base' : 'grid' }, root);
      const t = svg('text', { x: PAD.left - 8, y: y(v) + 4, 'text-anchor': 'end', class: 'tick' }, root);
      t.textContent = (spec.yFmt || fmtNum)(Math.round(v * 100) / 100);
    }

    // x labels (thinned out when they would collide)
    const maxLen = Math.max(...spec.categories.map(c => String(c).length), 1);
    const every = Math.max(1, Math.ceil((maxLen * 6.2 + 8) / band));
    spec.categories.forEach((c, i) => {
      if ((n - 1 - i) % every !== 0) return;
      const t = svg('text', { x: cx(i), y: H - 10, 'text-anchor': 'middle', class: 'tick' }, root);
      t.textContent = c;
    });

    // hover band, behind marks
    const hoverBand = svg('rect', { y: padTop, height: ih, width: band, class: 'hover-band', visibility: 'hidden' }, root);

    if (spec.refLine) {
      svg('line', { x1: PAD.left, x2: PAD.left + iw, y1: y(spec.refLine.value), y2: y(spec.refLine.value), class: 'ref-line' }, root);
      if (spec.refLine.label) {
        const t = svg('text', { x: PAD.left + iw - 2, y: y(spec.refLine.value) - 5, 'text-anchor': 'end', class: 'ref-label' }, root);
        t.textContent = spec.refLine.label;
      }
    }

    // columns
    const barLabels = []; // drawn last so lines never cover them
    if (bars.length) {
      bars.forEach((s, bi) => {
        s.values.forEach((v, i) => {
          if (v === null || v === undefined) return;
          const x = cx(i) - groupW / 2 + bi * (bw + BAR_GAP);
          const zero = y(0);
          const top = Math.min(zero, y(v));
          const bottom = Math.max(zero, y(v));
          const h = bottom - top;
          if (h > 0) {
            const r = Math.min(BAR_RADIUS, h, bw / 2);
            // Round the free end: the top of a positive bar, the bottom of a negative one.
            const d = v >= 0
              ? `M${x},${bottom} V${top + r} Q${x},${top} ${x + r},${top} H${x + bw - r} Q${x + bw},${top} ${x + bw},${top + r} V${bottom} Z`
              : `M${x},${top} V${bottom - r} Q${x},${bottom} ${x + r},${bottom} H${x + bw - r} Q${x + bw},${bottom} ${x + bw},${bottom - r} V${top} Z`;
            svg('path', { d, style: `fill:${(s.colors && s.colors[i]) || s.color}` }, root);
          }
          const below = v < 0;
          barLabels.push({
            x: x + bw / 2,
            y: below ? bottom + (label.vertical ? textW(label.size) + 4 : label.size + 3) : top - 4,
            text: labelOf(s, v),
          });
        });
      });
    }

    // lines (+ markers)
    const markers = [];
    spec.series.filter(s => s.kind === 'line').forEach(s => {
      let d = '';
      let pen = false; // true while a segment is open; a gap in the data lifts the pen
      s.values.forEach((v, i) => {
        if (v === null || v === undefined) { pen = false; return; }
        d += `${pen ? 'L' : 'M'}${cx(i)},${y(v)} `;
        pen = true;
      });
      svg('path', { d: d.trim(), class: 'line' + (s.dash ? ' dash' : ''), style: `stroke:${s.color}` }, root);
      const showAll = n <= 14;
      s.values.forEach((v, i) => {
        if (v === null || v === undefined) return;
        const m = svg('circle', { cx: cx(i), cy: y(v), r: 4, class: 'marker', style: `fill:${s.color}` }, root);
        if (!showAll) m.setAttribute('visibility', 'hidden');
        markers.push({ node: m, i, showAll });
      });
    });

    barLabels.forEach(l => {
      const t = label.vertical
        ? svg('text', { x: l.x + label.size * 0.35, y: l.y, 'text-anchor': 'start', transform: `rotate(-90 ${l.x + label.size * 0.35} ${l.y})`, class: 'bar-label', style: `font-size:${label.size}px` }, root)
        : svg('text', { x: l.x, y: l.y, 'text-anchor': 'middle', class: 'bar-label', style: `font-size:${label.size}px` }, root);
      t.textContent = l.text;
    });

    // direct end labels, nudged apart so they never overlap
    if (spec.endLabels) {
      const items = spec.series.map(s => {
        let last = s.values.length - 1;
        while (last >= 0 && (s.values[last] === null || s.values[last] === undefined)) last--;
        return last < 0 ? null : { s, y: y(s.values[last]), x: cx(last) };
      }).filter(Boolean).sort((a, b) => a.y - b.y);
      for (let k = 1; k < items.length; k++) if (items[k].y - items[k - 1].y < 13) items[k].y = items[k - 1].y + 13;
      items.forEach(it => {
        const t = svg('text', { x: it.x + 10, y: it.y + 4, class: 'end-label' }, root);
        t.textContent = it.s.name;
      });
    }

    // hit targets, larger than the marks
    for (let i = 0; i < n; i++) {
      const hit = svg('rect', { x: PAD.left + band * i, y: padTop, width: band, height: ih, fill: 'transparent' }, root);
      const on = evt => {
        hoverBand.setAttribute('x', PAD.left + band * i);
        hoverBand.setAttribute('visibility', 'visible');
        markers.forEach(m => { if (!m.showAll) m.node.setAttribute('visibility', m.i === i ? 'visible' : 'hidden'); });
        showTip(spec, i, evt);
      };
      hit.addEventListener('mouseenter', on);
      hit.addEventListener('mousemove', on);
      hit.addEventListener('mouseleave', () => {
        hoverBand.setAttribute('visibility', 'hidden');
        markers.forEach(m => { if (!m.showAll) m.node.setAttribute('visibility', 'hidden'); });
        hideTip();
      });
    }
  }

  // render fills `container` (a card body) with legend, chart and an optional table view.
  function render(container, spec) {
    container.textContent = '';
    if (!hasData(spec)) {
      container.appendChild(html('div', 'chart-empty', spec.emptyText || 'No data for this period.'));
      return;
    }
    const lg = legend(spec);
    if (lg) container.appendChild(lg);
    const host = html('div', 'chart-host');
    container.appendChild(host);
    const table = tableView(spec);
    table.hidden = true;
    container.appendChild(table);

    let last = 0;
    const redraw = () => {
      const w = Math.floor(host.clientWidth);
      if (w > 0 && w !== last) { last = w; draw(host, spec, w); }
    };
    redraw();
    new ResizeObserver(() => requestAnimationFrame(redraw)).observe(host);

    return {
      toggleTable() {
        table.hidden = !table.hidden;
        host.hidden = !table.hidden;
        if (!host.hidden) { last = 0; redraw(); }
        return !table.hidden;
      },
    };
  }

  return { render, fmtNum };
})();
