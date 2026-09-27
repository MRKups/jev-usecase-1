    async function loadDataset() {
      try {
        const res = await fetch('/api/dataset');
        const data = await res.json();
        dataset = Array.isArray(data) ? data : [];
        if (typeof renderDatasetTable === 'function') renderDatasetTable();
        if (typeof updateShowcaseView === 'function') updateShowcaseView();
      } catch (err) {
        console.error('Failed to load dataset:', err);
        dataset = [];
      }
    }
    async function generateDataset() {
      if (isGenerating) return;

      let count = parseInt(document.getElementById('gen-count').value, 10);
      if (isNaN(count) || count < 1) {
        count = 1;
      } else if (count > 99) {
        count = 99;
      }
      document.getElementById('gen-count').value = count;

      const btnGen = document.getElementById('btn-generate-dataset');
      const btnCancel = document.getElementById('btn-cancel-dataset');
      const progressContainer = document.getElementById('gen-progress-container');
      const progressText = document.getElementById('gen-progress-text');
      const countInput = document.getElementById('gen-count');

      btnGen.style.display = 'none';
      btnCancel.style.display = 'inline-flex';
      btnCancel.disabled = false;
      btnCancel.innerText = 'Cancel';
      progressContainer.style.display = 'inline-flex';
      countInput.disabled = true;

      targetGenCount = count;
      currentGenCount = 0;
      isGenerating = true;
      progressText.innerText = 'Generating ticket 1 of ' + targetGenCount + '...';

      if (typeof ensureSSEConnected === 'function') {
        ensureSSEConnected();
      }

      genAbortController = new AbortController();

      const pollTimer = setInterval(async () => {
        if (!isGenerating) {
          clearInterval(pollTimer);
          return;
        }
        try {
          const res = await fetch('/api/dataset');
          if (!res.ok) return;
          const fresh = await res.json();
          if (!Array.isArray(fresh)) return;

          const existingIds = new Set((dataset || []).map(t => t.id));
          const newlyAdded = [];
          for (const item of fresh) {
            if (!existingIds.has(item.id)) {
              newlyAdded.push(item);
            }
          }

          if (newlyAdded.length > 0) {
            dataset = fresh;
            triggerRowFlash(newlyAdded[0].id);
            currentGenCount = Math.min(targetGenCount, currentGenCount + newlyAdded.length);
            if (progressText) {
              if (currentGenCount < targetGenCount) {
                progressText.innerText = 'Generating ticket ' + (currentGenCount + 1) + ' of ' + targetGenCount + '...';
              } else {
                progressText.innerText = 'Finishing generation (' + targetGenCount + ' of ' + targetGenCount + ')...';
              }
            }
            if (typeof renderDatasetTable === 'function') renderDatasetTable();
            if (typeof updateShowcaseView === 'function') updateShowcaseView();
          }
        } catch (_) {}
      }, 1000);

      try {
        const res = await fetch('/api/dataset/generate', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ count: count }),
          signal: genAbortController.signal
        });
        if (!res.ok) {
          const errText = await res.text();
          throw new Error(errText);
        }
        const data = await res.json();
        await loadDataset();
        if (data.cancelled) {
          const suffix = data.generated === 1 ? 'ticket' : 'tickets';
          showToast('Generation cancelled. ' + data.generated + ' ' + suffix + ' generated.', 10000);
        } else if (data.error || (data.requested && data.generated < data.requested)) {
          const errMsg = data.error ? ': ' + data.error : '';
          showToast('Generated ' + data.generated + ' of ' + targetGenCount + ' tickets. Generator stopped' + errMsg, 10000);
        } else {
          const suffix = data.generated === 1 ? 'ticket' : 'tickets';
          showToast('Generated ' + data.generated + ' ' + suffix + ' successfully.', 10000);
        }
      } catch (err) {
        if (err.name === 'AbortError') {
          showToast('Generation cancelled.', 10000);
        } else {
          const msg = err.message || String(err);
          showToast(msg.toLowerCase().includes('failed') ? msg : 'Generation failed: ' + msg, 8000);
        }
        await loadDataset();
      } finally {
        clearInterval(pollTimer);
        isGenerating = false;
        genAbortController = null;
        btnGen.style.display = '';
        btnCancel.style.display = 'none';
        progressContainer.style.display = 'none';
        countInput.disabled = false;
      }
    }

    async function cancelDatasetGeneration() {
      const btnCancel = document.getElementById('btn-cancel-dataset');
      if (btnCancel) {
        btnCancel.disabled = true;
        btnCancel.innerText = 'Cancelling...';
      }
      try {
        await fetch('/api/dataset/cancel', { method: 'POST' });
      } catch (err) {
        console.error('Cancel request failed:', err);
        if (genAbortController) {
          genAbortController.abort();
        }
      }
    }

    async function saveDatasetFile() {
      if (isGenerating) return;
      const path = document.getElementById('dataset-file-path').value;
      try {
        const res = await fetch('/api/dataset/save', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ path: path })
        });
        if (!res.ok) {
          const errText = await res.text();
          throw new Error(errText);
        }
        const data = await res.json();
        showToast('Dataset (' + (data.count || 0) + ' tickets) saved to ' + data.path);
      } catch (err) {
        showToast('Save dataset failed: ' + err.message, 6000);
      }
    }

    async function loadDatasetFile() {
      if (isGenerating) return;
      const path = document.getElementById('dataset-file-path').value;
      try {
        const res = await fetch('/api/dataset/load', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ path: path })
        });
        if (!res.ok) {
          const errText = await res.text();
          throw new Error(errText);
        }
        const data = await res.json();
        await loadDataset();
        showToast('Loaded ' + (data.count || 0) + ' tickets from ' + data.path);
      } catch (err) {
        showToast('Load dataset failed: ' + err.message, 6000);
      }
    }

    async function clearDataset() {
      if (isGenerating) return;
      if (!confirm('Clear all tickets from memory?')) return;
      try {
        await fetch('/api/dataset/clear', { method: 'POST' });
      } catch (err) {
        console.error('Failed to clear dataset on server:', err);
      }
      dataset = [];
      selectedTicketId = null;
      latestFlashTicketId = null;
      if (flashTimer) clearTimeout(flashTimer);
      Object.keys(evaluatingTickets).forEach(k => delete evaluatingTickets[k]);
      if (typeof renderDatasetTable === 'function') renderDatasetTable();
      if (typeof updateShowcaseView === 'function') updateShowcaseView();
    }

    function renderDatasetTable() {
      const tbody = document.getElementById('dataset-table-body');
      if (!Array.isArray(dataset) || dataset.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" style="text-align: center; color: var(--text-muted); padding: 30px;">No tickets generated yet. Configure a generator on Page 1 and click \'Generate Tickets\'.</td></tr>';
        return;
      }

      tbody.innerHTML = dataset.map(t => {
        const urgency = t.reported_urgency || t.urgency || 'Medium';
        const isNew = t.id === latestFlashTicketId ? ' row-flash' : '';
        return `
        <tr id="row-dataset-${cleanId(t.id)}" class="ticket-row${isNew}" onclick="selectedTicketId='${t.id}'" ondblclick="if(typeof handleTicketRowDblClick==='function'){handleTicketRowDblClick('${t.id}')}">
          <td style="font-family: ui-monospace, monospace; font-size: 0.78rem;">${t.id}</td>
          <td><span class="pill urgency-${urgency}">${urgency}</span></td>
          <td>${t.department}</td>
          <td>${t.category}</td>
          <td>${t.affected_system}</td>
          <td style="font-weight: 500; color: var(--text-bright);">${t.summary}</td>
          <td style="color: var(--text-muted);">${t.reporter_name}</td>
        </tr>
      `;
      }).join('');
    }
