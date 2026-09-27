// Modal Component: Standalone Controller for Ticket Inspection Window (G2)
let currentModalTicketId = null;

const CANONICAL_MODAL_LABELS = {
  ticket_type: 'Ticket Type',
  technical_domain: 'Technical Domain',
  operational_urgency: 'Operational Urgency',
  security_incident: 'Security Incident',
  target_resolution_group: 'Resolution Group',
  blast_radius: 'Blast Radius'
};

const DEFAULT_MODAL_CRITERIA = [
  { id: 'ticket_type', label: 'Ticket Type' },
  { id: 'technical_domain', label: 'Technical Domain' },
  { id: 'operational_urgency', label: 'Operational Urgency' },
  { id: 'security_incident', label: 'Security Incident' },
  { id: 'target_resolution_group', label: 'Resolution Group' },
  { id: 'blast_radius', label: 'Blast Radius' }
];

function getQuestionModalLabel(q) {
  if (!q) return '';
  if (CANONICAL_MODAL_LABELS[q.id]) {
    return CANONICAL_MODAL_LABELS[q.id];
  }
  if (q.text && q.text.length <= 30 && !q.text.includes('?')) {
    return q.text;
  }
  const words = (q.id || '').split(/[_-]+/).filter(Boolean);
  if (words.length > 0) {
    return words.map(w => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase()).join(' ');
  }
  return q.id || 'Custom';
}

function getActiveModalCriteria() {
  if (Array.isArray(triageCriteria) && triageCriteria.length > 0) {
    return triageCriteria.map(q => ({
      id: q.id,
      label: getQuestionModalLabel(q)
    }));
  }
  return DEFAULT_MODAL_CRITERIA;
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

function formatModalAnswer(critId, val) {
  if (val === undefined || val === null) return '-';
  const s = String(val).trim();
  if (s === '') return '-';
  const pol = parseBinaryPolarity(val);
  if (pol !== null) {
    return pol ? 'Yes' : 'No';
  }
  return s;
}

function openTicketModal(ticket, meta = {}) {
  if (!ticket) return;
  currentModalTicketId = ticket.id;

  const backdrop = document.getElementById('ticket-modal-backdrop');
  if (!backdrop) return;

  // Populate Header
  document.getElementById('modal-ticket-id').innerText = ticket.id;
  document.getElementById('modal-ticket-title').innerText = ticket.summary || 'Untitled Ticket';
  document.getElementById('modal-reporter-name').innerText = ticket.reporter_name || 'Unknown';
  document.getElementById('modal-reporter-dept').innerText = ticket.department || 'General';
  document.getElementById('modal-filed-urgency').innerText = ticket.reported_urgency || ticket.urgency || 'Medium';

  // Populate Left Column
  document.getElementById('modal-desc-box').innerText = ticket.description || 'No description provided.';
  document.getElementById('modal-detail-reporter').innerText = ticket.reporter_name || '-';
  document.getElementById('modal-detail-dept').innerText = ticket.department || '-';
  document.getElementById('modal-detail-asset').innerText = ticket.affected_system || '-';
  document.getElementById('modal-detail-category').innerText = ticket.category || '-';
  document.getElementById('modal-detail-urgency').innerText = ticket.reported_urgency || ticket.urgency || 'Medium';

  // Populate Right Column
  renderModalDecisions(ticket, meta);

  // If the ticket is currently evaluating, ensure comparison timer is running
  if (typeof evaluatingTickets !== 'undefined' && evaluatingTickets[ticket.id]) {
    if (typeof startComparisonTimer === 'function') {
      startComparisonTimer();
    }
  }

  backdrop.classList.add('active');
  document.addEventListener('keydown', handleModalKeyDown);
}

function renderModalDecisions(ticket, meta = {}) {
  const evalSt = (typeof evaluatingTickets !== 'undefined') ? evaluatingTickets[ticket.id] : null;
  const isEvaluatingA = !!(evalSt && evalSt.a);
  const isEvaluatingB = !!(evalSt && evalSt.b);

  const resA = isEvaluatingA ? null : (ticket.triage_a || ticket.triage);
  const resB = isEvaluatingB ? null : ticket.triage_b;

  const nameA = meta.engineAName || (resA && resA.engine_name) || 'Decision Engine A';
  const nameB = meta.engineBName || (resB && resB.engine_name) || 'Decision Engine B';

  const provA = (appConfig.engine_a?.provider || 'jev').toLowerCase();
  const provB = (appConfig.engine_b?.provider || 'openai').toLowerCase();
  const rawModelA = (resA && resA.model) || appConfig.engine_a?.model || '';
  const rawModelB = (resB && resB.model) || appConfig.engine_b?.model || '';

  const modelA = meta.engineAModel || (typeof formatEngineModelBadge === 'function' ? formatEngineModelBadge(provA, rawModelA) : (typeof getEngineModelBadge === 'function' ? getEngineModelBadge('a') : ''));
  const modelB = meta.engineBModel || (typeof formatEngineModelBadge === 'function' ? formatEngineModelBadge(provB, rawModelB) : (typeof getEngineModelBadge === 'function' ? getEngineModelBadge('b') : ''));

  // Confidence in table headers
  const confA = resA && typeof resA.confidence === 'number' && resA.confidence > 0
    ? Math.round(resA.confidence * 100) + '%'
    : '';
  const confB = resB && typeof resB.confidence === 'number' && resB.confidence > 0
    ? Math.round(resB.confidence * 100) + '%'
    : '';

  const thA = document.getElementById('modal-th-engine-a');
  const thB = document.getElementById('modal-th-engine-b');
  if (thA) {
    thA.innerHTML = `
      <div class="modal-th-header-cell">
        <div style="display: flex; align-items: center; gap: 6px; flex-wrap: wrap;">
          <span class="modal-th-engine-name">${escapeHtml(nameA)}</span>
          ${modelA ? `<span class="group-model-badge" style="font-size: 0.68rem; padding: 1px 5px;">${escapeHtml(modelA)}</span>` : ''}
        </div>
        ${confA ? `<span class="modal-th-conf-badge" title="Model confidence score for decisions">${escapeHtml(confA)} confidence</span>` : ''}
      </div>
    `;
  }
  if (thB) {
    thB.innerHTML = `
      <div class="modal-th-header-cell">
        <div style="display: flex; align-items: center; gap: 6px; flex-wrap: wrap;">
          <span class="modal-th-engine-name">${escapeHtml(nameB)}</span>
          ${modelB ? `<span class="group-model-badge" style="font-size: 0.68rem; padding: 1px 5px;">${escapeHtml(modelB)}</span>` : ''}
        </div>
        ${confB ? `<span class="modal-th-conf-badge" title="Model confidence score for decisions">${escapeHtml(confB)} confidence</span>` : ''}
      </div>
    `;
  }

  // Criteria Table Rows across all active questions
  const tbody = document.getElementById('modal-criteria-tbody');
  const critList = getActiveModalCriteria();
  let matched = 0;
  let totalEvaluated = 0;

  const rowsHtml = critList.map(crit => {
    const valA = resA?.answers?.[crit.id];
    const valB = resB?.answers?.[crit.id];

    const hasA = valA !== undefined && valA !== null;
    const hasB = valB !== undefined && valB !== null;

    let isDiff = false;
    if (hasA && hasB) {
      totalEvaluated++;
      if (areAnswersEqual(valA, valB)) {
        matched++;
      } else {
        isDiff = true;
      }
    }

    const rowDiffClass = isDiff ? ' class="row-diff"' : '';
    const cellDiffClass = isDiff ? ' col-diff-val' : '';

    const textA = hasA ? formatModalAnswer(crit.id, valA) : (resA ? '-' : 'Waiting');
    const textB = hasB ? formatModalAnswer(crit.id, valB) : (resB ? '-' : 'Waiting');

    return `
      <tr${rowDiffClass}>
        <td class="modal-criteria-name">${escapeHtml(crit.label)}</td>
        <td class="${cellDiffClass}">${escapeHtml(textA)}</td>
        <td class="${cellDiffClass}">${escapeHtml(textB)}</td>
      </tr>
    `;
  }).join('');

  if (tbody) tbody.innerHTML = rowsHtml;

  // Match Summary and Speedup
  const summaryEl = document.getElementById('modal-match-summary');
  if (summaryEl) {
    if (hasCompletedBoth(resA, resB)) {
      const matchText = `${matched} of ${critList.length} match`;
      let speedupText = '';
      if (resA.latency_ms > 0 && resB.latency_ms > 0) {
        if (resB.latency_ms > resA.latency_ms) {
          const ratio = (resB.latency_ms / resA.latency_ms).toFixed(1);
          speedupText = ` · ${nameA} ${ratio}× faster`;
        } else if (resA.latency_ms > resB.latency_ms) {
          const ratio = (resA.latency_ms / resB.latency_ms).toFixed(1);
          speedupText = ` · ${nameB} ${ratio}× faster`;
        }
      }
      summaryEl.innerText = matchText + speedupText;
    } else if (resA && !resB) {
      summaryEl.innerText = `${nameA} complete · waiting for ${nameB}`;
    } else {
      summaryEl.innerText = 'Triage pending';
    }
  }

  // Response Time Bars on the Same Scale
  const barLabelA = document.getElementById('modal-bar-label-a');
  const barLabelB = document.getElementById('modal-bar-label-b');
  const barFillA = document.getElementById('modal-bar-fill-a');
  const barFillB = document.getElementById('modal-bar-fill-b');
  const barValA = document.getElementById('modal-bar-val-a');
  const barValB = document.getElementById('modal-bar-val-b');

  if (barLabelA) barLabelA.innerText = nameA;
  if (barLabelB) barLabelB.innerText = nameB;

  if (barValA) barValA.classList.remove('time-running');
  if (barValB) barValB.classList.remove('time-running');

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
    if (evalSt && evalSt.b && evalSt.startTime) {
      const elapsedMs = Math.max(0, Date.now() - evalSt.startTime);
      const elapsedSecStr = (elapsedMs / 1000).toFixed(1) + ' s';
      const maxLat = Math.max(latA, elapsedMs);
      const pctA = Math.max(4, Math.round((latA / maxLat) * 100));
      const pctB = Math.max(4, Math.round((elapsedMs / maxLat) * 100));

      if (barFillA) barFillA.style.width = pctA + '%';
      if (barFillB) barFillB.style.width = pctB + '%';
      if (barValA) barValA.innerText = formatLatencyString(latA);
      if (barValB) {
        barValB.innerText = elapsedSecStr;
        barValB.classList.add('time-running');
      }
    } else {
      if (barFillA) barFillA.style.width = '100%';
      if (barValA) barValA.innerText = formatLatencyString(latA);
      if (barFillB) barFillB.style.width = '0%';
      if (barValB) barValB.innerText = resB ? '-' : 'Waiting';
    }
  } else if (evalSt && evalSt.startTime) {
    const elapsedMs = Math.max(0, Date.now() - evalSt.startTime);
    const elapsedSecStr = (elapsedMs / 1000).toFixed(1) + ' s';
    if (evalSt.a) {
      if (barFillA) barFillA.style.width = '100%';
      if (barValA) {
        barValA.innerText = elapsedSecStr;
        barValA.classList.add('time-running');
      }
    }
    if (evalSt.b) {
      if (barFillB) barFillB.style.width = '100%';
      if (barValB) {
        barValB.innerText = elapsedSecStr;
        barValB.classList.add('time-running');
      }
    }
  } else {
    if (barFillA) barFillA.style.width = '0%';
    if (barFillB) barFillB.style.width = '0%';
    if (barValA) barValA.innerText = '-';
    if (barValB) barValB.innerText = '-';
  }

  // Recommended Action Side by Side
  const actHdrA = document.getElementById('modal-action-hdr-a');
  const actHdrB = document.getElementById('modal-action-hdr-b');
  const actBodyA = document.getElementById('modal-action-body-a');
  const actBodyB = document.getElementById('modal-action-body-b');

  if (actHdrA) actHdrA.innerText = nameA;
  if (actHdrB) actHdrB.innerText = nameB;

  const getActionText = (res, prov) => {
    if (prov === 'jev' || (res && res.engine_name && res.engine_name.toLowerCase().includes('jev'))) {
      return 'Jev is not able to generate a response to this question.';
    }
    return res?.recommended_action || (res ? 'No action specified.' : 'Waiting for decision...');
  };

  if (actBodyA) actBodyA.innerText = getActionText(resA, provA);
  if (actBodyB) actBodyB.innerText = getActionText(resB, provB);

  // Evaluate button state
  const evalBtn = document.getElementById('modal-eval-btn');
  if (evalBtn) {
    const isEvaluating = typeof evaluatingTickets !== 'undefined' && evaluatingTickets[ticket.id];
    if (isEvaluating) {
      evalBtn.disabled = true;
      evalBtn.innerText = 'Evaluating...';
      evalBtn.className = 'btn';
    } else {
      evalBtn.disabled = false;
      const isTriaged = (resA != null) || (resB != null);
      evalBtn.innerText = isTriaged ? 'Re-evaluate Ticket' : 'Evaluate Ticket';
      evalBtn.className = isTriaged ? 'btn' : 'btn btn-primary';
    }
  }
}

function triageCurrentModalTicket() {
  if (!currentModalTicketId) return;
  const evalBtn = document.getElementById('modal-eval-btn');
  if (evalBtn) {
    evalBtn.disabled = true;
    evalBtn.innerText = 'Evaluating...';
  }
  if (typeof triageSingleTicket === 'function') {
    triageSingleTicket(currentModalTicketId);
  }
}

function hasCompletedBoth(resA, resB) {
  return resA && resB && resA.answers && resB.answers;
}

function closeTicketModal() {
  const backdrop = document.getElementById('ticket-modal-backdrop');
  if (backdrop) backdrop.classList.remove('active');
  currentModalTicketId = null;
  document.removeEventListener('keydown', handleModalKeyDown);
}

function handleBackdropClick(e) {
  if (e.target && e.target.id === 'ticket-modal-backdrop') {
    closeTicketModal();
  }
}

function handleModalKeyDown(e) {
  if (e.key === 'Escape' || e.key === 'Esc') {
    closeTicketModal();
  }
}

function updateTicketModalIfOpen(updatedTicket, meta = {}) {
  if (currentModalTicketId && updatedTicket && currentModalTicketId === updatedTicket.id) {
    renderModalDecisions(updatedTicket, meta);
  }
}

function updateModalTimer(evalSt) {
  if (!currentModalTicketId || !evalSt || !evalSt.startTime) return;
  const now = Date.now();
  const elapsedMs = Math.max(0, now - evalSt.startTime);
  const elapsedSecStr = (elapsedMs / 1000).toFixed(1) + ' s';

  const ticket = dataset.find(item => item.id === currentModalTicketId);
  const resA = evalSt.a ? null : (ticket?.triage_a || ticket?.triage);
  const resB = evalSt.b ? null : ticket?.triage_b;

  const latA = (resA && typeof resA.latency_ms === 'number' && resA.latency_ms > 0) ? resA.latency_ms : null;
  const latB = (resB && typeof resB.latency_ms === 'number' && resB.latency_ms > 0) ? resB.latency_ms : null;

  const barFillA = document.getElementById('modal-bar-fill-a');
  const barFillB = document.getElementById('modal-bar-fill-b');
  const barValA = document.getElementById('modal-bar-val-a');
  const barValB = document.getElementById('modal-bar-val-b');

  if (evalSt.b) {
    if (barValB) {
      barValB.innerText = elapsedSecStr;
      barValB.classList.add('time-running');
    }
    if (latA !== null && latA > 0) {
      const maxLat = Math.max(latA, elapsedMs);
      const pctA = Math.max(4, Math.round((latA / maxLat) * 100));
      const pctB = Math.max(4, Math.round((elapsedMs / maxLat) * 100));
      if (barFillA) barFillA.style.width = pctA + '%';
      if (barFillB) barFillB.style.width = pctB + '%';
    } else {
      if (barFillB) barFillB.style.width = '100%';
    }
  }

  if (evalSt.a) {
    if (barValA) {
      barValA.innerText = elapsedSecStr;
      barValA.classList.add('time-running');
    }
    if (latB !== null && latB > 0) {
      const maxLat = Math.max(latB, elapsedMs);
      const pctA = Math.max(4, Math.round((latB / maxLat) * 100));
      const pctB = Math.max(4, Math.round((elapsedMs / maxLat) * 100));
      if (barFillA) barFillA.style.width = pctA + '%';
      if (barFillB) barFillB.style.width = pctB + '%';
    } else {
      if (barFillA) barFillA.style.width = '100%';
    }
  }
}
