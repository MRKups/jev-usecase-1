    function updateHeaders() {
      // Reserved for optional header indicator synchronization
    }

    function selectGenerator(gen) {
      appConfig.active_generator = gen;
      ['openai', 'anthropic', 'gemini'].forEach(id => {
        const opt = document.getElementById('opt-' + id);
        const fld = document.getElementById('fields-' + id);
        if (id === gen) {
          if (opt) opt.classList.add('selected');
          if (fld) fld.style.display = 'block';
        } else {
          if (opt) opt.classList.remove('selected');
          if (fld) fld.style.display = 'none';
        }
      });
      const statusEl = document.getElementById('provider-test-status');
      if (statusEl) statusEl.innerHTML = '';
      updateHeaders();
    }

    async function testActiveProvider() {
      const btn = document.getElementById('btn-test-provider');
      const statusEl = document.getElementById('provider-test-status');
      btn.disabled = true;
      btn.innerText = 'Testing...';
      statusEl.innerHTML = '<span style="color: var(--text-muted); font-size: 0.82rem;">Verifying provider connection...</span>';

      const payload = collectConfigFromUI();
      try {
        const res = await fetch('/api/config/test-provider', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload)
        });
        const data = await res.json();
        if (data.ok) {
          const provName = escapeHtml((data.provider || '').toUpperCase());
          const modelName = escapeHtml(data.model || '');
          statusEl.innerHTML = `<span class="pill triaged" style="padding: 4px 10px; font-weight: 500;">Connected: ${provName} (${modelName}) in ${data.latency_ms}ms</span>`;
        } else {
          const errMsg = escapeHtml(data.error || 'Connection failed');
          statusEl.innerHTML = `<span class="pill security-alert" style="padding: 4px 10px; font-weight: 500;">Failed: ${errMsg}</span>`;
        }
      } catch (err) {
        statusEl.innerHTML = `<span class="pill security-alert" style="padding: 4px 10px; font-weight: 500;">Network error: ${escapeHtml(err.message)}</span>`;
      } finally {
        btn.disabled = false;
        btn.innerText = 'Test Provider';
      }
    }


    async function loadConfig() {
      try {
        const res = await fetch('/api/config');
        appConfig = await res.json();
        renderConfigUI();
        if (typeof updateShowcaseView === 'function') updateShowcaseView();
      } catch (err) {
        console.error('Failed to load config:', err);
      }
    }


    const engineAState = {
      jev: { endpoint: 'https://api.typesafe.ai', model: 'jev-latest', api_key: '' },
      openai: { endpoint: 'https://api.openai.com/v1', model: 'gpt-4o-mini', api_key: '' },
      anthropic: { endpoint: 'https://api.anthropic.com', model: 'claude-3-5-haiku-latest', api_key: '' },
      gemini: { endpoint: '', model: 'gemini-2.5-flash', api_key: '' },
      disabled: { endpoint: '', model: '', api_key: '' },
    };

    const engineBState = {
      jev: { endpoint: 'https://api.typesafe.ai', model: 'jev-latest', api_key: '' },
      openai: { endpoint: 'https://api.openai.com/v1', model: 'gpt-4o-mini', api_key: '' },
      anthropic: { endpoint: 'https://api.anthropic.com', model: 'claude-3-5-haiku-latest', api_key: '' },
      gemini: { endpoint: '', model: 'gemini-2.5-flash', api_key: '' },
      disabled: { endpoint: '', model: '', api_key: '' },
    };

    let currentEvaluationMode = 'parallel';

    function setEvaluationMode(mode) {
      currentEvaluationMode = mode === 'serial' ? 'serial' : 'parallel';
      const btnSerial = document.getElementById('mode-serial');
      const btnParallel = document.getElementById('mode-parallel');
      const desc = document.getElementById('mode-description');
      if (btnSerial && btnParallel) {
        if (currentEvaluationMode === 'serial') {
          btnSerial.classList.add('active');
          btnParallel.classList.remove('active');
          if (desc) desc.innerText = 'Serial: processes 1 ticket at a time in sequence.';
        } else {
          btnParallel.classList.add('active');
          btnSerial.classList.remove('active');
          if (desc) desc.innerText = 'Parallel: dispatches 1 ticket per second with up to 10 in flight.';
        }
      }
    }
    // Backward compatibility alias
    window.setExecutionMode = setEvaluationMode;

    function renderConfigUI() {
      setEvaluationMode(appConfig.evaluation_mode || appConfig.execution_mode || 'parallel');
      selectGenerator(appConfig.active_generator || 'openai');
      document.getElementById('cfg-openai-key').value = appConfig.openai?.api_key || '';
      document.getElementById('cfg-openai-model').value = appConfig.openai?.model || 'gpt-4o-mini';
      document.getElementById('cfg-openai-base').value = appConfig.openai?.base_url || 'https://api.openai.com/v1';

      document.getElementById('cfg-anthropic-key').value = appConfig.anthropic?.api_key || '';
      document.getElementById('cfg-anthropic-model').value = appConfig.anthropic?.model || 'claude-3-5-haiku-latest';
      document.getElementById('cfg-anthropic-base').value = appConfig.anthropic?.base_url || 'https://api.anthropic.com';

      document.getElementById('cfg-gemini-key').value = appConfig.gemini?.api_key || '';
      document.getElementById('cfg-gemini-model').value = appConfig.gemini?.model || 'gemini-2.5-flash';

      if (appConfig.dataset_path && document.getElementById('dataset-file-path')) {
        document.getElementById('dataset-file-path').value = appConfig.dataset_path;
      }

      if (appConfig.openai) {
        engineAState.openai.endpoint = appConfig.openai.base_url || 'https://api.openai.com/v1';
        engineAState.openai.model = appConfig.openai.model || 'gpt-4o-mini';
        engineAState.openai.api_key = appConfig.openai.api_key || '';
        engineBState.openai.endpoint = appConfig.openai.base_url || 'https://api.openai.com/v1';
        engineBState.openai.model = appConfig.openai.model || 'gpt-4o-mini';
        engineBState.openai.api_key = appConfig.openai.api_key || '';
      }
      if (appConfig.anthropic) {
        engineAState.anthropic.endpoint = appConfig.anthropic.base_url || 'https://api.anthropic.com';
        engineAState.anthropic.model = appConfig.anthropic.model || 'claude-3-5-haiku-latest';
        engineAState.anthropic.api_key = appConfig.anthropic.api_key || '';
        engineBState.anthropic.endpoint = appConfig.anthropic.base_url || 'https://api.anthropic.com';
        engineBState.anthropic.model = appConfig.anthropic.model || 'claude-3-5-haiku-latest';
        engineBState.anthropic.api_key = appConfig.anthropic.api_key || '';
      }
      if (appConfig.gemini) {
        engineAState.gemini.model = appConfig.gemini.model || 'gemini-2.5-flash';
        engineAState.gemini.api_key = appConfig.gemini.api_key || '';
        engineBState.gemini.model = appConfig.gemini.model || 'gemini-2.5-flash';
        engineBState.gemini.api_key = appConfig.gemini.api_key || '';
      }
      if (appConfig.jev) {
        engineAState.jev.endpoint = appConfig.jev.endpoint || 'https://api.typesafe.ai';
        engineAState.jev.model = appConfig.jev.model || 'jev-latest';
        engineAState.jev.api_key = appConfig.jev.api_key || '';
        engineBState.jev.endpoint = appConfig.jev.endpoint || 'https://api.typesafe.ai';
        engineBState.jev.model = appConfig.jev.model || 'jev-latest';
        engineBState.jev.api_key = appConfig.jev.api_key || '';
      }

      const cfgA = appConfig.engine_a || { provider: 'jev' };
      let provA = (cfgA.provider || 'jev').toLowerCase();
      if (provA === 'none') provA = 'disabled';
      if (engineAState[provA]) {
        if (cfgA.endpoint) engineAState[provA].endpoint = cfgA.endpoint;
        if (cfgA.model) engineAState[provA].model = cfgA.model;
        if (cfgA.api_key) engineAState[provA].api_key = cfgA.api_key;
      }
      selectEngineProvider('a', provA);

      const cfgB = appConfig.engine_b || { provider: 'openai' };
      let provB = (cfgB.provider || 'openai').toLowerCase();
      if (provB === 'none') provB = 'disabled';
      if (engineBState[provB]) {
        if (cfgB.endpoint) engineBState[provB].endpoint = cfgB.endpoint;
        if (cfgB.model) engineBState[provB].model = cfgB.model;
        if (cfgB.api_key) engineBState[provB].api_key = cfgB.api_key;
      }
      selectEngineProvider('b', provB);

      updateHeaders();
    }

    function selectEngineProvider(slot, provider) {
      const prev = slot === 'a' ? currentEngineAProvider : currentEngineBProvider;
      if (prev && prev !== 'disabled' && prev !== provider) {
        const prevSt = slot === 'a' ? engineAState[prev] : engineBState[prev];
        if (prevSt) {
          const epEl = document.getElementById(`cfg-engine-${slot}-endpoint`);
          const modEl = document.getElementById(`cfg-engine-${slot}-model`);
          const keyEl = document.getElementById(`cfg-engine-${slot}-key`);
          if (epEl) prevSt.endpoint = epEl.value;
          if (modEl) prevSt.model = modEl.value;
          if (keyEl) prevSt.api_key = keyEl.value;
        }
      }

      if (slot === 'a') {
        currentEngineAProvider = provider;
      } else {
        currentEngineBProvider = provider;
      }

      const allProviders = ['jev', 'openai', 'anthropic', 'gemini', 'disabled'];
      allProviders.forEach(p => {
        const opt = document.getElementById(`opt-engine-${slot}-${p}`);
        if (opt) {
          if (p === provider) {
            opt.classList.add('selected');
          } else {
            opt.classList.remove('selected');
          }
        }
      });

      const fieldsEl = document.getElementById(`fields-engine-${slot}`);
      const disabledMsgEl = document.getElementById(`disabled-msg-engine-${slot}`);
      const btnTest = document.getElementById(`btn-test-engine-${slot}`);
      const statusEl = document.getElementById(`engine-${slot}-test-status`);
      if (statusEl) statusEl.innerHTML = '';

      if (provider === 'disabled' || provider === 'none') {
        if (fieldsEl) fieldsEl.style.display = 'none';
        if (disabledMsgEl) disabledMsgEl.style.display = 'block';
        if (btnTest) btnTest.style.display = 'none';
        return;
      }

      if (fieldsEl) fieldsEl.style.display = 'block';
      if (disabledMsgEl) disabledMsgEl.style.display = 'none';
      if (btnTest) btnTest.style.display = '';

      const st = slot === 'a' ? engineAState[provider] : engineBState[provider];
      const epEl = document.getElementById(`cfg-engine-${slot}-endpoint`);
      const modEl = document.getElementById(`cfg-engine-${slot}-model`);
      const keyEl = document.getElementById(`cfg-engine-${slot}-key`);
      const epGroup = document.getElementById(`group-engine-${slot}-endpoint`);
      const epLabel = document.getElementById(`lbl-engine-${slot}-endpoint`);
      const epNotice = document.getElementById(`notice-engine-${slot}-endpoint`);
      const modLabel = document.getElementById(`lbl-engine-${slot}-model`);
      const keyLabel = document.getElementById(`lbl-engine-${slot}-key`);

      if (provider === 'jev') {
        if (epGroup) epGroup.style.display = 'block';
        if (epLabel) epLabel.innerText = 'Service Endpoint URL';
        if (epEl) {
          epEl.disabled = false;
          epEl.placeholder = 'https://api.typesafe.ai';
          epEl.value = st?.endpoint || 'https://api.typesafe.ai';
        }
        if (epNotice) epNotice.innerText = 'TypeSafe Jev API endpoint (defaults to https://api.typesafe.ai).';
        if (modLabel) modLabel.innerText = 'Model';
        if (modEl) {
          modEl.placeholder = 'jev-latest';
          modEl.value = st?.model || 'jev-latest';
        }
        if (keyLabel) keyLabel.innerText = 'API Key';
        if (keyEl) {
          keyEl.placeholder = 'API Key';
          keyEl.value = st?.api_key || '';
        }
      } else if (provider === 'openai') {
        if (epGroup) epGroup.style.display = 'block';
        if (epLabel) epLabel.innerText = 'Base URL (Required for LM Studio, Ollama, OpenRouter, or proxies)';
        if (epEl) {
          epEl.disabled = false;
          epEl.placeholder = 'https://api.openai.com/v1';
          epEl.value = st?.endpoint || 'https://api.openai.com/v1';
        }
        if (epNotice) epNotice.innerText = 'Leave default for official OpenAI API, or point to local endpoint.';
        if (modLabel) modLabel.innerText = 'Model';
        if (modEl) {
          modEl.placeholder = 'gpt-4o-mini';
          modEl.value = st?.model || 'gpt-4o-mini';
        }
        if (keyLabel) keyLabel.innerText = 'API Key (Optional for LM Studio or Ollama)';
        if (keyEl) {
          keyEl.placeholder = 'sk-...';
          keyEl.value = st?.api_key || '';
        }
      } else if (provider === 'anthropic') {
        if (epGroup) epGroup.style.display = 'block';
        if (epLabel) epLabel.innerText = 'Base URL (Optional)';
        if (epEl) {
          epEl.disabled = false;
          epEl.placeholder = 'https://api.anthropic.com';
          epEl.value = st?.endpoint || 'https://api.anthropic.com';
        }
        if (epNotice) epNotice.innerText = 'Leave default for official Anthropic API.';
        if (modLabel) modLabel.innerText = 'Model';
        if (modEl) {
          modEl.placeholder = 'claude-3-5-haiku-latest';
          modEl.value = st?.model || 'claude-3-5-haiku-latest';
        }
        if (keyLabel) keyLabel.innerText = 'Anthropic API Key';
        if (keyEl) {
          keyEl.placeholder = 'sk-ant-...';
          keyEl.value = st?.api_key || '';
        }
      } else if (provider === 'gemini') {
        if (epGroup) epGroup.style.display = 'block';
        if (epLabel) epLabel.innerText = 'Base URL (Official Endpoint)';
        if (epEl) {
          epEl.placeholder = 'https://generativelanguage.googleapis.com';
          epEl.value = 'https://generativelanguage.googleapis.com';
          epEl.disabled = true;
        }
        if (epNotice) epNotice.innerText = 'Google Gemini API uses official cloud endpoints.';
        if (modLabel) modLabel.innerText = 'Model';
        if (modEl) {
          modEl.placeholder = 'gemini-2.5-flash';
          modEl.value = st?.model || 'gemini-2.5-flash';
        }
        if (keyLabel) keyLabel.innerText = 'Google Gemini API Key';
        if (keyEl) {
          keyEl.placeholder = 'AIzaSy...';
          keyEl.value = st?.api_key || '';
        }
      }
    }

    async function testEngine(slot) {
      const btn = document.getElementById(`btn-test-engine-${slot}`);
      const statusEl = document.getElementById(`engine-${slot}-test-status`);
      btn.disabled = true;
      btn.innerText = 'Testing...';
      statusEl.innerHTML = '<span style="color: var(--text-muted); font-size: 0.82rem;">Verifying connection...</span>';

      const cfg = collectEngineConfig(slot);
      try {
        const res = await fetch('/api/config/test-engine', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            slot: slot,
            provider: cfg.provider,
            endpoint: cfg.endpoint,
            model: cfg.model,
            api_key: cfg.api_key
          })
        });
        const data = await res.json();
        if (data.ok) {
          if (data.provider === 'none') {
            statusEl.innerHTML = '<span class="pill" style="padding: 4px 10px; font-weight: 500;">Disabled</span>';
          } else {
            const pName = escapeHtml((data.provider || '').toUpperCase());
            const mName = escapeHtml(data.model || '');
            statusEl.innerHTML = `<span class="pill triaged" style="padding: 4px 10px; font-weight: 500;">Connected: ${pName} (${mName}) in ${data.latency_ms}ms</span>`;
          }
        } else {
          const errMsg = escapeHtml(data.error || 'Connection failed');
          statusEl.innerHTML = `<span class="pill security-alert" style="padding: 4px 10px; font-weight: 500;">Failed: ${errMsg}</span>`;
        }
      } catch (err) {
        statusEl.innerHTML = `<span class="pill security-alert" style="padding: 4px 10px; font-weight: 500;">Network error: ${escapeHtml(err.message)}</span>`;
      } finally {
        btn.disabled = false;
        btn.innerText = 'Test Connection';
      }
    }

    function collectEngineConfig(slot) {
      const provider = slot === 'a' ? currentEngineAProvider : currentEngineBProvider;
      if (provider === 'disabled' || provider === 'none') {
        return {
          provider: 'none',
          endpoint: '',
          model: '',
          api_key: ''
        };
      }
      const epEl = document.getElementById(`cfg-engine-${slot}-endpoint`);
      const modEl = document.getElementById(`cfg-engine-${slot}-model`);
      const keyEl = document.getElementById(`cfg-engine-${slot}-key`);
      return {
        provider: provider,
        endpoint: (provider === 'gemini') ? '' : (epEl ? epEl.value.trim() : ''),
        model: modEl ? modEl.value.trim() : '',
        api_key: keyEl ? keyEl.value.trim() : ''
      };
    }

    function collectConfigFromUI() {
      const engA = collectEngineConfig('a');
      const engB = collectEngineConfig('b');
      const dsInput = document.getElementById('dataset-file-path');
      return {
        addr: appConfig.addr || '127.0.0.1:8080',
        active_generator: appConfig.active_generator || 'openai',
        openai: {
          api_key: document.getElementById('cfg-openai-key').value,
          model: document.getElementById('cfg-openai-model').value,
          base_url: document.getElementById('cfg-openai-base').value,
        },
        anthropic: {
          api_key: document.getElementById('cfg-anthropic-key').value,
          model: document.getElementById('cfg-anthropic-model').value,
          base_url: document.getElementById('cfg-anthropic-base').value,
        },
        gemini: {
          api_key: document.getElementById('cfg-gemini-key').value,
          model: document.getElementById('cfg-gemini-model').value,
        },
        jev: {
          endpoint: engA.provider === 'jev' ? engA.endpoint : (engB.provider === 'jev' ? engB.endpoint : (engineAState.jev.endpoint || appConfig.jev?.endpoint || 'https://api.typesafe.ai')),
          model: engA.provider === 'jev' ? engA.model : (engB.provider === 'jev' ? engB.model : (engineAState.jev.model || appConfig.jev?.model || 'jev-latest')),
          api_key: engA.provider === 'jev' ? engA.api_key : (engB.provider === 'jev' ? engB.api_key : (engineAState.jev.api_key || appConfig.jev?.api_key || '')),
        },
        engine_a: engA,
        engine_b: engB,
        dataset_path: (dsInput && dsInput.value.trim()) ? dsInput.value.trim() : (appConfig.dataset_path || 'dataset.json'),
        evaluation_mode: currentEvaluationMode
      };
    }

    async function applyConfig() {
      const payload = collectConfigFromUI();
      try {
        const res = await fetch('/api/config', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload)
        });
        if (!res.ok) {
          const errText = await res.text();
          throw new Error(errText);
        }
        const data = await res.json();
        appConfig = data.config;
        renderConfigUI();
        if (typeof updateShowcaseView === 'function') updateShowcaseView();
        showToast('Configuration applied successfully.');
      } catch (err) {
        showToast('Failed to apply config: ' + err.message, 6000);
      }
    }

    async function saveConfigFile() {
      const payload = collectConfigFromUI();
      await fetch('/api/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      const path = document.getElementById('cfg-file-path').value || 'config.json';
      try {
        const res = await fetch('/api/config/save', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ path: path })
        });
        const data = await res.json();
        showToast('Configuration saved to ' + data.path);
      } catch (err) {
        showToast('Save failed: ' + err.message, 6000);
      }
    }

    async function loadConfigFile() {
      const path = document.getElementById('cfg-file-path').value || 'config.json';
      try {
        const res = await fetch('/api/config/load', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ path: path })
        });
        if (!res.ok) {
          const errText = await res.text();
          throw new Error(errText);
        }
        const data = await res.json();
        appConfig = data.config;
        renderConfigUI();
        if (typeof updateShowcaseView === 'function') updateShowcaseView();
        showToast('Loaded configuration from ' + path);
      } catch (err) {
        showToast('Load failed: ' + err.message, 6000);
      }
    }

    function toggleKeyVisibility(inputId, btn) {
      const input = document.getElementById(inputId);
      if (!input) return;
      input.classList.toggle('masked-key');
      if (btn) {
        const isMasked = input.classList.contains('masked-key');
        btn.innerHTML = isMasked
          ? '<svg class="key-eye-icon" viewBox="0 0 24 24" width="15" height="15" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg>'
          : '<svg class="key-eye-icon" viewBox="0 0 24 24" width="15" height="15" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"></path><line x1="1" y1="1" x2="23" y2="23"></line></svg>';
        btn.title = isMasked ? 'Show Key' : 'Hide Key';
        btn.setAttribute('aria-label', isMasked ? 'Show API key' : 'Hide API key');
      }
    }

