// Shared application state across pages
let appConfig = {};
let dataset = [];
let selectedTicketId = null;
let triageCriteria = [];
let currentEngineAProvider = 'jev';
let currentEngineBProvider = 'openai';
let isGenerating = false;
let currentGenCount = 0;
let targetGenCount = 0;
let genAbortController = null;
let latestFlashTicketId = null;
let flashTimer = null;
const evaluatingTickets = {};

function cleanId(id) {
  if (!id) return '';
  return String(id).replace(/[^a-zA-Z0-9_-]/g, '');
}

function triggerRowFlash(ticketId) {
  latestFlashTicketId = ticketId;
  if (flashTimer) clearTimeout(flashTimer);
  flashTimer = setTimeout(() => {
    latestFlashTicketId = null;
    const dsRow = document.getElementById('row-dataset-' + cleanId(ticketId));
    if (dsRow) dsRow.classList.remove('row-flash');
    const scRow = document.getElementById('row-showcase-' + cleanId(ticketId));
    if (scRow) scRow.classList.remove('row-flash');
    const compRow = document.getElementById('row-ticket-' + cleanId(ticketId));
    if (compRow) compRow.classList.remove('row-flash');
  }, 3500);
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

    function formatEngineModelBadge(prov, model) {
      prov = (prov || '').toLowerCase();
      model = (model || '').trim();
      if (prov === 'disabled' || prov === 'none') return '';

      if (prov === 'jev' || prov === 'typesafe') {
        if (!model || model === 'jev-latest' || model === 'jev') {
          return 'typesafe/jev';
        }
        if (model.toLowerCase().startsWith('typesafe/')) {
          return model.toLowerCase();
        }
        return 'typesafe/' + model;
      }

      if (model) {
        if (model.includes('/')) {
          return model;
        }
        if (prov === 'gemini') {
          return 'google/' + model;
        }
        return `${prov}/${model}`;
      }

      if (prov === 'gemini') return 'google/gemini';
      return prov;
    }

    function showToast(message, durationMs = 10000) {
      let container = document.getElementById('toast-container');
      if (!container) {
        container = document.createElement('div');
        container.id = 'toast-container';
        container.className = 'toast-container';
        document.body.appendChild(container);
      }

      const toast = document.createElement('div');
      toast.className = 'toast';
      toast.innerHTML = `
        <div class="toast-icon">
          <svg viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"></polyline></svg>
        </div>
        <div class="toast-message">${escapeHtml(message)}</div>
        <button class="toast-close" title="Dismiss" aria-label="Close">&times;</button>
      `;

      let timer = null;
      const dismiss = () => {
        if (timer) clearTimeout(timer);
        toast.classList.add('toast-hide');
        setTimeout(() => {
          if (toast.parentNode) {
            toast.parentNode.removeChild(toast);
          }
        }, 300);
      };

      const closeBtn = toast.querySelector('.toast-close');
      if (closeBtn) {
        closeBtn.onclick = (e) => {
          e.stopPropagation();
          dismiss();
        };
      }

      container.appendChild(toast);

      if (durationMs > 0) {
        timer = setTimeout(dismiss, durationMs);
      }
    }


    function switchPage(pageId) {
      document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
      document.querySelectorAll('.page').forEach(pg => pg.classList.remove('active'));

      if (pageId === 'config') {
        document.querySelectorAll('.tab-btn')[0].classList.add('active');
        document.getElementById('page-config').classList.add('active');
      } else if (pageId === 'dataset') {
        document.querySelectorAll('.tab-btn')[1].classList.add('active');
        document.getElementById('page-dataset').classList.add('active');
      } else if (pageId === 'criteria') {
        document.querySelectorAll('.tab-btn')[2].classList.add('active');
        document.getElementById('page-criteria').classList.add('active');
        if (!triageCriteria || triageCriteria.length === 0) {
          loadCriteria();
        } else {
          renderCriteriaCards();
        }
      } else if (pageId === 'showcase') {
        document.querySelectorAll('.tab-btn')[3].classList.add('active');
        document.getElementById('page-showcase').classList.add('active');
        updateShowcaseView();
      }
    }


    // SSE Stream for real-time updates
    let evtSource = null;
    let sseReconnectTimer = null;

    function handleSSEMessage(rawData) {
      if (!rawData) return;
      let item;
      try {
        item = JSON.parse(rawData);
      } catch (err) {
        return;
      }
      if (!item || !item.id) return;
      if (!Array.isArray(dataset)) dataset = [];

      try {
        const idx = dataset.findIndex(t => t.id === item.id);
        if (idx !== -1) {
          dataset[idx] = item;
          if (item.status === 'evaluating') {
            const isEngineAActive = typeof currentEngineAProvider !== 'undefined' && currentEngineAProvider !== 'disabled' && currentEngineAProvider !== 'none';
            const isEngineBActive = typeof currentEngineBProvider !== 'undefined' && currentEngineBProvider !== 'disabled' && currentEngineBProvider !== 'none';
            evaluatingTickets[item.id] = {
              a: isEngineAActive && item.triage_a == null,
              b: isEngineBActive && item.triage_b == null,
              startTime: Date.now()
            };
            if (typeof startComparisonTimer === 'function') startComparisonTimer();
          } else if (evaluatingTickets[item.id]) {
            if (item.triage_a != null) evaluatingTickets[item.id].a = false;
            if (item.triage_b != null) evaluatingTickets[item.id].b = false;
            if (!evaluatingTickets[item.id].a && !evaluatingTickets[item.id].b) {
              delete evaluatingTickets[item.id];
            }
          }
          if (typeof updateTicketModalIfOpen === 'function') {
            updateTicketModalIfOpen(item, {
              engineAName: typeof getEngineDisplayName === 'function' ? getEngineDisplayName('a') : 'Decision Engine A',
              engineBName: typeof getEngineDisplayName === 'function' ? getEngineDisplayName('b') : 'Decision Engine B',
              engineAModel: typeof getEngineModelBadge === 'function' ? getEngineModelBadge('a') : '',
              engineBModel: typeof getEngineModelBadge === 'function' ? getEngineModelBadge('b') : ''
            });
          }
        } else {
          dataset.unshift(item);
          triggerRowFlash(item.id);
          if (isGenerating) {
            currentGenCount++;
            const progressText = document.getElementById('gen-progress-text');
            if (progressText) {
              if (currentGenCount < targetGenCount) {
                progressText.innerText = 'Generating ticket ' + (currentGenCount + 1) + ' of ' + targetGenCount + '...';
              } else {
                progressText.innerText = 'Finishing generation (' + targetGenCount + ' of ' + targetGenCount + ')...';
              }
            }
          }
        }
        if (typeof renderDatasetTable === 'function') renderDatasetTable();
        if (typeof updateShowcaseView === 'function') updateShowcaseView();
      } catch (err) {
        console.error('Error handling SSE message:', err);
      }
    }

    function initSSEStream() {
      if (evtSource) {
        try {
          evtSource.close();
        } catch (_) {}
        evtSource = null;
      }

      try {
        evtSource = new EventSource('/api/stream');

        evtSource.onopen = () => {
          if (sseReconnectTimer) {
            clearTimeout(sseReconnectTimer);
            sseReconnectTimer = null;
          }
        };

        evtSource.onmessage = (e) => {
          handleSSEMessage(e.data);
        };

        evtSource.onerror = () => {
          if (evtSource && evtSource.readyState === EventSource.CLOSED) {
            if (!sseReconnectTimer) {
              sseReconnectTimer = setTimeout(() => {
                sseReconnectTimer = null;
                initSSEStream();
              }, 2000);
            }
          }
        };
      } catch (err) {
        if (!sseReconnectTimer) {
          sseReconnectTimer = setTimeout(() => {
            sseReconnectTimer = null;
            initSSEStream();
          }, 2000);
        }
      }
    }

    function ensureSSEConnected() {
      if (!evtSource || evtSource.readyState === EventSource.CLOSED) {
        initSSEStream();
      }
    }

    initSSEStream();

    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'visible') {
        ensureSSEConnected();
      }
    });


    function getCurrentTheme() {
      return document.documentElement.getAttribute('data-theme') || 'dark';
    }

    function applyTheme(theme, save = true) {
      document.documentElement.setAttribute('data-theme', theme);
      if (save) {
        localStorage.setItem('theme', theme);
      }
    }

    function toggleTheme() {
      const current = getCurrentTheme();
      const next = current === 'dark' ? 'light' : 'dark';
      applyTheme(next, true);
    }


if (window.matchMedia) {
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', e => {
    if (!localStorage.getItem('theme')) {
      applyTheme(e.matches ? 'dark' : 'light', false);
    }
  });
}

// Initial data load on DOM ready after all scripts have loaded
document.addEventListener('DOMContentLoaded', () => {
  if (typeof loadConfig === 'function') loadConfig();
  if (typeof loadDataset === 'function') loadDataset();
  if (typeof loadCriteria === 'function') loadCriteria();
});
