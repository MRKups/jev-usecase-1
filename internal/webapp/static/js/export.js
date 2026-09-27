// Export Module: CSV and Report Generators for Decision Engine Triage Results

function csvEscape(val) {
  if (val === null || val === undefined) return '""';
  const str = String(val);
  return '"' + str.replace(/"/g, '""') + '"';
}

function exportEvaluationCSV() {
  if (!Array.isArray(dataset) || dataset.length === 0) {
    if (typeof showToast === 'function') {
      showToast('No tickets loaded to export.', 4000);
    }
    return;
  }

  const criteria = (typeof getActiveCriteria === 'function')
    ? getActiveCriteria()
    : (triageCriteria && triageCriteria.length > 0 ? triageCriteria : DEFAULT_CRITERIA);

  const modelA = (typeof getEngineModelBadge === 'function') ? getEngineModelBadge('a') : 'Engine A';
  const modelB = (typeof getEngineModelBadge === 'function') ? getEngineModelBadge('b') : 'Engine B';

  // Build CSV Header row
  const headers = [
    'Ticket ID',
    'Title',
    'Description',
    'Filed Domain',
    'Filed Urgency',
    'Reporter',
    'Affected System',
    'Created At',
    'Engine A Model',
    'Engine A Latency (ms)',
  ];

  criteria.forEach(q => {
    const label = (typeof getQuestionTableHeader === 'function') ? getQuestionTableHeader(q) : q.id;
    headers.push('Engine A: ' + label);
  });

  headers.push(
    'Engine B Model',
    'Engine B Latency (ms)'
  );

  criteria.forEach(q => {
    const label = (typeof getQuestionTableHeader === 'function') ? getQuestionTableHeader(q) : q.id;
    headers.push('Engine B: ' + label);
  });

  headers.push(
    'Agreement Score',
    'Agreement Percentage',
    'Agreed Criteria Count',
    'Total Evaluated Criteria',
    'Disagreed Criteria'
  );

  const csvRows = [];
  csvRows.push(headers.map(csvEscape).join(','));

  // Build each data row
  dataset.forEach(item => {
    const t = item.ticket || item;
    const resA = item.triage_a || item.triage;
    const resB = item.triage_b;

    const hasA = resA && resA.answers;
    const hasB = resB && resB.answers;

    let agreementScore = '';
    let agreementPct = '';
    let agreedCountStr = '';
    let totalCritStr = '';
    const disagreedList = [];

    if (hasA && hasB) {
      let matched = 0;
      const total = criteria.length;
      criteria.forEach(q => {
        const k = q.id;
        const vA = resA.answers[k];
        const vB = resB.answers[k];
        const valAExists = vA !== undefined && vA !== null && String(vA).trim() !== '';
        const valBExists = vB !== undefined && vB !== null && String(vB).trim() !== '';

        if (valAExists && valBExists && typeof areAnswersEqual === 'function' && areAnswersEqual(vA, vB)) {
          matched++;
        } else {
          const qName = (typeof getQuestionTableHeader === 'function') ? getQuestionTableHeader(q) : k;
          disagreedList.push(qName);
        }
      });

      agreementScore = matched + '/' + total;
      agreementPct = ((matched / total) * 100).toFixed(1) + '%';
      agreedCountStr = String(matched);
      totalCritStr = String(total);
    }

    const row = [
      t.id || '',
      t.summary || '',
      t.description || '',
      t.domain || '',
      t.urgency || '',
      t.reporter || '',
      t.affected_system || '',
      t.created_at || '',
      hasA ? (resA.model || modelA || '') : '',
      hasA && typeof resA.latency_ms === 'number' ? resA.latency_ms : '',
    ];

    criteria.forEach(q => {
      row.push(hasA ? (resA.answers[q.id] ?? '') : '');
    });

    row.push(
      hasB ? (resB.model || modelB || '') : '',
      hasB && typeof resB.latency_ms === 'number' ? resB.latency_ms : ''
    );

    criteria.forEach(q => {
      row.push(hasB ? (resB.answers[q.id] ?? '') : '');
    });

    row.push(
      agreementScore,
      agreementPct,
      agreedCountStr,
      totalCritStr,
      disagreedList.join('; ')
    );

    csvRows.push(row.map(csvEscape).join(','));
  });

  const csvContent = '\uFEFF' + csvRows.join('\r\n');
  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  const dateStr = new Date().toISOString().slice(0, 10);
  a.download = 'ticket-eval-dataset-' + dateStr + '.csv';
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  setTimeout(() => URL.revokeObjectURL(url), 1000);

  if (typeof showToast === 'function') {
    showToast('Exported ' + dataset.length + ' tickets to CSV.');
  }
}

function initStandaloneReport() {
  let snapshotData = {};
  try {
    const snapEl = document.getElementById('snapshot-data');
    if (snapEl && snapEl.textContent) {
      snapshotData = JSON.parse(snapEl.textContent);
    }
  } catch (err) {
    console.error('Failed to parse snapshot data:', err);
  }

  const dataset = snapshotData.dataset || [];
  const triageCriteria = snapshotData.criteria || [];
  const appConfig = {
    engine_a: snapshotData.engineA || {},
    engine_b: snapshotData.engineB || {}
  };
  const exportedAt = snapshotData.exportedAt || new Date().toISOString();

  function getCurrentTheme() {
    return document.documentElement.getAttribute('data-theme') || 'dark';
  }

  function applyTheme(theme, save = true) {
    document.documentElement.setAttribute('data-theme', theme);
    if (save) {
      try {
        localStorage.setItem('theme', theme);
      } catch (e) {}
    }
  }

  function toggleTheme() {
    const current = getCurrentTheme();
    const next = current === 'dark' ? 'light' : 'dark';
    applyTheme(next, true);
  }

  if (window.matchMedia) {
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
      try {
        if (!localStorage.getItem('theme')) {
          applyTheme(e.matches ? 'dark' : 'light', false);
        }
      } catch (err) {}
    });
  }

  window.toggleTheme = toggleTheme;

  let selectedTicketId = null;

  const CANONICAL_TABLE_HEADERS = {
    ticket_type: 'Type',
    technical_domain: 'Domain',
    operational_urgency: 'Urgency',
    security_incident: 'Security',
    target_resolution_group: 'Group',
    blast_radius: 'Blast radius'
  };

  const CANONICAL_MODAL_LABELS = {
    ticket_type: 'Ticket Type',
    technical_domain: 'Technical Domain',
    operational_urgency: 'Operational Urgency',
    security_incident: 'Security Incident',
    target_resolution_group: 'Resolution Group',
    blast_radius: 'Blast Radius'
  };

  const DEFAULT_CRITERIA = [
    { id: 'ticket_type', text: 'Is this ticket an Incident or a Service Request?' },
    { id: 'technical_domain', text: 'What is the primary technical domain of this ticket?' },
    { id: 'operational_urgency', text: 'What is the verified operational urgency level based on technical impact?' },
    { id: 'security_incident', text: 'Does this issue indicate a security breach, unauthorized access, or policy violation?' },
    { id: 'target_resolution_group', text: 'Which support tier or engineering team should resolve this issue?' },
    { id: 'blast_radius', text: 'Does this issue affect an isolated user or indicate a broader systemic outage?' }
  ];

  function cleanId(id) {
    if (!id) return '';
    return String(id).replace(/[^a-zA-Z0-9_-]/g, '');
  }

  function escapeHtml(str) {
    if (str === null || str === undefined) return '';
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }

  function getActiveCriteria() {
    if (Array.isArray(triageCriteria) && triageCriteria.length > 0) return triageCriteria;
    return DEFAULT_CRITERIA;
  }

  function formatEngineModelBadge(prov, model) {
    prov = (prov || '').toLowerCase();
    model = (model || '').trim();
    if (prov === 'disabled' || prov === 'none') return '';
    if (prov === 'jev' || prov === 'typesafe') {
      if (!model || model === 'jev-latest' || model === 'jev') return 'typesafe/jev';
      if (model.toLowerCase().startsWith('typesafe/')) return model.toLowerCase();
      return 'typesafe/' + model;
    }
    if (model) {
      if (model.includes('/')) return model;
      if (prov === 'gemini') return 'google/' + model;
      return prov + '/' + model;
    }
    if (prov === 'gemini') return 'google/gemini';
    return prov;
  }

  function getEngineModelBadge(engine, ticketRes) {
    const cfg = engine === 'a' ? appConfig.engine_a : appConfig.engine_b;
    const prov = (cfg && cfg.provider) || (engine === 'a' ? 'jev' : 'openai');
    const rawModel = (ticketRes && ticketRes.model) || (cfg && cfg.model) || '';
    return formatEngineModelBadge(prov, rawModel);
  }

  function getQuestionTableHeader(q) {
    if (!q) return '';
    if (CANONICAL_TABLE_HEADERS[q.id]) return CANONICAL_TABLE_HEADERS[q.id];
    const words = (q.id || '').split(/[_-]+/).filter(Boolean);
    if (words.length > 0) {
      const formatted = words.map(w => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase()).join(' ');
      return formatted.length > 14 ? formatted.slice(0, 13) + '…' : formatted;
    }
    return q.id || 'Custom';
  }

  function getQuestionModalLabel(q) {
    if (!q) return '';
    if (CANONICAL_MODAL_LABELS[q.id]) return CANONICAL_MODAL_LABELS[q.id];
    if (q.label) return q.label;
    if (q.text && q.text.length <= 30 && !q.text.includes('?')) return q.text;
    const words = (q.id || '').split(/[_-]+/).filter(Boolean);
    if (words.length > 0) {
      return words.map(w => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase()).join(' ');
    }
    return q.id || 'Custom';
  }

  function getQuestionColumnWidth(id) {
    switch (id) {
      case 'ticket_type': return 65;
      case 'technical_domain': return 80;
      case 'operational_urgency': return 65;
      case 'security_incident': return 55;
      case 'target_resolution_group': return 110;
      case 'blast_radius': return 90;
      default: return 85;
    }
  }

  function parseBinaryPolarity(val) {
    if (val === null || val === undefined) return null;
    if (typeof val === 'boolean') return val;
    if (typeof val === 'number') {
      if (val === 1) return true;
      if (val === 0) return false;
    }
    let s = String(val).trim().toLowerCase();
    if (!s) return null;
    s = s.replace(/^[\s`'"*._,:;()[\]{}]+|[\s`'"*._,:;()[\]{}]+$/g, '');
    if (s.endsWith('.0')) s = s.slice(0, -2);
    const affirmatives = new Set(['yes', 'true', '1', 'y', 't', 'positive', 'confirmed', 'pass', 'on', 'enabled']);
    const negatives = new Set(['no', 'false', '0', 'n', 'f', 'negative', 'unconfirmed', 'fail', 'off', 'disabled', 'none']);
    if (affirmatives.has(s)) return true;
    if (negatives.has(s)) return false;
    const leadingToken = s.split(/[\s\-_,:;]+/)[0];
    if (leadingToken) {
      if (['yes', 'true', 'y', 't', 'positive', 'confirmed'].includes(leadingToken)) return true;
      if (['no', 'false', 'n', 'f', 'negative', 'none'].includes(leadingToken)) return false;
    }
    return null;
  }

  function normalizeAnswerValue(val) {
    if (val === null || val === undefined) return '';
    const pol = parseBinaryPolarity(val);
    if (pol === true) return 'yes';
    if (pol === false) return 'no';
    return String(val).trim().toLowerCase();
  }

  function areAnswersEqual(valA, valB) {
    if (valA === undefined || valA === null || valB === undefined || valB === null) return false;
    const polA = parseBinaryPolarity(valA);
    const polB = parseBinaryPolarity(valB);
    if (polA !== null && polB !== null) return polA === polB;
    return normalizeAnswerValue(valA) === normalizeAnswerValue(valB);
  }

  function shortenCellValue(key, val) {
    if (val === null || val === undefined) return '';
    const s = String(val).trim();
    if (s === '') return '';
    if (key === 'ticket_type') {
      const lower = s.toLowerCase();
      if (lower.includes('incident')) return 'Incident';
      if (lower.includes('request')) return 'Request';
      return s;
    }
    if (key === 'technical_domain') {
      const map = {
        identity_access_management: 'IAM',
        access_management: 'IAM',
        network_infrastructure: 'Network',
        network: 'Network',
        hardware_endpoint: 'Hardware',
        endpoint: 'Endpoint',
        hardware: 'Hardware',
        data_storage: 'Storage',
        storage: 'Storage',
        communication_collaboration: 'Collab',
        collaboration: 'Collab',
        enterprise_software: 'Software',
        cloud_platform: 'Cloud',
        cloud: 'Cloud',
        application: 'App',
        security: 'Security'
      };
      const cleaned = s.toLowerCase().replace(/[\s-]+/g, '_');
      if (map[cleaned]) return map[cleaned];
      const words = s.split(/[\s\-_]+/);
      return words[0].charAt(0).toUpperCase() + words[0].slice(1);
    }
    if (key === 'operational_urgency') {
      const lower = s.toLowerCase();
      if (lower.includes('crit')) return 'Critical';
      if (lower.includes('high')) return 'High';
      if (lower.includes('med')) return 'Medium';
      if (lower.includes('low')) return 'Low';
      return s;
    }
    if (key === 'security_incident') {
      const pol = parseBinaryPolarity(val);
      if (pol === true) return 'Yes';
      if (pol === false) return 'No';
      return s;
    }
    if (key === 'target_resolution_group') {
      const map = {
        endpoint_engineering: 'Endpoint Eng',
        field_services: 'Field Serv',
        helpdesk_tier_1: 'Tier 1 Support',
        helpdesk_tier_2: 'Tier 2 Support',
        identity_operations: 'IAM Ops',
        network_operations_center: 'NOC',
        security_operations_center: 'SOC',
        storage_virtualization_team: 'Storage Eng'
      };
      const cleaned = s.toLowerCase().replace(/[\s-]+/g, '_');
      if (map[cleaned]) return map[cleaned];
      if (s.length > 14) return s.slice(0, 13) + '…';
      return s;
    }
    if (key === 'blast_radius') {
      const lower = s.toLowerCase();
      if (lower.includes('organization')) return 'Entire Org';
      if (lower.includes('multi')) return 'Multi Dept';
      if (lower.includes('single')) return 'Single Dept';
      if (lower.includes('individual')) return 'Single User';
      if (s.length > 13) return s.slice(0, 12) + '…';
      return s;
    }
    return s.length > 16 ? s.slice(0, 15) + '…' : s;
  }

  function formatLatencyString(ms) {
    if (ms === null || ms === undefined || ms < 0) return '-';
    if (ms < 1000) return ms + ' ms';
    return (ms / 1000).toFixed(1) + ' s';
  }

  function calculateMedian(arr) {
    if (!arr || arr.length === 0) return null;
    const sorted = arr.slice().sort((a, b) => a - b);
    const mid = Math.floor(sorted.length / 2);
    if (sorted.length % 2 === 1) return sorted[mid];
    return Math.round((sorted[mid - 1] + sorted[mid]) / 2);
  }

  function updateComparisonTableStructure(tableQuestions) {
    const colSpanCount = tableQuestions.length + 1;
    const hdrA = document.getElementById('hdr-engine-a-group');
    const hdrB = document.getElementById('hdr-engine-b-group');
    if (hdrA) hdrA.colSpan = colSpanCount;
    if (hdrB) hdrB.colSpan = colSpanCount;

    const colgroup = document.getElementById('comparison-colgroup');
    if (colgroup) {
      let cols = '<col style="width: 28px;"><col style="width: 60px;"><col style="width: 220px;"><col style="width: 120px;"><col style="width: 65px;"><col style="width: 50px;">';
      tableQuestions.forEach(q => { cols += '<col style="width: ' + getQuestionColumnWidth(q.id) + 'px;">'; });
      cols += '<col style="width: 65px;">';
      tableQuestions.forEach(q => { cols += '<col style="width: ' + getQuestionColumnWidth(q.id) + 'px;">'; });
      cols += '<col style="width: 65px;">';
      colgroup.innerHTML = cols;
    }

    const subHeaderRow = document.getElementById('comparison-sub-header-row');
    if (subHeaderRow) {
      let ths = '<th></th><th>ID</th><th>Title</th><th>Filed as</th><th>Urgency</th><th class="col-border-ticket-end" style="text-align: center;">Agree</th>';
      tableQuestions.forEach(q => { ths += '<th>' + escapeHtml(getQuestionTableHeader(q)) + '</th>'; });
      ths += '<th class="col-border-engine-a-end" style="text-align: right;">Time</th>';
      tableQuestions.forEach(q => { ths += '<th>' + escapeHtml(getQuestionTableHeader(q)) + '</th>'; });
      ths += '<th style="text-align: right;">Time</th>';
      subHeaderRow.innerHTML = ths;
    }
  }

  function updateGroupHeaders(tableQuestions, activeCriteria) {
    const latenciesA = [];
    const latenciesB = [];
    dataset.forEach(t => {
      const resA = t.triage_a || t.triage;
      const resB = t.triage_b;
      if (resA && typeof resA.latency_ms === 'number' && resA.latency_ms > 0) latenciesA.push(resA.latency_ms);
      if (resB && typeof resB.latency_ms === 'number' && resB.latency_ms > 0) latenciesB.push(resB.latency_ms);
    });

    const medA = calculateMedian(latenciesA);
    const medB = calculateMedian(latenciesB);

    const medAEl = document.getElementById('hdr-engine-a-median');
    const medBEl = document.getElementById('hdr-engine-b-median');
    if (medAEl) medAEl.innerText = medA !== null ? 'median ' + formatLatencyString(medA) : 'median -';
    if (medBEl) medBEl.innerText = medB !== null ? 'median ' + formatLatencyString(medB) : 'median -';

    const modelAEl = document.getElementById('hdr-engine-a-model');
    const modelBEl = document.getElementById('hdr-engine-b-model');
    const modelA = getEngineModelBadge('a');
    const modelB = getEngineModelBadge('b');
    if (modelAEl) {
      modelAEl.innerText = modelA;
      modelAEl.style.display = 'inline-block';
    }
    if (modelBEl) {
      modelBEl.innerText = modelB;
      modelBEl.style.display = 'inline-block';
    }
  }

  function renderComparisonRow(t, tableQuestions, activeCriteria) {
    const resA = t.triage_a || t.triage;
    const resB = t.triage_b;
    const tk = t.ticket || t;

    const hasA = resA && resA.answers;
    const hasB = resB && resB.answers;

    let dotSymbol = '○';
    let dotClass = 'status-dot-waiting';
    if (hasA && hasB) {
      dotSymbol = '●';
      dotClass = 'status-dot-done';
    } else if (hasA || hasB) {
      dotSymbol = '◐';
      dotClass = 'status-dot-running';
    }

    let agreeText = '<span class="faint-dot">·</span>';
    let agreeClass = 'agree-pending';
    const diffs = {};

    if (hasA && hasB) {
      let matched = 0;
      const totalCrit = activeCriteria.length;
      activeCriteria.forEach(q => {
        const k = q.id;
        const vA = resA.answers[k];
        const vB = resB.answers[k];
        const valAExists = vA !== undefined && vA !== null && String(vA).trim() !== '';
        const valBExists = vB !== undefined && vB !== null && String(vB).trim() !== '';
        if (valAExists && valBExists && areAnswersEqual(vA, vB)) {
          matched++;
        } else {
          diffs[k] = true;
        }
      });

      agreeText = matched + '/' + totalCrit;
      if (matched === totalCrit && totalCrit > 0) {
        agreeClass = 'agree-full';
      } else {
        agreeClass = 'agree-diff';
      }
    }

    const renderCellA = (k, extraClass = '') => {
      if (!hasA) return '<td class="' + extraClass + '"><span class="faint-dot">·</span></td>';
      const val = resA.answers[k];
      if (val === undefined || val === null || val === '') return '<td class="' + extraClass + '"><span class="faint-dot">·</span></td>';
      const shortVal = shortenCellValue(k, val);
      const isDiff = diffs[k];
      const classes = [];
      if (extraClass) classes.push(extraClass);
      if (isDiff) classes.push('cell-diff');
      const classAttr = classes.length > 0 ? ' class="' + classes.join(' ') + '"' : '';
      return '<td' + classAttr + '>' + escapeHtml(shortVal) + '</td>';
    };

    const renderCellB = (k, extraClass = '') => {
      if (!hasB) return '<td class="' + extraClass + '"><span class="faint-dot">·</span></td>';
      const val = resB.answers[k];
      if (val === undefined || val === null || val === '') return '<td class="' + extraClass + '"><span class="faint-dot">·</span></td>';
      const shortVal = shortenCellValue(k, val);
      const isDiff = diffs[k];
      const classes = [];
      if (extraClass) classes.push(extraClass);
      if (isDiff) classes.push('cell-diff');
      const classAttr = classes.length > 0 ? ' class="' + classes.join(' ') + '"' : '';
      return '<td' + classAttr + '>' + escapeHtml(shortVal) + '</td>';
    };

    let timeA = '<span class="faint-dot">·</span>';
    if (resA && resA.latency_ms > 0) timeA = formatLatencyString(resA.latency_ms);

    let timeB = '<span class="faint-dot">·</span>';
    if (resB && resB.latency_ms > 0) timeB = formatLatencyString(resB.latency_ms);

    const isSelected = selectedTicketId === tk.id;
    const rowClass = 'comp-ticket-row' + (isSelected ? ' row-selected' : '');
    const filedCategory = tk.category || tk.domain || '-';
    const filedUrgency = tk.reported_urgency || tk.urgency || 'Medium';

    let html = '<tr class="' + rowClass + '" id="row-ticket-' + cleanId(tk.id) + '" data-ticket-id="' + escapeHtml(tk.id) + '">';
    html += '<td class="cell-dot"><span class="status-dot-icon ' + dotClass + '">' + dotSymbol + '</span></td>';
    html += '<td class="cell-id">' + escapeHtml(tk.id) + '</td>';
    html += '<td class="cell-title" title="' + escapeHtml(tk.summary) + '">' + escapeHtml(tk.summary) + '</td>';
    html += '<td class="cell-filed-as">' + escapeHtml(filedCategory) + '</td>';
    html += '<td class="cell-filed-urgency">' + escapeHtml(filedUrgency) + '</td>';
    html += '<td class="cell-agree col-border-ticket-end ' + agreeClass + '">' + agreeText + '</td>';

    tableQuestions.forEach(q => { html += renderCellA(q.id); });
    html += '<td class="cell-time col-border-engine-a-end">' + timeA + '</td>';

    tableQuestions.forEach(q => { html += renderCellB(q.id); });
    html += '<td class="cell-time">' + timeB + '</td>';
    html += '</tr>';

    return html;
  }

  function renderTable() {
    const tableQuestions = getActiveCriteria().slice(0, 6);
    updateComparisonTableStructure(tableQuestions);
    updateGroupHeaders(tableQuestions, getActiveCriteria());

    const tbody = document.getElementById('comparison-tbody');
    if (!tbody) return;

    const searchInput = document.getElementById('report-search-input');
    const searchTerm = (searchInput && searchInput.value ? searchInput.value : '').trim().toLowerCase();
    const diffsCheckbox = document.getElementById('filter-diffs-only');
    const diffsOnly = !!(diffsCheckbox && diffsCheckbox.checked);
    const activeCrit = getActiveCriteria();

    const filtered = dataset.filter(item => {
      const t = item.ticket || item;
      const resA = item.triage_a || item.triage;
      const resB = item.triage_b;

      if (searchTerm) {
        const text = ((t.id || '') + ' ' + (t.summary || '') + ' ' + (t.description || '') + ' ' + (t.category || t.domain || '') + ' ' + (t.reporter_name || t.reporter || '')).toLowerCase();
        if (!text.includes(searchTerm)) return false;
      }

      if (diffsOnly) {
        if (!resA || !resA.answers || !resB || !resB.answers) return false;
        let hasDiff = false;
        for (const q of activeCrit) {
          const vA = resA.answers[q.id];
          const vB = resB.answers[q.id];
          if (!areAnswersEqual(vA, vB)) {
            hasDiff = true;
            break;
          }
        }
        if (!hasDiff) return false;
      }

      return true;
    });

    if (filtered.length === 0) {
      tbody.innerHTML = '<tr><td colspan="20" style="text-align: center; color: var(--text-muted); padding: 40px;">No matching tickets found.</td></tr>';
      return;
    }

    tbody.innerHTML = filtered.map(t => renderComparisonRow(t, tableQuestions, activeCrit)).join('');
  }

  function selectRow(ticketId) {
    selectedTicketId = ticketId;
    document.querySelectorAll('.comp-ticket-row').forEach(r => r.classList.remove('row-selected'));
    const row = document.getElementById('row-ticket-' + cleanId(ticketId));
    if (row) row.classList.add('row-selected');
  }

  function formatModalAnswer(critId, val) {
    if (val === undefined || val === null || String(val).trim() === '') return '-';
    if (critId === 'security_incident') {
      const pol = parseBinaryPolarity(val);
      if (pol === true) return 'Yes';
      if (pol === false) return 'No';
    }
    return String(val);
  }

  function renderModalDecisions(critList, resA, resB) {
    const tbody = document.getElementById('modal-criteria-tbody');
    const matchEl = document.getElementById('modal-match-summary');
    if (!tbody) return;

    const hasA = resA && resA.answers;
    const hasB = resB && resB.answers;

    if (!hasA && !hasB) {
      tbody.innerHTML = '<tr><td colspan="3" style="text-align: center; color: var(--text-muted); padding: 24px;">No evaluation results available for this ticket.</td></tr>';
      if (matchEl) matchEl.innerText = '-';
      return;
    }

    let matchCount = 0;
    const totalCount = critList.length;

    tbody.innerHTML = critList.map(crit => {
      const valA = hasA ? resA.answers[crit.id] : undefined;
      const valB = hasB ? resB.answers[crit.id] : undefined;
      const valAExists = valA !== undefined && valA !== null && String(valA).trim() !== '';
      const valBExists = valB !== undefined && valB !== null && String(valB).trim() !== '';

      const isMatch = valAExists && valBExists && areAnswersEqual(valA, valB);
      if (isMatch) matchCount++;
      const isDiff = hasA && hasB && !isMatch;

      const rowDiffClass = isDiff ? ' class="row-diff"' : '';
      const cellDiffClass = isDiff ? ' col-diff-val' : '';

      const textA = hasA ? formatModalAnswer(crit.id, valA) : '-';
      const textB = hasB ? formatModalAnswer(crit.id, valB) : '-';

      return '<tr' + rowDiffClass + '>' +
        '<td class="modal-criteria-name">' + escapeHtml(getQuestionModalLabel(crit)) + '</td>' +
        '<td class="' + cellDiffClass + '">' + escapeHtml(textA) + '</td>' +
        '<td class="' + cellDiffClass + '">' + escapeHtml(textB) + '</td>' +
      '</tr>';
    }).join('');

    if (matchEl) {
      if (hasA && hasB) {
        const matchText = matchCount + ' of ' + totalCount + ' match';
        let speedupText = '';
        const latA = (resA && typeof resA.latency_ms === 'number' && resA.latency_ms > 0) ? resA.latency_ms : null;
        const latB = (resB && typeof resB.latency_ms === 'number' && resB.latency_ms > 0) ? resB.latency_ms : null;
        if (latA !== null && latB !== null && latA > 0 && latB > 0) {
          if (latB > latA) {
            const ratio = (latB / latA).toFixed(1);
            speedupText = ' · Decision Engine A ' + ratio + '× faster';
          } else if (latA > latB) {
            const ratio = (latA / latB).toFixed(1);
            speedupText = ' · Decision Engine B ' + ratio + '× faster';
          }
        }
        matchEl.innerText = matchText + speedupText;
        matchEl.className = 'modal-match-summary ' + (matchCount === totalCount ? 'match-all' : 'match-partial');
      } else {
        matchEl.innerText = '-';
        matchEl.className = 'modal-match-summary';
      }
    }
  }

  function openTicketModal(ticketId) {
    const item = dataset.find(t => t.id === ticketId || (t.ticket && t.ticket.id === ticketId));
    if (!item) return;
    const t = item.ticket || item;
    const resA = item.triage_a || item.triage;
    const resB = item.triage_b;

    const repName = t.reporter_name || t.reporter || 'Unknown';
    const repDept = t.department || t.domain || 'General';
    const repUrg = t.reported_urgency || t.urgency || 'Medium';

    const modalIdEl = document.getElementById('modal-ticket-id');
    if (modalIdEl) modalIdEl.innerText = t.id || ('#' + cleanId(ticketId));

    const modalTitleEl = document.getElementById('modal-ticket-title');
    if (modalTitleEl) modalTitleEl.innerText = t.summary || 'Untitled Ticket';

    const modalRepNameEl = document.getElementById('modal-reporter-name');
    if (modalRepNameEl) modalRepNameEl.innerText = repName;

    const modalRepDeptEl = document.getElementById('modal-reporter-dept');
    if (modalRepDeptEl) modalRepDeptEl.innerText = repDept;

    const modalFiledUrgEl = document.getElementById('modal-filed-urgency');
    if (modalFiledUrgEl) modalFiledUrgEl.innerText = repUrg;

    const modalDescEl = document.getElementById('modal-desc-box');
    if (modalDescEl) modalDescEl.innerText = t.description || 'No description provided.';

    const detailRep = document.getElementById('modal-detail-reporter');
    if (detailRep) detailRep.innerText = repName;

    const detailDept = document.getElementById('modal-detail-dept');
    if (detailDept) detailDept.innerText = repDept;

    const detailAsset = document.getElementById('modal-detail-asset');
    if (detailAsset) detailAsset.innerText = t.affected_system || '-';

    const detailCat = document.getElementById('modal-detail-category');
    if (detailCat) detailCat.innerText = t.category || repDept || '-';

    const detailUrg = document.getElementById('modal-detail-urgency');
    if (detailUrg) detailUrg.innerText = repUrg;

    const nameA = 'Decision Engine A';
    const nameB = 'Decision Engine B';
    const modelA = getEngineModelBadge('a', resA);
    const modelB = getEngineModelBadge('b', resB);

    const confA = resA && typeof resA.confidence === 'number' && resA.confidence > 0
      ? Math.round(resA.confidence * 100) + '%'
      : '';
    const confB = resB && typeof resB.confidence === 'number' && resB.confidence > 0
      ? Math.round(resB.confidence * 100) + '%'
      : '';

    const thA = document.getElementById('modal-th-engine-a');
    const thB = document.getElementById('modal-th-engine-b');
    if (thA) {
      thA.innerHTML = '<div class="modal-th-header-cell">' +
        '<div style="display: flex; align-items: center; gap: 6px; flex-wrap: wrap;">' +
          '<span class="modal-th-engine-name">' + escapeHtml(nameA) + '</span>' +
          (modelA ? '<span class="group-model-badge" style="font-size: 0.68rem; padding: 1px 5px;">' + escapeHtml(modelA) + '</span>' : '') +
        '</div>' +
        (confA ? '<span class="modal-th-conf-badge" title="Model confidence score for decisions">' + escapeHtml(confA) + ' confidence</span>' : '') +
      '</div>';
    }
    if (thB) {
      thB.innerHTML = '<div class="modal-th-header-cell">' +
        '<div style="display: flex; align-items: center; gap: 6px; flex-wrap: wrap;">' +
          '<span class="modal-th-engine-name">' + escapeHtml(nameB) + '</span>' +
          (modelB ? '<span class="group-model-badge" style="font-size: 0.68rem; padding: 1px 5px;">' + escapeHtml(modelB) + '</span>' : '') +
        '</div>' +
        (confB ? '<span class="modal-th-conf-badge" title="Model confidence score for decisions">' + escapeHtml(confB) + ' confidence</span>' : '') +
      '</div>';
    }

    renderModalDecisions(getActiveCriteria(), resA, resB);

    const barLabelA = document.getElementById('modal-bar-label-a');
    const barLabelB = document.getElementById('modal-bar-label-b');
    if (barLabelA) barLabelA.innerText = nameA;
    if (barLabelB) barLabelB.innerText = nameB;

    const barFillA = document.getElementById('modal-bar-fill-a');
    const barFillB = document.getElementById('modal-bar-fill-b');
    const barValA = document.getElementById('modal-bar-val-a');
    const barValB = document.getElementById('modal-bar-val-b');

    const latA = (resA && typeof resA.latency_ms === 'number' && resA.latency_ms > 0) ? resA.latency_ms : null;
    const latB = (resB && typeof resB.latency_ms === 'number' && resB.latency_ms > 0) ? resB.latency_ms : null;

    if (latA !== null && latB !== null && latA > 0 && latB > 0) {
      const maxLat = Math.max(latA, latB);
      const pctA = Math.max(4, Math.round((latA / maxLat) * 100));
      const pctB = Math.max(4, Math.round((latB / maxLat) * 100));
      if (barFillA) barFillA.style.width = pctA + '%';
      if (barFillB) barFillB.style.width = pctB + '%';
      if (barValA) barValA.innerText = formatLatencyString(latA);
      if (barValB) barValB.innerText = formatLatencyString(latB);
    } else if (latA !== null && latA > 0) {
      if (barFillA) barFillA.style.width = '100%';
      if (barFillB) barFillB.style.width = '0%';
      if (barValA) barValA.innerText = formatLatencyString(latA);
      if (barValB) barValB.innerText = '-';
    } else if (latB !== null && latB > 0) {
      if (barFillA) barFillA.style.width = '0%';
      if (barFillB) barFillB.style.width = '100%';
      if (barValA) barValA.innerText = '-';
      if (barValB) barValB.innerText = formatLatencyString(latB);
    } else {
      if (barFillA) barFillA.style.width = '0%';
      if (barFillB) barFillB.style.width = '0%';
      if (barValA) barValA.innerText = '-';
      if (barValB) barValB.innerText = '-';
    }

    const actBodyA = document.getElementById('modal-action-body-a');
    const actBodyB = document.getElementById('modal-action-body-b');
    const actHdrA = document.getElementById('modal-action-hdr-a');
    const actHdrB = document.getElementById('modal-action-hdr-b');
    if (actHdrA) actHdrA.innerText = nameA;
    if (actHdrB) actHdrB.innerText = nameB;
    const getActionText = (res, engine) => {
      const cfg = engine === 'a' ? appConfig.engine_a : appConfig.engine_b;
      const prov = (cfg && cfg.provider) || (engine === 'a' ? 'jev' : 'openai');
      if (prov === 'jev' || (res && res.engine_name && res.engine_name.toLowerCase().includes('jev'))) {
        return 'Jev is not able to generate a response to this question.';
      }
      return (res && res.recommended_action) ? res.recommended_action : 'No specific action generated.';
    };
    if (actBodyA) actBodyA.innerText = getActionText(resA, 'a');
    if (actBodyB) actBodyB.innerText = getActionText(resB, 'b');

    const backdrop = document.getElementById('ticket-modal-backdrop');
    if (backdrop) backdrop.classList.add('active');
  }

  function closeTicketModal() {
    const backdrop = document.getElementById('ticket-modal-backdrop');
    if (backdrop) backdrop.classList.remove('active');
  }

  function handleBackdropClick(event) {
    if (event.target.id === 'ticket-modal-backdrop') closeTicketModal();
  }

  window.selectRow = selectRow;
  window.openTicketModal = openTicketModal;
  window.closeTicketModal = closeTicketModal;
  window.handleBackdropClick = handleBackdropClick;

  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      closeTicketModal();
    } else if (e.key === 'Enter') {
      if (selectedTicketId) openTicketModal(selectedTicketId);
    }
  });

  const searchInput = document.getElementById('report-search-input');
  if (searchInput) searchInput.addEventListener('input', renderTable);
  const diffsCheckbox = document.getElementById('filter-diffs-only');
  if (diffsCheckbox) diffsCheckbox.addEventListener('change', renderTable);

  const tbody = document.getElementById('comparison-tbody');
  if (tbody) {
    tbody.addEventListener('click', (e) => {
      const row = e.target.closest('.comp-ticket-row');
      if (!row) return;
      const id = row.getAttribute('data-ticket-id');
      if (id) selectRow(id);
    });
    tbody.addEventListener('dblclick', (e) => {
      const row = e.target.closest('.comp-ticket-row');
      if (!row) return;
      const id = row.getAttribute('data-ticket-id');
      if (id) openTicketModal(id);
    });
  }

  let matchedTotal = 0;
  let evaluatedTickets = 0;
  dataset.forEach(item => {
    const resA = item.triage_a || item.triage;
    const resB = item.triage_b;
    if (resA && resA.answers && resB && resB.answers) {
      evaluatedTickets++;
      let matchedQuestions = 0;
      const totalQ = getActiveCriteria().length;
      getActiveCriteria().forEach(q => {
        if (areAnswersEqual(resA.answers[q.id], resB.answers[q.id])) matchedQuestions++;
      });
      matchedTotal += (matchedQuestions / totalQ);
    }
  });

  const avgAgreementPct = evaluatedTickets > 0 ? ((matchedTotal / evaluatedTickets) * 100).toFixed(1) + '%' : '-';
  const statusEl = document.getElementById('report-status-text');
  if (statusEl) {
    statusEl.innerText = dataset.length + ' tickets · ' + avgAgreementPct + ' Mean Agreement';
  }
  const dateEl = document.getElementById('report-export-date');
  if (dateEl) {
    const d = new Date(exportedAt);
    dateEl.innerText = 'Snapshot captured ' + d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  renderTable();
}

async function exportEvaluationHTML() {
  if (!Array.isArray(dataset) || dataset.length === 0) {
    if (typeof showToast === 'function') {
      showToast('No tickets loaded to export.', 4000);
    }
    return;
  }

  if (typeof showToast === 'function') {
    showToast('Generating standalone HTML report...');
  }

  let styleCss = '';
  let decisionCss = '';
  let modalCss = '';
  try {
    const [resStyle, resDecision, resModal] = await Promise.all([
      fetch('/static/css/style.css'),
      fetch('/static/css/decision.css'),
      fetch('/static/css/modal.css')
    ]);
    if (resStyle.ok) styleCss = await resStyle.text();
    if (resDecision.ok) decisionCss = await resDecision.text();
    if (resModal.ok) modalCss = await resModal.text();
  } catch (err) {
    console.error('Failed to load stylesheets for export:', err);
  }

  const criteria = (typeof getActiveCriteria === 'function')
    ? getActiveCriteria()
    : (triageCriteria && triageCriteria.length > 0 ? triageCriteria : DEFAULT_CRITERIA);

  const modelA = (typeof getEngineModelBadge === 'function') ? getEngineModelBadge('a') : 'Engine A';
  const modelB = (typeof getEngineModelBadge === 'function') ? getEngineModelBadge('b') : 'Engine B';

  const snapshot = {
    exportedAt: new Date().toISOString(),
    engineA: {
      provider: appConfig.engine_a?.provider || currentEngineAProvider,
      model: modelA
    },
    engineB: {
      provider: appConfig.engine_b?.provider || currentEngineBProvider,
      model: modelB
    },
    criteria: criteria,
    dataset: dataset
  };

  const jsonPayload = JSON.stringify(snapshot).replace(/</g, '\\u003c').replace(/>/g, '\\u003e');
  const modalBackdropEl = document.getElementById('ticket-modal-backdrop');
  let modalMarkup = modalBackdropEl ? modalBackdropEl.outerHTML : '';
  modalMarkup = modalMarkup.replace(/\\bactive\\b/g, '').trim();

  const standaloneScriptCode = '(' + initStandaloneReport.toString() + ')();';

  const htmlContent = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Triage Benchmark Snapshot</title>
  <script>
    (function() {
      try {
        const savedTheme = localStorage.getItem('theme');
        if (savedTheme === 'light' || savedTheme === 'dark') {
          document.documentElement.setAttribute('data-theme', savedTheme);
        } else {
          const prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
          document.documentElement.setAttribute('data-theme', prefersDark ? 'dark' : 'light');
        }
      } catch (e) {
        document.documentElement.setAttribute('data-theme', 'dark');
      }
    })();
  </script>
  <style>
${styleCss}
${decisionCss}
${modalCss}

    /* Standalone Report Overrides */
    body {
      margin: 0;
      padding: 0;
      background: var(--bg);
      color: var(--text);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      min-height: 100vh;
      display: flex;
      flex-direction: column;
    }
    header {
      display: flex !important;
      justify-content: space-between !important;
      align-items: center !important;
      background: var(--card-bg);
      border-bottom: 1px solid var(--card-border);
      padding: 12px 24px;
      flex-shrink: 0;
    }
    .report-brand {
      display: flex;
      align-items: center;
      gap: 12px;
    }
    .report-brand h1 {
      margin: 0;
      font-size: 1.05rem;
      font-weight: 600;
      letter-spacing: -0.01em;
      color: var(--text-bright);
    }
    .report-badge {
      display: inline-block;
      padding: 2px 7px;
      font-size: 0.68rem;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.04em;
      border-radius: 4px;
      background: #1f2a3c;
      color: #79c0ff;
      border: 1px solid #388bfd80;
    }
    [data-theme="light"] .report-badge {
      background: #ddf4ff;
      color: #0969da;
      border: 1px solid #54aeff66;
    }
    .report-container {
      flex: 1;
      display: flex;
      flex-direction: column;
      padding: 16px 24px 24px;
      max-width: 100%;
      box-sizing: border-box;
      overflow: hidden;
    }
    .comparison-view-container {
      margin-top: 0;
      flex: 1;
      display: flex;
      flex-direction: column;
      background: var(--card-bg);
    }
    .comparison-table-scroll {
      flex: 1;
      overflow-y: auto;
      overflow-x: auto;
    }
    #modal-eval-btn {
      display: none !important;
    }
  </style>
</head>
<body>
  <header>
    <div class="report-brand">
      <h1>Jev Usecase - Ticket System</h1>
      <span class="report-badge">Data Snapshot</span>
    </div>

    <div class="header-actions">
      <button id="theme-toggle" class="theme-toggle-pill" onclick="toggleTheme()" type="button" title="Toggle Light or Dark theme" aria-label="Toggle Light or Dark theme">
        <span class="theme-icon theme-sun" id="theme-sun">
          <svg class="theme-glyph" viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="4"></circle>
            <path d="M12 2v2"></path>
            <path d="M12 20v2"></path>
            <path d="M4.93 4.93l1.41 1.41"></path>
            <path d="M17.66 17.66l1.41 1.41"></path>
            <path d="M2 12h2"></path>
            <path d="M20 12h2"></path>
            <path d="M6.34 17.66l-1.41 1.41"></path>
            <path d="M19.07 4.93l-1.41 1.41"></path>
          </svg>
        </span>
        <span class="theme-icon theme-moon" id="theme-moon">
          <svg class="theme-glyph" viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
          </svg>
        </span>
      </button>
    </div>
  </header>

  <main class="report-container">
    <div class="showcase-action-bar">
      <div class="action-bar-left" style="display: flex; align-items: center; gap: 14px;">
        <span id="report-status-text" style="color: var(--text-bright); font-size: 0.85rem; font-weight: 600;">
          Loading tickets...
        </span>
        <span id="report-export-date" style="color: var(--text-muted); font-size: 0.78rem;"></span>
      </div>
      <div class="action-bar-right" style="display: flex; align-items: center; gap: 14px;">
        <input type="text" id="report-search-input" class="form-control" placeholder="Search tickets..." style="padding: 4px 10px; font-size: 0.8rem; width: 170px;">
        <label style="display: inline-flex; align-items: center; gap: 5px; font-size: 0.8rem; color: var(--text-muted); cursor: pointer; user-select: none;">
          <input type="checkbox" id="filter-diffs-only" style="cursor: pointer;">
          Disagreements only
        </label>
        <span style="color: var(--text-muted); font-size: 0.8rem;">Click a row to select, double-click to open details.</span>
      </div>
    </div>

    <div class="comparison-view-container">
      <div class="comparison-table-scroll">
        <table class="comparison-table" id="comparison-table">
          <colgroup id="comparison-colgroup"></colgroup>
          <thead id="comparison-thead">
            <tr class="group-header-row">
              <th colspan="6" class="group-header-ticket">
                <div class="group-header-content">
                  <span class="group-title">Ticket</span>
                </div>
              </th>
              <th colspan="7" class="group-header-engine-a" id="hdr-engine-a-group">
                <div class="group-header-content">
                  <span class="group-title" id="hdr-engine-a-title">Decision Engine A</span>
                  <span class="group-model-badge" id="hdr-engine-a-model">typesafe/jev</span>
                  <span class="group-median-badge" id="hdr-engine-a-median">median -</span>
                </div>
              </th>
              <th colspan="7" class="group-header-engine-b" id="hdr-engine-b-group">
                <div class="group-header-content">
                  <span class="group-title" id="hdr-engine-b-title">Decision Engine B</span>
                  <span class="group-model-badge" id="hdr-engine-b-model">model</span>
                  <span class="group-median-badge" id="hdr-engine-b-median">median -</span>
                </div>
              </th>
            </tr>
            <tr class="sub-header-row" id="comparison-sub-header-row"></tr>
          </thead>
          <tbody id="comparison-tbody"></tbody>
        </table>
      </div>
    </div>
  </main>

  ${modalMarkup}

  <script id="snapshot-data" type="application/json">
${jsonPayload}
  </script>

  <script>
${standaloneScriptCode}
  </script>
</body>
</html>`;

  const blob = new Blob([htmlContent], { type: 'text/html;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  const dateStr = new Date().toISOString().slice(0, 10);
  a.download = 'ticket-eval-report-' + dateStr + '.html';
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  setTimeout(() => URL.revokeObjectURL(url), 1000);

  if (typeof showToast === 'function') {
    showToast('Exported standalone HTML report.');
  }
}

window.exportEvaluationCSV = exportEvaluationCSV;
window.exportEvaluationHTML = exportEvaluationHTML;
