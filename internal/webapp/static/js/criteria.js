    let newQuestionChoices = ['Option 1', 'Option 2'];

    function renderNewQuestionChoices() {
      const container = document.getElementById('new-q-choices-container');
      if (!container) return;
      container.innerHTML = newQuestionChoices.map((choice, i) => `
        <span class="pill" style="display: inline-flex; align-items: center; gap: 6px; padding: 3px 8px; font-weight: 500;">
          <span>${escapeHtml(choice)}</span>
          <button type="button" style="background: none; border: none; color: var(--text-muted); cursor: pointer; padding: 0 2px; font-size: 0.85rem; line-height: 1;" title="Remove choice" onclick="removeChoiceFromNewQuestion(${i})">&times;</button>
        </span>
      `).join('');
    }

    function onNewQuestionTypeChange() {
      const typeEl = document.getElementById('new-q-type');
      if (!typeEl) return;
      if (typeEl.value === 'noul') {
        newQuestionChoices = ['Yes', 'No'];
      } else if (newQuestionChoices.length <= 2 && newQuestionChoices.includes('Yes') && newQuestionChoices.includes('No')) {
        newQuestionChoices = ['Option 1', 'Option 2'];
      }
      renderNewQuestionChoices();
    }

    function addChoiceToNewQuestion() {
      const input = document.getElementById('new-choice-text');
      if (!input) return;
      const val = (input.value || '').trim();
      if (!val) {
        showToast('Please type a choice name before adding.', 4000);
        return;
      }
      if (newQuestionChoices.some(c => c.toLowerCase() === val.toLowerCase())) {
        showToast('Choice "' + val + '" already exists.', 5000);
        return;
      }
      newQuestionChoices.push(val);
      renderNewQuestionChoices();
      input.value = '';
      input.focus();
    }

    function removeChoiceFromNewQuestion(idx) {
      if (newQuestionChoices.length <= 1) {
        showToast('At least one choice is required for a question.', 5000);
        return;
      }
      newQuestionChoices.splice(idx, 1);
      renderNewQuestionChoices();
    }


    async function loadCriteria() {
      try {
        const res = await fetch('/api/criteria');
        if (!res.ok) throw new Error(await res.text());
        triageCriteria = await res.json();
        renderCriteriaCards();
        if (typeof updateShowcaseView === 'function') updateShowcaseView();
      } catch (err) {
        console.error('Failed to load criteria:', err);
      }
    }

    function renderCriteriaCards() {
      const container = document.getElementById('criteria-list');
      if (!container) return;
      if (!triageCriteria || triageCriteria.length === 0) {
        container.innerHTML = '<div class="card" style="text-align: center; color: var(--text-muted); padding: 30px;">No operational criteria configured. Click "Reset Defaults" to restore standard questions.</div>';
        return;
      }

      container.innerHTML = triageCriteria.map((q, idx) => {
        const optList = q.options || [];
        const rubricRows = optList.map((opt, optIdx) => {
          const critVal = (q.criteria && q.criteria[opt]) ? q.criteria[opt] : '';
          return `
            <tr>
              <td style="width: 220px; font-weight: 500;">
                <span class="option-chip">${escapeHtml(opt)}</span>
              </td>
              <td>
                <input type="text" class="form-control" style="font-size: 0.82rem; padding: 5px 8px;"
                  data-q-idx="${idx}" data-opt-idx="${optIdx}"
                  value="${escapeHtml(critVal)}"
                  placeholder="Rubric criteria for ${escapeHtml(opt)}..."
                  onchange="updateOptionCriteria(${idx}, ${optIdx}, this.value)">
              </td>
              <td style="width: 40px; text-align: center;">
                ${optList.length > 1 ? `<button class="btn btn-danger" style="padding: 2px 6px; font-size: 0.75rem;" title="Remove this choice" onclick="deleteChoiceFromQuestion(${idx}, ${optIdx})">&times;</button>` : ''}
              </td>
            </tr>
          `;
        }).join('');

        const deleteBtn = triageCriteria.length > 1
          ? `<button class="btn btn-danger" style="padding: 3px 8px; font-size: 0.78rem;" onclick="deleteQuestion(${idx})">Remove Question</button>`
          : '';

        return `
          <div class="criteria-card">
            <div class="criteria-header">
              <div class="criteria-badges">
                <span class="criteria-num">Q${idx + 1}</span>
                <span class="criteria-id">${escapeHtml(q.id)}</span>
                <span class="criteria-type">${escapeHtml(q.type || 'choice')}</span>
                <span class="pill" style="font-size: 0.72rem;">${optList.length} choices</span>
              </div>
              <div>${deleteBtn}</div>
            </div>

            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 14px; margin-bottom: 12px;">
              <div class="form-group" style="margin-bottom: 0;">
                <label class="form-label">Question Text</label>
                <input type="text" class="form-control" value="${escapeHtml(q.text)}"
                  onchange="updateQuestionField(${idx}, 'text', this.value)">
              </div>
              <div class="form-group" style="margin-bottom: 0;">
                <label class="form-label">Instructions</label>
                <input type="text" class="form-control" value="${escapeHtml(q.instructions || '')}"
                  onchange="updateQuestionField(${idx}, 'instructions', this.value)">
              </div>
            </div>

            <div style="margin-top: 10px;">
              <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px;">
                <span style="font-size: 0.75rem; text-transform: uppercase; color: var(--text-muted); font-weight: 600;">
                  Decision Rubric Guidelines
                </span>
                <span style="font-size: 0.75rem; color: var(--text-muted);">
                  ${optList.length} choices configured
                </span>
              </div>
              <table class="rubric-table">
                <thead>
                  <tr>
                    <th style="width: 220px;">Choice / Option</th>
                    <th>Evaluation Guideline</th>
                    <th style="width: 40px; text-align: center;">Action</th>
                  </tr>
                </thead>
                <tbody>
                  ${rubricRows}
                </tbody>
              </table>

              <div style="display: flex; gap: 8px; align-items: center; margin-top: 10px; padding-top: 8px; border-top: 1px dashed var(--card-border);">
                <input type="text" class="form-control" id="add-choice-input-${idx}" placeholder="Add another choice..." style="max-width: 280px; padding: 5px 8px; font-size: 0.82rem;" onkeydown="if(event.key==='Enter'){addChoiceToQuestion(${idx});event.preventDefault();}">
                <button class="btn" type="button" style="padding: 5px 12px; font-size: 0.8rem;" onclick="addChoiceToQuestion(${idx})">+ Add Choice</button>
              </div>
            </div>
          </div>
        `;
      }).join('');
    }

    function addChoiceToQuestion(idx) {
      const input = document.getElementById('add-choice-input-' + idx);
      if (!input) return;
      const val = (input.value || '').trim();
      if (!val) {
        showToast('Please type a choice name before adding.', 4000);
        return;
      }
      const q = triageCriteria[idx];
      if (!q) return;
      if (!q.options) q.options = [];
      if (q.options.some(opt => opt.toLowerCase() === val.toLowerCase())) {
        showToast('Choice "' + val + '" already exists for this question.', 5000);
        return;
      }
      q.options.push(val);
      if (!q.criteria) q.criteria = {};
      q.criteria[val] = 'Evaluation guideline for ' + val;
      renderCriteriaCards();
      showToast('Choice "' + val + '" added to question. Click "Apply Criteria" or "Save JSON" to persist.', 5000);
    }

    function deleteChoiceFromQuestion(idx, optIdx) {
      const q = triageCriteria[idx];
      if (!q || !q.options) return;
      if (q.options.length <= 1) {
        showToast('A question must have at least one choice.', 5000);
        return;
      }
      const opt = q.options[optIdx];
      if (!opt) return;
      if (confirm('Remove choice "' + opt + '" from question ' + q.id + '?')) {
        q.options.splice(optIdx, 1);
        if (q.criteria) delete q.criteria[opt];
        renderCriteriaCards();
        showToast('Choice "' + opt + '" removed. Click "Apply Criteria" or "Save JSON" to persist.', 5000);
      }
    }

    function updateQuestionField(idx, field, val) {
      if (triageCriteria[idx]) {
        triageCriteria[idx][field] = val;
      }
    }

    function updateOptionCriteria(idx, optIdx, val) {
      if (triageCriteria[idx] && triageCriteria[idx].options) {
        const opt = triageCriteria[idx].options[optIdx];
        if (opt) {
          if (!triageCriteria[idx].criteria) {
            triageCriteria[idx].criteria = {};
          }
          triageCriteria[idx].criteria[opt] = val;
        }
      }
    }

    function toggleAddQuestionForm(force) {
      const box = document.getElementById('add-criteria-box');
      if (!box) return;
      if (typeof force === 'boolean') {
        box.style.display = force ? 'block' : 'none';
      } else {
        box.style.display = box.style.display === 'none' ? 'block' : 'none';
      }
      if (box.style.display === 'block') {
        renderNewQuestionChoices();
        document.getElementById('new-q-id').focus();
      }
    }

    function submitAddQuestion() {
      const idInput = document.getElementById('new-q-id');
      const typeInput = document.getElementById('new-q-type');
      const textInput = document.getElementById('new-q-text');
      const instInput = document.getElementById('new-q-instructions');

      const id = (idInput.value || '').trim().toLowerCase().replace(/[^a-z0-9_]/g, '_');
      const type = typeInput.value || 'choice';
      const text = (textInput.value || '').trim();
      const instructions = (instInput.value || '').trim();

      if (!id) {
        showToast('Please provide a unique Question ID in snake_case (e.g. root_cause).', 5000);
        return;
      }
      if (!text) {
        showToast('Please provide the Question Text.', 5000);
        return;
      }
      if (newQuestionChoices.length === 0) {
        showToast('Please add at least one choice for the question.', 5000);
        return;
      }
      if (triageCriteria.some(q => q.id === id)) {
        showToast('A question with ID "' + id + '" already exists.', 5000);
        return;
      }

      const newQ = {
        id: id,
        type: type,
        text: text,
        instructions: instructions,
        options: [...newQuestionChoices],
        criteria: {}
      };
      newQuestionChoices.forEach(o => {
        newQ.criteria[o] = 'Evaluation guideline for ' + o;
      });

      triageCriteria.push(newQ);
      renderCriteriaCards();
      if (typeof updateShowcaseView === 'function') updateShowcaseView();
      toggleAddQuestionForm(false);

      idInput.value = '';
      textInput.value = '';
      instInput.value = '';
      newQuestionChoices = ['Option 1', 'Option 2'];

      showToast('Question "' + id + '" added. Click "Apply Criteria" or "Save JSON" to persist.', 5000);
    }

    function deleteQuestion(idx) {
      if (triageCriteria.length <= 1) {
        showToast('At least one operational triage question is required.', 5000);
        return;
      }
      const q = triageCriteria[idx];
      if (confirm('Are you sure you want to remove question "' + q.id + '"?')) {
        triageCriteria.splice(idx, 1);
        renderCriteriaCards();
        if (typeof updateShowcaseView === 'function') updateShowcaseView();
        showToast('Question "' + q.id + '" removed. Click "Apply Criteria" or "Save JSON" to persist.', 5000);
      }
    }

    async function saveCriteria() {
      const btn = document.getElementById('btn-save-criteria');
      btn.disabled = true;
      btn.innerText = 'Applying...';
      try {
        const res = await fetch('/api/criteria', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(triageCriteria)
        });
        if (!res.ok) {
          throw new Error(await res.text());
        }
        const data = await res.json();
        triageCriteria = data.questions;
        renderCriteriaCards();
        if (typeof updateShowcaseView === 'function') updateShowcaseView();
        showToast('Triage criteria and rubric successfully applied in memory.');
      } catch (err) {
        showToast('Failed to apply criteria: ' + err.message, 6000);
      } finally {
        btn.disabled = false;
        btn.innerText = 'Apply Criteria';
      }
    }

    async function saveCriteriaFile() {
      const pathInput = document.getElementById('criteria-file-path');
      const targetPath = (pathInput && pathInput.value.trim()) ? pathInput.value.trim() : 'questions.json';
      try {
        const res = await fetch('/api/criteria/save', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ path: targetPath })
        });
        if (!res.ok) {
          throw new Error(await res.text());
        }
        const data = await res.json();
        showToast('Saved ' + data.count + ' triage questions to ' + data.path + '.');
      } catch (err) {
        showToast('Failed to save criteria JSON: ' + err.message, 6000);
      }
    }

    async function loadCriteriaFile() {
      const pathInput = document.getElementById('criteria-file-path');
      const targetPath = (pathInput && pathInput.value.trim()) ? pathInput.value.trim() : 'questions.json';
      try {
        const res = await fetch('/api/criteria/load', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ path: targetPath })
        });
        if (!res.ok) {
          throw new Error(await res.text());
        }
        const data = await res.json();
        triageCriteria = data.questions;
        renderCriteriaCards();
        if (typeof updateShowcaseView === 'function') updateShowcaseView();
        showToast('Loaded ' + data.count + ' triage questions from ' + data.path + '.');
      } catch (err) {
        showToast('Failed to load criteria JSON: ' + err.message, 6000);
      }
    }

    async function resetCriteria() {
      if (!confirm('Reset all triage questions and rubric to standard defaults?')) {
        return;
      }
      const btn = document.getElementById('btn-reset-criteria');
      btn.disabled = true;
      try {
        const res = await fetch('/api/criteria/reset', { method: 'POST' });
        if (!res.ok) {
          throw new Error(await res.text());
        }
        const data = await res.json();
        triageCriteria = data.questions;
        renderCriteriaCards();
        if (typeof updateShowcaseView === 'function') updateShowcaseView();
        showToast('Triage criteria reset to standard defaults.');
      } catch (err) {
        showToast('Failed to reset criteria: ' + err.message, 6000);
      } finally {
        btn.disabled = false;
      }
    }
