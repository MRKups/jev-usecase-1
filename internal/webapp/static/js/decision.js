// Page Module: Comparison View Controller for 3-Part Showcase Grid
let comparisonTimerInterval = null;
let isTriageAllRunning = false;
let triageAbortController = null;

const CANONICAL_TABLE_HEADERS = {
  ticket_type: 'Type',
  technical_domain: 'Domain',
  operational_urgency: 'Urgency',
  security_incident: 'Security',
  target_resolution_group: 'Group',
  blast_radius: 'Blast radius'
};

const DEFAULT_CRITERIA = [
  { id: 'ticket_type', text: 'Is this ticket an Incident or a Service Request?' },
  { id: 'technical_domain', text: 'What is the primary technical domain of this ticket?' },
  { id: 'operational_urgency', text: 'What is the verified operational urgency level based on technical impact?' },
  { id: 'security_incident', text: 'Does this issue indicate a security breach, unauthorized access, or policy violation?' },
  { id: 'target_resolution_group', text: 'Which support tier or engineering team should resolve this issue?' },
  { id: 'blast_radius', text: 'Does this issue affect an isolated user or indicate a broader systemic outage?' }
];

function getActiveCriteria() {
  if (Array.isArray(triageCriteria) && triageCriteria.length > 0) {
    return triageCriteria;
  }
  return DEFAULT_CRITERIA;
}

function getQuestionTableHeader(q) {
  if (!q) return '';
  if (CANONICAL_TABLE_HEADERS[q.id]) {
    return CANONICAL_TABLE_HEADERS[q.id];
  }
  const words = (q.id || '').split(/[_-]+/).filter(Boolean);
  if (words.length > 0) {
    const formatted = words.map(w => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase()).join(' ');
    return formatted.length > 14 ? formatted.slice(0, 13) + '…' : formatted;
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

function updateComparisonTableStructure(tableQuestions) {
  const colSpanCount = tableQuestions.length + 1;
  const hdrA = document.getElementById('hdr-engine-a-group');
  const hdrB = document.getElementById('hdr-engine-b-group');
  if (hdrA) hdrA.colSpan = colSpanCount;
  if (hdrB) hdrB.colSpan = colSpanCount;

  const colgroup = document.getElementById('comparison-colgroup');
  if (colgroup) {
    let colHtml = `
      <!-- Ticket columns (~33.3%) -->
      <col style="width: 28px;">
      <col style="width: 60px;">
      <col style="width: 220px;">
      <col style="width: 120px;">
      <col style="width: 65px;">
      <col style="width: 50px;">
    `;

    // Decision Engine A columns
    tableQuestions.forEach(q => {
      colHtml += `<col style="width: ${getQuestionColumnWidth(q.id)}px;">\n`;
    });
    colHtml += `<col style="width: 65px;">\n`;

    // Decision Engine B columns
    tableQuestions.forEach(q => {
      colHtml += `<col style="width: ${getQuestionColumnWidth(q.id)}px;">\n`;
    });
    colHtml += `<col style="width: 65px;">\n`;

    colgroup.innerHTML = colHtml;
  }

  const subHeaderRow = document.getElementById('comparison-sub-header-row');
  if (subHeaderRow) {
    let subHtml = `
      <!-- Ticket -->
      <th></th>
      <th>ID</th>
      <th>Title</th>
      <th>Filed as</th>
      <th>Urgency</th>
      <th class="col-border-ticket-end" style="text-align: center;">Agree</th>
    `;

    // Decision Engine A
    tableQuestions.forEach(q => {
      subHtml += `<th>${escapeHtml(getQuestionTableHeader(q))}</th>\n`;
    });
    subHtml += `<th class="col-border-engine-a-end" style="text-align: right;">Time</th>\n`;

    // Decision Engine B
    tableQuestions.forEach(q => {
      subHtml += `<th>${escapeHtml(getQuestionTableHeader(q))}</th>\n`;
    });
    subHtml += `<th style="text-align: right;">Time</th>\n`;

    subHeaderRow.innerHTML = subHtml;
  }
}

function getEngineDisplayName(engineKey) {
  if (engineKey === 'a') {
    if (appConfig.engine_a && appConfig.engine_a.name) return appConfig.engine_a.name;
    return 'Decision Engine A';
  }
  if (engineKey === 'b') {
    if (appConfig.engine_b && appConfig.engine_b.name) return appConfig.engine_b.name;
    return 'Decision Engine B';
  }
  return 'Decision Engine';
}

function getEngineModelBadge(engineKey) {
  const cfg = (engineKey === 'a' ? appConfig.engine_a : appConfig.engine_b) || {};
  const prov = (cfg.provider || (engineKey === 'a' ? 'jev' : 'openai')).toLowerCase();
  if (prov === 'disabled' || prov === 'none') return '';

  let model = (cfg.model || '').trim();

  // Check if any triaged ticket in dataset has a specific model reported by the engine
  if (Array.isArray(dataset)) {
    for (const t of dataset) {
      const res = engineKey === 'a' ? (t.triage_a || t.triage) : t.triage_b;
      if (res && res.model && res.model.trim()) {
        model = res.model.trim();
        break;
      }
    }
  }

  if (typeof formatEngineModelBadge === 'function') {
    return formatEngineModelBadge(prov, model);
  }

  // Fallback if formatEngineModelBadge is undefined
  if (prov === 'jev' || prov === 'typesafe') {
    if (!model || model === 'jev-latest' || model === 'jev') return 'typesafe/jev';
    if (model.toLowerCase().startsWith('typesafe/')) return model.toLowerCase();
    return 'typesafe/' + model;
  }
  if (model) {
    if (model.includes('/')) return model;
    if (prov === 'gemini') return 'google/' + model;
    return `${prov}/${model}`;
  }
  if (prov === 'gemini') return 'google/gemini';
  return prov;
}

function getEngineModel(engineKey) {
  return getEngineModelBadge(engineKey);
}

function calculateMedian(numbers) {
  if (!numbers || numbers.length === 0) return null;
  const sorted = [...numbers].sort((a, b) => a - b);
  const mid = Math.floor(sorted.length / 2);
  if (sorted.length % 2 !== 0) {
    return sorted[mid];
  }
  return Math.round((sorted[mid - 1] + sorted[mid]) / 2);
}

function formatLatencyString(ms) {
  if (ms === null || ms === undefined || ms < 0) return '-';
  if (ms < 1000) return ms + ' ms';
  return (ms / 1000).toFixed(1) + ' s';
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
  if (s.endsWith('.0')) {
    s = s.slice(0, -2);
  }
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
    if (s === 'Access & Identity') return 'Access';
    return s;
  }

  if (key === 'target_resolution_group') {
    if (s === 'Service Desk' || s === 'Service Desk Tier 1') return 'Service Desk';
    if (s === 'Identity & Access Management') return 'IAM';
    if (s === 'Site Reliability Engineering') return 'SRE';
    if (s === 'Network Operations' || s === 'Network Engineering') return 'Network Ops';
    if (s === 'SecOps') return 'SecOps';
    return s;
  }

  if (key === 'blast_radius') {
    if (s.toLowerCase().includes('single')) return 'Single User';
    if (s.toLowerCase().includes('multiple') || s.toLowerCase().includes('department')) return 'Multiple Users';
    if (s.toLowerCase().includes('company')) return 'Company-wide';
    return s;
  }

  const pol = parseBinaryPolarity(val);
  if (pol !== null) {
    return pol ? 'Yes' : 'No';
  }

  return s;
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
  if (polA !== null && polB !== null) {
    return polA === polB;
  }
  return normalizeAnswerValue(valA) === normalizeAnswerValue(valB);
}

function updateShowcaseView() {
  const tbody = document.getElementById('comparison-tbody');
  if (!tbody) return;

  const activeCriteria = getActiveCriteria();
  const tableQuestions = activeCriteria.slice(0, 6);

  updateComparisonTableStructure(tableQuestions);

  const total = Array.isArray(dataset) ? dataset.length : 0;
  const statusEl = document.getElementById('showcase-status-text');
  if (statusEl) {
    if (isTriageAllRunning) {
      const runningCount = Object.keys(evaluatingTickets).length;
      const completedCount = Array.isArray(dataset) ? dataset.filter(t => (t.triage_a || t.triage) != null || t.triage_b != null).length : 0;
      const evalMode = appConfig.evaluation_mode || appConfig.execution_mode;
      if (evalMode === 'serial') {
        statusEl.innerText = `Evaluating: ${completedCount} of ${total} complete (serial)`;
      } else {
        statusEl.innerText = `Evaluating: ${completedCount} of ${total} complete · ${runningCount} in flight`;
      }
    } else {
      statusEl.innerText = `${total} tickets loaded`;
    }
  }

  const exportBtn = document.getElementById('btn-export-csv');
  if (exportBtn) {
    exportBtn.disabled = total === 0;
  }
  const exportHtmlBtn = document.getElementById('btn-export-html');
  if (exportHtmlBtn) {
    exportHtmlBtn.disabled = total === 0;
  }

  const totalCols = 6 + (tableQuestions.length + 1) * 2;

  if (!Array.isArray(dataset) || dataset.length === 0) {
    tbody.innerHTML = `
      <tr>
        <td colspan="${totalCols}" style="text-align: center; color: var(--text-muted); padding: 40px;">
          No tickets generated yet. Generate tickets on page 2.
        </td>
      </tr>
    `;
    updateGroupHeaders([], []);
    return;
  }

  const latenciesA = [];
  const latenciesB = [];

  dataset.forEach(t => {
    const resA = t.triage_a || t.triage;
    const resB = t.triage_b;
    if (resA && typeof resA.latency_ms === 'number' && resA.latency_ms > 0) {
      latenciesA.push(resA.latency_ms);
    }
    if (resB && typeof resB.latency_ms === 'number' && resB.latency_ms > 0) {
      latenciesB.push(resB.latency_ms);
    }
  });

  updateGroupHeaders(latenciesA, latenciesB);

  tbody.innerHTML = dataset.map(t => {
    return renderComparisonRow(t, tableQuestions, activeCriteria);
  }).join('');

  if (selectedTicketId) {
    const selRow = document.getElementById('row-ticket-' + cleanId(selectedTicketId));
    if (selRow) selRow.classList.add('row-selected');
  }
  updateSelectedActionButton(selectedTicketId);

  // Update modal if currently viewing this ticket
  if (currentModalTicketId) {
    const activeTicket = Array.isArray(dataset) ? dataset.find(item => item.id === currentModalTicketId) : null;
    if (activeTicket) {
      updateTicketModalIfOpen(activeTicket, {
        engineAName: getEngineDisplayName('a'),
        engineBName: getEngineDisplayName('b'),
        engineAModel: getEngineModelBadge('a'),
        engineBModel: getEngineModelBadge('b')
      });
    }
  }
}

function updateGroupHeaders(latenciesA, latenciesB) {
  const nameA = getEngineDisplayName('a');
  const nameB = getEngineDisplayName('b');
  const modelA = getEngineModelBadge('a');
  const modelB = getEngineModelBadge('b');

  const titleAEl = document.getElementById('hdr-engine-a-title');
  const medianAEl = document.getElementById('hdr-engine-a-median');
  const modelAEl = document.getElementById('hdr-engine-a-model');
  const titleBEl = document.getElementById('hdr-engine-b-title');
  const medianBEl = document.getElementById('hdr-engine-b-median');
  const modelBEl = document.getElementById('hdr-engine-b-model');

  if (titleAEl) titleAEl.innerText = nameA;
  if (titleBEl) titleBEl.innerText = nameB;

  const medA = calculateMedian(latenciesA);
  const medB = calculateMedian(latenciesB);

  if (medianAEl) {
    medianAEl.innerText = medA !== null ? `median ${formatLatencyString(medA)}` : 'median -';
  }

  if (medianBEl) {
    medianBEl.innerText = medB !== null ? `median ${formatLatencyString(medB)}` : 'median -';
  }

  if (modelAEl) {
    if (modelA) {
      modelAEl.innerText = modelA;
      modelAEl.style.display = 'inline-block';
    } else {
      modelAEl.style.display = 'none';
    }
  }

  if (modelBEl) {
    if (modelB) {
      modelBEl.innerText = modelB;
      modelBEl.style.display = 'inline-block';
    } else {
      modelBEl.style.display = 'none';
    }
  }
}

function renderComparisonRow(t, tableQuestions, activeCriteria) {
  if (!tableQuestions) tableQuestions = getActiveCriteria().slice(0, 6);
  if (!activeCriteria) activeCriteria = getActiveCriteria();

  const resA = t.triage_a || t.triage;
  const resB = t.triage_b;
  const evalSt = evaluatingTickets[t.id];

  const isEvaluatingA = !!(evalSt && evalSt.a);
  const isEvaluatingB = !!(evalSt && evalSt.b);
  const isRunning = isEvaluatingA || isEvaluatingB;

  const hasA = resA && resA.answers && !isEvaluatingA;
  const hasB = resB && resB.answers && !isEvaluatingB;

  // Status Dot
  let dotSymbol = '○';
  let dotClass = 'status-dot-waiting';
  if (hasA && hasB) {
    dotSymbol = '●';
    dotClass = 'status-dot-done';
  } else if (isRunning || hasA || hasB) {
    dotSymbol = '◐';
    dotClass = 'status-dot-running';
  }

  // Agreement Score across all active criteria
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

    agreeText = `${matched}/${totalCrit}`;
    if (matched === totalCrit && totalCrit > 0) {
      agreeClass = 'agree-full';
    } else {
      agreeClass = 'agree-diff';
    }
  }

  const isSelected = t.id === selectedTicketId ? ' row-selected' : '';
  const isFlash = t.id === latestFlashTicketId ? ' row-flash' : '';

  // Helpers for Engine A and B cells
  const renderCellA = (k, extraClass = '') => {
    if (!hasA) return `<td class="${extraClass}"><span class="faint-dot">·</span></td>`;
    const val = resA.answers[k];
    if (val === undefined || val === null || val === '') {
      return `<td class="${extraClass}"><span class="faint-dot">·</span></td>`;
    }
    const shortVal = shortenCellValue(k, val);
    const isDiff = diffs[k];

    const classes = [];
    if (extraClass) classes.push(extraClass);
    if (isDiff) classes.push('cell-diff');

    const classAttr = classes.length > 0 ? ` class="${classes.join(' ')}"` : '';
    return `<td${classAttr}>${escapeHtml(shortVal)}</td>`;
  };

  const renderCellB = (k, extraClass = '') => {
    if (!hasB) return `<td class="${extraClass}"><span class="faint-dot">·</span></td>`;
    const val = resB.answers[k];
    if (val === undefined || val === null || val === '') {
      return `<td class="${extraClass}"><span class="faint-dot">·</span></td>`;
    }
    const shortVal = shortenCellValue(k, val);
    const isDiff = diffs[k];

    const classes = [];
    if (extraClass) classes.push(extraClass);
    if (isDiff) classes.push('cell-diff');

    const classAttr = classes.length > 0 ? ` class="${classes.join(' ')}"` : '';
    return `<td${classAttr}>${escapeHtml(shortVal)}</td>`;
  };

  let timeA = '<span class="faint-dot">·</span>';
  if (isEvaluatingA && evalSt.startTime) {
    const elapsedSec = ((Date.now() - evalSt.startTime) / 1000).toFixed(1);
    timeA = `<span class="time-running" id="timer-a-${cleanId(t.id)}">${elapsedSec} s</span>`;
  } else if (resA && resA.latency_ms > 0) {
    timeA = formatLatencyString(resA.latency_ms);
  }

  let timeB = '<span class="faint-dot">·</span>';
  if (isEvaluatingB && evalSt.startTime) {
    const elapsedSec = ((Date.now() - evalSt.startTime) / 1000).toFixed(1);
    timeB = `<span class="time-running" id="timer-b-${cleanId(t.id)}">${elapsedSec} s</span>`;
  } else if (resB && resB.latency_ms > 0) {
    timeB = formatLatencyString(resB.latency_ms);
  }

  const urgency = t.reported_urgency || t.urgency || 'Medium';

  const cellsA = tableQuestions.map(q => renderCellA(q.id)).join('');
  const cellsB = tableQuestions.map(q => renderCellB(q.id)).join('');

  return `
    <tr id="row-ticket-${cleanId(t.id)}" class="comp-ticket-row${isSelected}${isFlash}" onclick="handleTicketRowClick('${t.id}')" ondblclick="handleTicketRowDblClick('${t.id}')">
      <!-- Ticket -->
      <td class="cell-dot"><span class="status-dot-icon ${dotClass}">${dotSymbol}</span></td>
      <td class="cell-id">${t.id}</td>
      <td class="cell-title" title="${escapeHtml(t.summary)}">${escapeHtml(t.summary)}</td>
      <td class="cell-filed-as">${escapeHtml(t.category || '-')}</td>
      <td class="cell-filed-urgency">${escapeHtml(urgency)}</td>
      <td class="cell-agree col-border-ticket-end ${agreeClass}">${agreeText}</td>

      <!-- Decision Engine A -->
      ${cellsA}
      <td class="cell-time col-border-engine-a-end" id="cell-time-a-${cleanId(t.id)}">${timeA}</td>

      <!-- Decision Engine B -->
      ${cellsB}
      <td class="cell-time" id="cell-time-b-${cleanId(t.id)}">${timeB}</td>
    </tr>
  `;
}

function handleTicketRowClick(id) {
  selectedTicketId = id;
  document.querySelectorAll('.comp-ticket-row').forEach(r => r.classList.remove('row-selected'));
  const row = document.getElementById('row-ticket-' + cleanId(id));
  if (row) row.classList.add('row-selected');
  updateSelectedActionButton(id);
}

function updateSelectedActionButton(id) {
  const btn = document.getElementById('btn-eval-selected');
  if (!btn) return;
  if (!id) {
    btn.disabled = true;
    btn.innerText = 'Evaluate Selected';
    return;
  }
  if (isTriageAllRunning) {
    btn.disabled = true;
    return;
  }
  const isEvaluating = typeof evaluatingTickets !== 'undefined' && evaluatingTickets[id];
  if (isEvaluating) {
    btn.disabled = true;
    btn.innerText = `Evaluating ${id}...`;
    return;
  }
  const ticket = Array.isArray(dataset) ? dataset.find(item => item.id === id) : null;
  const isTriaged = ticket && ((ticket.triage_a || ticket.triage) != null || ticket.triage_b != null);
  btn.disabled = false;
  btn.innerText = isTriaged ? `Re-evaluate ${id}` : `Evaluate ${id}`;
}

function triageSelectedTicket() {
  if (!selectedTicketId) {
    showToast('Select a ticket first by clicking a row.', 4000);
    return;
  }
  triageSingleTicket(selectedTicketId);
}

function handleTicketRowDblClick(id) {
  handleTicketRowClick(id);

  const ticket = Array.isArray(dataset) ? dataset.find(item => item.id === id) : null;
  if (!ticket) return;

  openTicketModal(ticket, {
    engineAName: getEngineDisplayName('a'),
    engineBName: getEngineDisplayName('b'),
    engineAModel: getEngineModelBadge('a'),
    engineBModel: getEngineModelBadge('b')
  });
}

// Keyboard shortcuts for ticket navigation and evaluation
document.addEventListener('keydown', (e) => {
  if (e.target && (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA')) return;
  const showcasePage = document.getElementById('page-showcase');
  if (!showcasePage || !showcasePage.classList.contains('active')) return;
  if (typeof currentModalTicketId !== 'undefined' && currentModalTicketId) return;

  if (e.key === 'e' || e.key === 'E') {
    if (selectedTicketId && !isTriageAllRunning) {
      e.preventDefault();
      triageSelectedTicket();
    }
  } else if (e.key === 'Enter') {
    if (selectedTicketId) {
      e.preventDefault();
      handleTicketRowDblClick(selectedTicketId);
    }
  } else if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    if (!Array.isArray(dataset) || dataset.length === 0) return;
    e.preventDefault();
    let currIdx = dataset.findIndex(t => t.id === selectedTicketId);
    if (e.key === 'ArrowDown') {
      currIdx = (currIdx + 1) % dataset.length;
    } else {
      currIdx = currIdx <= 0 ? dataset.length - 1 : currIdx - 1;
    }
    const nextTicket = dataset[currIdx];
    if (nextTicket) {
      handleTicketRowClick(nextTicket.id);
      const row = document.getElementById('row-ticket-' + cleanId(nextTicket.id));
      if (row && typeof row.scrollIntoView === 'function') {
        row.scrollIntoView({ block: 'nearest' });
      }
    }
  }
});

// Timer for in-flight evaluations
function startComparisonTimer() {
  if (comparisonTimerInterval) return;
  comparisonTimerInterval = setInterval(() => {
    let hasRunning = false;
    const now = Date.now();
    Object.keys(evaluatingTickets).forEach(id => {
      const st = evaluatingTickets[id];
      if (st && st.startTime && (st.a || st.b)) {
        hasRunning = true;
        const elapsedSec = ((now - st.startTime) / 1000).toFixed(1);

        if (st.b) {
          const timerBEl = document.getElementById('timer-b-' + cleanId(id));
          if (timerBEl) {
            timerBEl.innerText = `${elapsedSec} s`;
          }
        }

        if (st.a) {
          const timerAEl = document.getElementById('timer-a-' + cleanId(id));
          if (timerAEl) {
            timerAEl.innerText = `${elapsedSec} s`;
          }
        }

        // Live update modal timer if this ticket is currently open in modal
        if (typeof updateModalTimer === 'function' && typeof currentModalTicketId !== 'undefined' && currentModalTicketId === id) {
          updateModalTimer(st);
        }
      }
    });
    if (!hasRunning && comparisonTimerInterval) {
      clearInterval(comparisonTimerInterval);
      comparisonTimerInterval = null;
    }
  }, 100);
}

async function triageSingleTicket(id) {
  const isEngineAActive = currentEngineAProvider !== 'disabled' && currentEngineAProvider !== 'none';
  const isEngineBActive = currentEngineBProvider !== 'disabled' && currentEngineBProvider !== 'none';
  if (!isEngineAActive && !isEngineBActive) {
    showToast('No Decision Engine is active. Enable Engine A or B on the Configuration page.', 5000);
    return;
  }

  // Clear previous evaluation on the client for active engines
  const ticket = Array.isArray(dataset) ? dataset.find(item => item.id === id) : null;
  if (ticket) {
    if (isEngineAActive) ticket.triage_a = null;
    if (isEngineBActive) ticket.triage_b = null;
  }

  evaluatingTickets[id] = {
    a: isEngineAActive,
    b: isEngineBActive,
    startTime: Date.now()
  };
  startComparisonTimer();
  updateShowcaseView();

  if (typeof currentModalTicketId !== 'undefined' && currentModalTicketId === id && ticket) {
    renderModalDecisions(ticket, {
      engineAName: getEngineDisplayName('a'),
      engineBName: getEngineDisplayName('b')
    });
  }

  try {
    const res = await fetch('/api/triage/' + encodeURIComponent(id), { method: 'POST' });
    if (!res.ok) {
      const errText = await res.text();
      throw new Error(errText);
    }
    const updated = await res.json();
    delete evaluatingTickets[id];
    if (Array.isArray(dataset)) {
      const idx = dataset.findIndex(t => t.id === id);
      if (idx !== -1) {
        dataset[idx] = updated;
        updateShowcaseView();
        if (typeof currentModalTicketId !== 'undefined' && currentModalTicketId === id) {
          renderModalDecisions(updated, {
            engineAName: getEngineDisplayName('a'),
            engineBName: getEngineDisplayName('b')
          });
        }
      }
    }
  } catch (err) {
    delete evaluatingTickets[id];
    updateShowcaseView();
    showToast('Triage failed: ' + err.message, 8000);
  }
}

async function runTriageAll() {
  const isEngineAActive = currentEngineAProvider !== 'disabled' && currentEngineAProvider !== 'none';
  const isEngineBActive = currentEngineBProvider !== 'disabled' && currentEngineBProvider !== 'none';
  if (!isEngineAActive && !isEngineBActive) {
    showToast('No Decision Engine is active. Enable Engine A or B on the Configuration page.', 5000);
    return;
  }

  isTriageAllRunning = true;
  if (typeof ensureSSEConnected === 'function') {
    ensureSSEConnected();
  }
  triageAbortController = new AbortController();

  const btn = document.getElementById('btn-run-all-triage');
  const btnCancel = document.getElementById('btn-cancel-triage');
  const btnEvalSelected = document.getElementById('btn-eval-selected');

  if (btn) {
    btn.disabled = true;
    btn.innerText = 'Evaluating tickets...';
  }
  if (btnCancel) {
    btnCancel.style.display = 'inline-flex';
    btnCancel.disabled = false;
    btnCancel.innerText = 'Cancel';
  }
  if (btnEvalSelected) {
    btnEvalSelected.disabled = true;
  }

  const statusEl = document.getElementById('showcase-status-text');
  if (statusEl) {
    statusEl.innerText = `Evaluating ${dataset ? dataset.length : 0} tickets...`;
  }

  const triagePollTimer = setInterval(async () => {
    if (!isTriageAllRunning) {
      clearInterval(triagePollTimer);
      return;
    }
    try {
      const res = await fetch('/api/dataset');
      if (!res.ok) return;
      const fresh = await res.json();
      if (!Array.isArray(fresh)) return;
      dataset = fresh;
      if (typeof updateShowcaseView === 'function') updateShowcaseView();
    } catch (_) {}
  }, 1000);

  try {
    const res = await fetch('/api/triage/run', {
      method: 'POST',
      signal: triageAbortController.signal
    });
    if (!res.ok) {
      const errText = await res.text();
      throw new Error(errText);
    }
    const data = await res.json();
    Object.keys(evaluatingTickets).forEach(k => delete evaluatingTickets[k]);
    await loadDataset();

    if (data.cancelled) {
      showToast(`Triage cancelled. ${data.triaged_count} tickets evaluated.`, 8000);
    } else {
      const nameA = getEngineDisplayName('a');
      const nameB = getEngineDisplayName('b');

      if (data.has_engine_a && data.has_engine_b) {
        showToast(`Dual triage complete for ${data.triaged_count} tickets. ${nameA}: ${data.avg_latency_ms} ms avg, ${nameB}: ${data.avg_latency_b_ms} ms avg.`);
      } else if (data.has_engine_b) {
        showToast(`Triaged ${data.triaged_count} tickets with ${nameB} in ${data.avg_latency_b_ms} ms average latency.`);
      } else {
        showToast(`Triaged ${data.triaged_count} tickets with ${nameA} in ${data.avg_latency_ms} ms average latency.`);
      }
    }
  } catch (err) {
    Object.keys(evaluatingTickets).forEach(k => delete evaluatingTickets[k]);
    await loadDataset();
    if (err.name === 'AbortError') {
      showToast('Triage run cancelled.', 8000);
    } else {
      showToast('Batch triage failed: ' + err.message, 8000);
    }
  } finally {
    clearInterval(triagePollTimer);
    isTriageAllRunning = false;
    triageAbortController = null;
    if (btn) {
      btn.disabled = false;
      btn.innerText = 'Evaluate All Tickets';
    }
    if (btnCancel) {
      btnCancel.style.display = 'none';
      btnCancel.disabled = false;
      btnCancel.innerText = 'Cancel';
    }
    updateSelectedActionButton(selectedTicketId);
    updateShowcaseView();
  }
}

async function cancelTriageRun() {
  const btnCancel = document.getElementById('btn-cancel-triage');
  if (btnCancel) {
    btnCancel.disabled = true;
    btnCancel.innerText = 'Cancelling...';
  }
  try {
    await fetch('/api/triage/cancel', { method: 'POST' });
  } catch (err) {
    console.error('Cancel triage failed:', err);
    if (triageAbortController) {
      triageAbortController.abort();
    }
  }
}
