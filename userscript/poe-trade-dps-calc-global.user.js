// ==UserScript==
// @name         POE Trade DPS Calculator (Global)
// @namespace    https://github.com/pob-recalc-api
// @version      1.0.0
// @description  Inject DPS calculation buttons on pathofexile.com trade pages. Uses POB Recalc API to compare gear swaps. No translation needed — items are already in English, only JSON→POB format conversion is performed server-side.
// @author       POB Recalc API
// @match        https://www.pathofexile.com/trade/*
// @match        https://pathofexile.com/trade/*
// @grant        GM_xmlhttpRequest
// @grant        GM_setValue
// @grant        GM_getValue
// @grant        GM_registerMenuCommand
// @grant        GM_addStyle
// @connect      localhost
// @connect      *
// @run-at       document-idle
// ==/UserScript==

(function () {
  'use strict';

  // =========================================================================
  // Config
  // =========================================================================

  const DEFAULT_CONFIG = {
    apiUrl: 'http://localhost:8080',
    pobCode: '',
    defaultSlot: 'auto', // auto = server detects slot from item type
    concurrency: 4, // batch calculation concurrency
  };

  function getConfig() {
    return {
      apiUrl: GM_getValue('apiUrl', DEFAULT_CONFIG.apiUrl),
      pobCode: GM_getValue('pobCode', DEFAULT_CONFIG.pobCode),
      defaultSlot: GM_getValue('defaultSlot', DEFAULT_CONFIG.defaultSlot),
      concurrency: parseInt(GM_getValue('concurrency', DEFAULT_CONFIG.concurrency), 10) || 4,
    };
  }

  // =========================================================================
  // Style injection
  // =========================================================================

  GM_addStyle(`
    /* Calc button */
    .pob-calc-btn {
      display: inline-flex;
      align-items: center;
      gap: 4px;
      padding: 3px 8px;
      margin: 4px 2px;
      border: 1px solid #af6025;
      border-radius: 3px;
      background: linear-gradient(180deg, #1a1a1a 0%, #0d0d0d 100%);
      color: #af6025;
      font-size: 11px;
      font-family: "FontinSmallCaps", "Fontin-SmallCaps", Verdana, Arial, sans-serif;
      cursor: pointer;
      transition: all 0.2s;
      white-space: nowrap;
    }
    .pob-calc-btn:hover {
      background: linear-gradient(180deg, #2a2a2a 0%, #1a1a1a 100%);
      border-color: #d4a76a;
      color: #d4a76a;
    }
    .pob-calc-btn.loading {
      opacity: 0.6;
      cursor: wait;
    }
    .pob-calc-btn.loading::after {
      content: '';
      width: 10px;
      height: 10px;
      border: 2px solid #af6025;
      border-top-color: transparent;
      border-radius: 50%;
      animation: pob-spin 0.8s linear infinite;
    }
    @keyframes pob-spin {
      to { transform: rotate(360deg); }
    }

    /* DPS result panel */
    .pob-result-panel {
      margin: 6px 0;
      padding: 8px 10px;
      background: #111;
      border: 1px solid #333;
      border-radius: 4px;
      font-size: 12px;
      line-height: 1.6;
      color: #ccc;
      font-family: "FontinSmallCaps", Verdana, sans-serif;
      max-width: 400px;
    }
    .pob-result-panel .pob-stat-row {
      display: flex;
      justify-content: space-between;
      padding: 1px 0;
    }
    .pob-result-panel .pob-stat-name {
      color: #999;
    }
    .pob-result-panel .pob-stat-val {
      font-weight: bold;
    }
    .pob-result-panel .pob-stat-val.positive {
      color: #20c820;
    }
    .pob-result-panel .pob-stat-val.negative {
      color: #e03030;
    }
    .pob-result-panel .pob-stat-val.neutral {
      color: #888;
    }
    .pob-result-panel .pob-section-title {
      color: #af6025;
      font-weight: bold;
      margin-top: 6px;
      margin-bottom: 2px;
      border-bottom: 1px solid #333;
      padding-bottom: 2px;
    }
    .pob-result-panel .pob-section-title:first-child {
      margin-top: 0;
    }
    .pob-result-panel .pob-error {
      color: #e03030;
      font-style: italic;
    }
    .pob-result-panel .pob-close-btn {
      float: right;
      cursor: pointer;
      color: #666;
      font-size: 14px;
      line-height: 1;
      padding: 0 4px;
    }
    .pob-result-panel .pob-close-btn:hover {
      color: #e03030;
    }

    /* Settings panel */
    .pob-settings-overlay {
      position: fixed;
      top: 0; left: 0; right: 0; bottom: 0;
      background: rgba(0,0,0,0.7);
      z-index: 99999;
      display: flex;
      align-items: center;
      justify-content: center;
    }
    .pob-settings-panel {
      background: #1a1a1a;
      border: 2px solid #af6025;
      border-radius: 8px;
      padding: 20px 24px;
      min-width: 450px;
      max-width: 600px;
      color: #ccc;
      font-family: "FontinSmallCaps", Verdana, sans-serif;
    }
    .pob-settings-panel h2 {
      margin: 0 0 16px 0;
      color: #af6025;
      font-size: 18px;
      border-bottom: 1px solid #333;
      padding-bottom: 8px;
    }
    .pob-settings-panel label {
      display: block;
      margin-bottom: 4px;
      color: #af6025;
      font-size: 13px;
    }
    .pob-settings-panel input,
    .pob-settings-panel textarea,
    .pob-settings-panel select {
      width: 100%;
      padding: 6px 8px;
      margin-bottom: 12px;
      background: #0d0d0d;
      border: 1px solid #444;
      border-radius: 4px;
      color: #ccc;
      font-size: 13px;
      font-family: monospace;
      box-sizing: border-box;
    }
    .pob-settings-panel textarea {
      height: 80px;
      resize: vertical;
    }
    .pob-settings-panel input:focus,
    .pob-settings-panel textarea:focus {
      border-color: #af6025;
      outline: none;
    }
    .pob-settings-panel .pob-btn-row {
      display: flex;
      gap: 8px;
      justify-content: flex-end;
      margin-top: 8px;
    }
    .pob-settings-panel button {
      padding: 6px 16px;
      border: 1px solid #af6025;
      border-radius: 4px;
      background: #1a1a1a;
      color: #af6025;
      cursor: pointer;
      font-size: 13px;
    }
    .pob-settings-panel button:hover {
      background: #2a2a2a;
    }
    .pob-settings-panel button.pob-primary {
      background: #af6025;
      color: #fff;
    }
    .pob-settings-panel button.pob-primary:hover {
      background: #c87030;
    }
    .pob-settings-panel .pob-hint {
      font-size: 11px;
      color: #666;
      margin-top: -8px;
      margin-bottom: 12px;
    }

    /* Float entry button */
    .pob-float-entry {
      position: fixed;
      bottom: 20px;
      right: 20px;
      z-index: 99998;
      padding: 8px 14px;
      background: linear-gradient(180deg, #2a2a2a 0%, #111 100%);
      border: 2px solid #af6025;
      border-radius: 6px;
      color: #af6025;
      font-size: 13px;
      font-family: "FontinSmallCaps", Verdana, sans-serif;
      cursor: pointer;
      box-shadow: 0 2px 8px rgba(0,0,0,0.5);
      transition: all 0.2s;
    }
    .pob-float-entry:hover {
      border-color: #d4a76a;
      color: #d4a76a;
      transform: translateY(-1px);
    }
    .pob-float-entry .pob-status-dot {
      display: inline-block;
      width: 8px;
      height: 8px;
      border-radius: 50%;
      margin-right: 6px;
    }
    .pob-float-entry .pob-status-dot.connected {
      background: #20c820;
    }
    .pob-float-entry .pob-status-dot.disconnected {
      background: #e03030;
    }

    /* Quick summary */
    .pob-quick-summary {
      display: inline-block;
      margin-left: 6px;
      font-size: 11px;
      font-weight: bold;
      font-family: "FontinSmallCaps", Verdana, sans-serif;
      vertical-align: middle;
    }

    /* Calc-all button */
    .pob-calc-all-btn {
      position: fixed;
      bottom: 60px;
      right: 20px;
      z-index: 99998;
      padding: 8px 14px;
      background: linear-gradient(180deg, #2a2a2a 0%, #111 100%);
      border: 2px solid #af6025;
      border-radius: 6px;
      color: #af6025;
      font-size: 13px;
      font-family: "FontinSmallCaps", Verdana, sans-serif;
      cursor: pointer;
      box-shadow: 0 2px 8px rgba(0,0,0,0.5);
      transition: all 0.2s;
      white-space: nowrap;
    }
    .pob-calc-all-btn:hover {
      border-color: #d4a76a;
      color: #d4a76a;
      transform: translateY(-1px);
    }
    .pob-calc-all-btn.loading {
      opacity: 0.7;
      cursor: wait;
    }
    .pob-calc-all-btn .pob-progress {
      font-size: 11px;
      margin-left: 4px;
      color: #888;
    }

    /* Best item highlight */
    .pob-best-item {
      outline: 2px solid #20c820 !important;
      outline-offset: -2px;
    }
  `);

  // =========================================================================
  // API calls — No translation needed for global server
  // =========================================================================

  /**
   * Call /convert-item: Convert English item JSON to POB item text (no translation)
   * @param {Object} itemData - Item JSON from trade API
   * @returns {Promise<{item_text: string, slot: string}>}
   */
  function callConvertItemAPI(itemData) {
    return new Promise((resolve, reject) => {
      const config = getConfig();
      const requestBody = JSON.stringify(itemData);
      GM_xmlhttpRequest({
        method: 'POST',
        url: `${config.apiUrl}/convert-item`,
        headers: { 'Content-Type': 'application/json' },
        data: requestBody,
        timeout: 15000,
        onload: function (response) {
          if (response.status === 200) {
            try {
              resolve(JSON.parse(response.responseText));
            } catch (e) {
              reject(new Error('JSON parse error: ' + e.message + ' | response: ' + response.responseText.substring(0, 500)));
            }
          } else {
            const respBody = response.responseText || '(empty)';
            console.error(`[POB DPS Calc] Convert API ${response.status} response:`, respBody);
            reject(new Error(`Convert API error (${response.status}): ${respBody.substring(0, 500)}`));
          }
        },
        onerror: function (err) {
          console.error('[POB DPS Calc] Convert API network error:', err);
          reject(new Error('Network error: cannot connect to API'));
        },
        ontimeout: function () {
          reject(new Error('Convert request timeout (15s)'));
        },
      });
    });
  }

  /**
   * Call /replace-item: Replace item in POB build and compare stats
   * @param {string} pobCode
   * @param {string} slot
   * @param {string} itemText - English POB item text
   * @returns {Promise<Object>} - DPS comparison result
   */
  function callReplaceItemAPI(pobCode, slot, itemText) {
    return new Promise((resolve, reject) => {
      const config = getConfig();
      GM_xmlhttpRequest({
        method: 'POST',
        url: `${config.apiUrl}/replace-item`,
        headers: { 'Content-Type': 'application/json' },
        data: JSON.stringify({
          pob_code: pobCode,
          slot: slot,
          item_text: itemText,
        }),
        timeout: 60000,
        onload: function (response) {
          if (response.status === 200) {
            try {
              resolve(JSON.parse(response.responseText));
            } catch (e) {
              reject(new Error('JSON parse error: ' + e.message + ' | response: ' + response.responseText.substring(0, 500)));
            }
          } else {
            const respBody = response.responseText || '(empty)';
            console.error(`[POB DPS Calc] Replace API ${response.status} response:`, respBody);
            console.error(`[POB DPS Calc] Replace API request: slot=${slot}, itemText length=${itemText.length}`);
            reject(new Error(`API error (${response.status}): ${respBody.substring(0, 500)}`));
          }
        },
        onerror: function (err) {
          console.error('[POB DPS Calc] Replace API network error:', err);
          reject(new Error('Network error: cannot connect to POB API at ' + config.apiUrl));
        },
        ontimeout: function () {
          reject(new Error(`Request timeout (60s) | slot=${slot}`));
        },
      });
    });
  }

  function checkApiHealth() {
    return new Promise((resolve) => {
      const config = getConfig();
      GM_xmlhttpRequest({
        method: 'GET',
        url: `${config.apiUrl}/health`,
        timeout: 5000,
        onload: function (response) {
          resolve(response.status === 200);
        },
        onerror: function () {
          resolve(false);
        },
        ontimeout: function () {
          resolve(false);
        },
      });
    });
  }

  // =========================================================================
  // Result panel rendering
  // =========================================================================

  function renderResultPanel(result, container) {
    const old = container.querySelector('.pob-result-panel');
    if (old) old.remove();

    const panel = document.createElement('div');
    panel.className = 'pob-result-panel';

    const closeBtn = document.createElement('span');
    closeBtn.className = 'pob-close-btn';
    closeBtn.textContent = '✕';
    closeBtn.onclick = () => panel.remove();
    panel.appendChild(closeBtn);

    const dpsStats = [
      { key: 'CombinedDPS', label: 'Combined DPS' },
      { key: 'TotalDPS', label: 'Total DPS' },
      { key: 'TotalDot', label: 'DoT DPS' },
      { key: 'AverageDamage', label: 'Avg Damage' },
      { key: 'Speed', label: 'Attack/Cast Speed' },
      { key: 'CritChance', label: 'Crit Chance' },
      { key: 'CritMultiplier', label: 'Crit Multi' },
    ];

    const defStats = [
      { key: 'Life', label: 'Life' },
      { key: 'EnergyShield', label: 'Energy Shield' },
      { key: 'Mana', label: 'Mana' },
      { key: 'ManaUnreserved', label: 'Unreserved Mana' },
      { key: 'Armour', label: 'Armour' },
      { key: 'Evasion', label: 'Evasion' },
      { key: 'BlockChance', label: 'Block Chance' },
      { key: 'SpellBlockChance', label: 'Spell Block' },
    ];

    const resStats = [
      { key: 'FireResist', label: 'Fire Res' },
      { key: 'ColdResist', label: 'Cold Res' },
      { key: 'LightningResist', label: 'Lightning Res' },
      { key: 'ChaosResist', label: 'Chaos Res' },
    ];

    function addSection(title, stats) {
      const titleEl = document.createElement('div');
      titleEl.className = 'pob-section-title';
      titleEl.textContent = title;
      panel.appendChild(titleEl);

      for (const stat of stats) {
        const diff = result.diff[stat.key];
        if (diff === undefined || diff === null) continue;
        const isKey = ['CombinedDPS', 'TotalDPS', 'Life', 'EnergyShield'].includes(stat.key);
        if (!isKey && Math.abs(diff) < 0.01) continue;

        const row = document.createElement('div');
        row.className = 'pob-stat-row';

        const nameSpan = document.createElement('span');
        nameSpan.className = 'pob-stat-name';
        nameSpan.textContent = stat.label;

        const valSpan = document.createElement('span');
        valSpan.className = 'pob-stat-val';

        let formatted;
        if (Math.abs(diff) >= 1000) {
          formatted = diff >= 0 ? `+${numberFormat(diff)}` : numberFormat(diff);
        } else if (stat.key === 'Speed') {
          formatted = diff >= 0 ? `+${diff.toFixed(2)}` : diff.toFixed(2);
        } else if (stat.key.includes('Chance') || stat.key.includes('Multiplier') || stat.key.includes('Resist')) {
          formatted = diff >= 0 ? `+${diff.toFixed(1)}%` : `${diff.toFixed(1)}%`;
        } else {
          formatted = diff >= 0 ? `+${Math.round(diff)}` : `${Math.round(diff)}`;
        }

        if (Math.abs(diff) < 0.01) {
          valSpan.classList.add('neutral');
          formatted = '—';
        } else if (diff > 0) {
          valSpan.classList.add('positive');
        } else {
          valSpan.classList.add('negative');
        }

        const beforeVal = result.before[stat.key] || 0;
        const afterVal = result.after[stat.key] || 0;
        valSpan.title = `${numberFormat(beforeVal)} → ${numberFormat(afterVal)}`;
        valSpan.textContent = formatted;

        row.appendChild(nameSpan);
        row.appendChild(valSpan);
        panel.appendChild(row);
      }
    }

    addSection('⚔️ Offence', dpsStats);
    addSection('🛡️ Defence', defStats);
    addSection('🔥 Resistances', resStats);

    container.appendChild(panel);
  }

  function renderErrorPanel(errorMsg, container) {
    const old = container.querySelector('.pob-result-panel');
    if (old) old.remove();

    const panel = document.createElement('div');
    panel.className = 'pob-result-panel';

    const closeBtn = document.createElement('span');
    closeBtn.className = 'pob-close-btn';
    closeBtn.textContent = '✕';
    closeBtn.onclick = () => panel.remove();
    panel.appendChild(closeBtn);

    const errorDiv = document.createElement('div');
    errorDiv.className = 'pob-error';
    errorDiv.textContent = '❌ ' + errorMsg;
    panel.appendChild(errorDiv);

    container.appendChild(panel);
  }

  function numberFormat(n) {
    if (typeof n !== 'number') return String(n);
    if (Math.abs(n) >= 1e6) return (n / 1e6).toFixed(2) + 'M';
    if (Math.abs(n) >= 1e3) return (n / 1e3).toFixed(1) + 'K';
    return Math.round(n).toString();
  }

  // =========================================================================
  // Intercept fetch to capture trade API item data
  // =========================================================================

  const itemDataCache = {};
  const calcResultCache = {};

  const origFetch = window.fetch;
  window.fetch = async function (...args) {
    const response = await origFetch.apply(this, args);
    const url = typeof args[0] === 'string' ? args[0] : (args[0] && args[0].url) || '';

    // Capture /api/trade/fetch or /api/trade2/fetch responses (item details)
    if (url.includes('/api/trade/fetch') || url.includes('/api/trade2/fetch')) {
      try {
        const cloned = response.clone();
        const data = await cloned.json();
        if (data && data.result) {
          for (const entry of data.result) {
            if (entry && entry.id && entry.item) {
              itemDataCache[entry.id] = entry.item;
            }
          }
          console.log(`[POB DPS Calc] Cached ${data.result.length} items`);
          setTimeout(() => injectCalcButtons(), 500);
        }
      } catch (e) {
        // Ignore parse errors
      }
    }

    return response;
  };

  // Also intercept XMLHttpRequest (fallback)
  const origXHROpen = XMLHttpRequest.prototype.open;
  const origXHRSend = XMLHttpRequest.prototype.send;
  XMLHttpRequest.prototype.open = function (method, url, ...rest) {
    this._pobUrl = url;
    return origXHROpen.apply(this, [method, url, ...rest]);
  };
  XMLHttpRequest.prototype.send = function (...args) {
    if (this._pobUrl && (this._pobUrl.includes('/api/trade/fetch') || this._pobUrl.includes('/api/trade2/fetch'))) {
      this.addEventListener('load', function () {
        try {
          const data = JSON.parse(this.responseText);
          if (data && data.result) {
            for (const entry of data.result) {
              if (entry && entry.id && entry.item) {
                itemDataCache[entry.id] = entry.item;
              }
            }
            setTimeout(() => injectCalcButtons(), 500);
          }
        } catch (e) {
          // Ignore
        }
      });
    }
    return origXHRSend.apply(this, args);
  };

  // =========================================================================
  // Button injection logic
  // =========================================================================

  /**
   * Calculate DPS for a single item (convert + compare)
   */
  async function calcSingleItem(row, btn) {
    const config = getConfig();
    const itemId = row.getAttribute('data-id') || row.id || '';

    if (!config.pobCode) {
      return { row, itemId, result: null, error: 'POB Code not configured' };
    }

    const rawItemData = itemId ? itemDataCache[itemId] : null;
    if (!rawItemData) {
      return { row, itemId, result: null, error: 'Item data not found. Please refresh the page.' };
    }

    // Unidentified items have no mods — skip
    if (rawItemData.identified === false) {
      const msg = 'Unidentified item, cannot calculate';
      renderErrorPanel(msg, row);
      return { row, itemId, result: null, error: msg };
    }

    // Loading state
    btn.classList.add('loading');
    btn.innerHTML = 'Converting';
    btn.disabled = true;

    const itemName = rawItemData.name
      ? `${rawItemData.name}${rawItemData.typeLine ? ' ' + rawItemData.typeLine : ''}`
      : (rawItemData.typeLine || 'Unknown item');

    let convertResult = null;
    let step = 'Convert';

    try {
      // Step 1: Convert (no translation, just format conversion)
      console.log(`[POB DPS Calc] [${itemId}] Converting: ${itemName}`);
      convertResult = await callConvertItemAPI(rawItemData);
      const itemText = convertResult.item_text;
      const slot = config.defaultSlot === 'auto' ? convertResult.slot : config.defaultSlot;
      console.log(`[POB DPS Calc] [${itemId}] Converted: slot=${slot}, itemText length=${itemText.length}`);

      // Step 2: Replace & compare
      step = 'Calculate';
      btn.innerHTML = 'Calculating';
      console.log(`[POB DPS Calc] [${itemId}] Comparing: slot=${slot}`);
      const result = await callReplaceItemAPI(config.pobCode, slot, itemText);

      const diffSummary = result.diff ? Object.entries(result.diff)
        .filter(([, v]) => v !== 0 && v !== undefined)
        .map(([k, v]) => `${k}: ${v >= 0 ? '+' : ''}${typeof v === 'number' ? v.toFixed(1) : v}`)
        .join(', ') : 'No change';
      console.log(`[POB DPS Calc] [${itemId}] ✅ ${itemName} | ${diffSummary}`);

      showQuickSummary(btn, result);
      renderResultPanel(result, row);

      const entry = { row, itemId, result, error: null };
      calcResultCache[itemId] = entry;
      return entry;
    } catch (err) {
      console.error(`[POB DPS Calc] [${itemId}] ❌ ${step} failed: ${itemName}`);
      console.error(`[POB DPS Calc] [${itemId}] Error:`, err.message);
      console.error(`[POB DPS Calc] [${itemId}] Raw item data:`, JSON.stringify(rawItemData, null, 2));
      if (convertResult) {
        console.error(`[POB DPS Calc] [${itemId}] Convert result:`, JSON.stringify(convertResult, null, 2));
      }
      console.error(`[POB DPS Calc] [${itemId}] Error object:`, err);

      renderErrorPanel(`[${step}] ${err.message}`, row);
      const entry = { row, itemId, result: null, error: `[${step}] ${err.message}` };
      calcResultCache[itemId] = entry;
      return entry;
    } finally {
      btn.classList.remove('loading');
      btn.innerHTML = '⚡ Calc DPS';
      btn.disabled = false;
    }
  }

  /**
   * Show quick DPS/Life change summary next to the button
   */
  function showQuickSummary(btn, result) {
    const dpsChange = result.diff && result.diff.TotalDPS !== undefined ? result.diff.TotalDPS : null;
    const lifeChange = result.diff && result.diff.Life !== undefined ? result.diff.Life : null;
    let summary = '';
    if (dpsChange !== null) {
      summary += `DPS: ${dpsChange >= 0 ? '+' : ''}${numberFormat(dpsChange)}`;
    }
    if (lifeChange !== null && Math.abs(lifeChange) >= 1) {
      summary += ` | Life: ${lifeChange >= 0 ? '+' : ''}${Math.round(lifeChange)}`;
    }
    if (summary) {
      let tag = btn.parentElement.querySelector('.pob-quick-summary');
      if (!tag) {
        tag = document.createElement('span');
        tag.className = 'pob-quick-summary';
        tag.style.cssText = 'margin-left:6px;font-size:11px;font-weight:bold;';
        btn.parentElement.insertBefore(tag, btn.nextSibling);
      }
      const color = (dpsChange || 0) >= 0 ? '#20c820' : '#e03030';
      tag.style.color = color;
      tag.textContent = summary;
    }
  }

  function injectCalcButtons() {
    const itemRows = document.querySelectorAll(
      '.resultset .row[data-id], [class*="search-result"][data-id], [class*="result-row"][data-id]'
    );

    const rows = itemRows.length > 0
      ? itemRows
      : document.querySelectorAll('.resultset .row, [class*="search-result"], [class*="result-row"]');

    if (rows.length === 0) {
      return;
    }

    for (const row of rows) {
      if (row.querySelector('.pob-calc-btn')) continue;

      const itemId = row.getAttribute('data-id') || '';
      if (!itemId) continue;

      const btn = document.createElement('button');
      btn.className = 'pob-calc-btn';
      btn.setAttribute('data-pob-item-id', itemId);
      btn.innerHTML = '⚡ Calc DPS';
      btn.title = 'POB DPS Calculator (replace & compare)';

      btn.addEventListener('click', async (e) => {
        e.preventDefault();
        e.stopPropagation();

        const config = getConfig();
        if (!config.pobCode) {
          showSettings();
          return;
        }

        await calcSingleItem(row, btn);
      });

      const header = row.querySelector('.middle, .itemPopupContainer, [class*="item-info"], [class*="details"]');
      if (header) {
        header.appendChild(btn);
      } else {
        row.insertBefore(btn, row.firstChild);
      }
    }
  }

  // =========================================================================
  // Calc-all + best item highlight
  // =========================================================================

  let calcAllRunning = false;

  async function calcAllItems() {
    if (calcAllRunning) return;

    const config = getConfig();
    if (!config.pobCode) {
      showSettings();
      return;
    }

    calcAllRunning = true;
    const calcAllBtn = document.getElementById('pob-calc-all-btn');
    if (calcAllBtn) {
      calcAllBtn.classList.add('loading');
    }

    injectCalcButtons();

    const allPairs = [];
    const itemBtns = document.querySelectorAll('[data-id] .pob-calc-btn');
    for (const btn of itemBtns) {
      const row = btn.closest('[data-id]');
      if (!row) continue;
      const itemId = row.getAttribute('data-id');
      if (!itemId || !itemDataCache[itemId]) continue;
      allPairs.push({ row, btn, itemId });
    }

    const needCalc = [];
    let cachedCount = 0;
    for (const pair of allPairs) {
      if (calcResultCache[pair.itemId] && calcResultCache[pair.itemId].result) {
        calcResultCache[pair.itemId].row = pair.row;
        cachedCount++;
      } else {
        needCalc.push(pair);
      }
    }

    const totalAll = allPairs.length;

    if (totalAll === 0) {
      if (calcAllBtn) {
        calcAllBtn.classList.remove('loading');
        calcAllBtn.innerHTML = '⚡ Calc All <span class="pob-progress">(nothing to calc)</span>';
        setTimeout(() => { calcAllBtn.innerHTML = '⚡ Calc All DPS'; }, 2000);
      }
      calcAllRunning = false;
      return;
    }

    // Clear previous best markers
    document.querySelectorAll('.pob-best-item').forEach(el => {
      el.classList.remove('pob-best-item');
      el.style.outline = '';
      el.style.outlineOffset = '';
      const badge = el.querySelector('.pob-best-badge');
      if (badge) badge.remove();
    });

    if (needCalc.length === 0) {
      if (calcAllBtn) {
        calcAllBtn.innerHTML = `⚡ All cached (${cachedCount} items), finding best...`;
      }
    }

    const concurrency = config.concurrency || 4;
    let completedNew = 0;

    async function calcWithSemaphore(pair) {
      if (!calcAllRunning) return;
      const { row, btn } = pair;
      await calcSingleItem(row, btn);
      completedNew++;
      if (calcAllBtn) {
        const doneCount = cachedCount + completedNew;
        calcAllBtn.innerHTML = `⚡ Calculating <span class="pob-progress">(${doneCount}/${totalAll}, concurrency ${concurrency})</span>`;
      }
    }

    if (needCalc.length > 0) {
      if (calcAllBtn) {
        calcAllBtn.innerHTML = `⚡ Calculating <span class="pob-progress">(${cachedCount}/${totalAll}, concurrency ${concurrency})</span>`;
      }

      let running = 0;
      let idx = 0;
      await new Promise((resolve) => {
        function tryNext() {
          if (!calcAllRunning && running === 0) { resolve(); return; }
          while (running < concurrency && idx < needCalc.length && calcAllRunning) {
            const pair = needCalc[idx++];
            running++;
            calcWithSemaphore(pair).finally(() => {
              running--;
              if (idx >= needCalc.length && running === 0) {
                resolve();
              } else {
                tryNext();
              }
            });
          }
          if (idx >= needCalc.length && running === 0) {
            resolve();
          }
        }
        tryNext();
      });
    }

    // Find best item
    let bestResult = null;
    let bestDpsGain = -Infinity;
    let successCount = 0;
    let failCount = 0;

    for (const pair of allPairs) {
      const cached = calcResultCache[pair.itemId];
      if (!cached) continue;
      if (cached.result) {
        successCount++;
        if (cached.result.diff) {
          const dpsGain = cached.result.diff.CombinedDPS !== undefined
            ? cached.result.diff.CombinedDPS
            : (cached.result.diff.TotalDPS || 0);
          if (dpsGain > bestDpsGain) {
            bestDpsGain = dpsGain;
            bestResult = { ...cached, row: pair.row };
          }
        }
      } else if (cached.error) {
        failCount++;
      }
    }

    // Highlight best
    if (bestResult && bestDpsGain > -Infinity) {
      const isPositive = bestDpsGain >= 0;
      const highlightColor = isPositive ? '#20c820' : '#d4a017';
      const bestRow = bestResult.row;
      bestRow.classList.add('pob-best-item');
      bestRow.style.outline = `2px solid ${highlightColor}`;
      bestRow.style.outlineOffset = '-2px';

      let badge = bestRow.querySelector('.pob-best-badge');
      if (!badge) {
        badge = document.createElement('div');
        badge.className = 'pob-best-badge';
        badge.style.cssText = `
          position: absolute; top: 4px; right: 8px;
          background: ${highlightColor}; color: #000;
          font-size: 12px; font-weight: bold;
          padding: 3px 10px; border-radius: 3px;
          z-index: 100; pointer-events: none;
          font-family: "FontinSmallCaps", Verdana, sans-serif;
          box-shadow: 0 1px 4px rgba(0,0,0,0.5);
        `;
        const pos = window.getComputedStyle(bestRow).position;
        if (pos === 'static') {
          bestRow.style.position = 'relative';
        }
        bestRow.appendChild(badge);
      }

      badge.textContent = isPositive
        ? `🏆 Best DPS (+${numberFormat(bestDpsGain)})`
        : `⭐ Least DPS Loss (${numberFormat(bestDpsGain)})`;

      bestResult.row.scrollIntoView({ behavior: 'smooth', block: 'center' });
      console.log(`[POB DPS Calc] 🏆 Best item: ${bestResult.itemId}, DPS change: ${bestDpsGain >= 0 ? '+' : ''}${numberFormat(bestDpsGain)}`);
    }

    // Restore button
    if (calcAllBtn) {
      calcAllBtn.classList.remove('loading');
      let statusText = `✅ Done ${successCount}/${totalAll}`;
      if (cachedCount > 0 && needCalc.length > 0) statusText += ` (cached ${cachedCount})`;
      if (failCount > 0) statusText += ` (${failCount} failed)`;
      if (bestResult && bestDpsGain > -Infinity) {
        statusText += bestDpsGain >= 0
          ? ` | 🏆 Best: +${numberFormat(bestDpsGain)} DPS`
          : ` | ⭐ Best: ${numberFormat(bestDpsGain)} DPS`;
      }
      calcAllBtn.innerHTML = `⚡ ${statusText}`;
      setTimeout(() => { calcAllBtn.innerHTML = '⚡ Calc All DPS'; }, 8000);
    }

    calcAllRunning = false;
  }

  function stopCalcAll() {
    calcAllRunning = false;
  }

  function clearCalcCacheAndRecalc() {
    for (const key of Object.keys(calcResultCache)) {
      delete calcResultCache[key];
    }

    document.querySelectorAll('.pob-quick-summary').forEach(el => el.remove());
    document.querySelectorAll('.pob-result-panel').forEach(el => el.remove());

    document.querySelectorAll('.pob-best-item').forEach(el => {
      el.classList.remove('pob-best-item');
      el.style.outline = '';
      el.style.outlineOffset = '';
      const badge = el.querySelector('.pob-best-badge');
      if (badge) badge.remove();
    });

    console.log('[POB DPS Calc] Cleared all calculation cache');
    calcAllItems();
  }

  function createCalcAllButton() {
    const btn = document.createElement('div');
    btn.className = 'pob-calc-all-btn';
    btn.id = 'pob-calc-all-btn';
    btn.innerHTML = '⚡ Calc All DPS';
    btn.title = 'Calculate DPS for all items on this page (cached results will be skipped), highlight the best item';
    btn.addEventListener('click', () => {
      if (calcAllRunning) {
        stopCalcAll();
      } else {
        calcAllItems();
      }
    });
    document.body.appendChild(btn);

    const clearBtn = document.createElement('div');
    clearBtn.className = 'pob-calc-all-btn';
    clearBtn.id = 'pob-clear-cache-btn';
    clearBtn.style.bottom = '100px';
    clearBtn.style.fontSize = '11px';
    clearBtn.style.padding = '5px 10px';
    clearBtn.innerHTML = '🔄 Clear Cache & Recalc';
    clearBtn.title = 'Clear all cached calculation results and recalculate everything';
    clearBtn.addEventListener('click', () => {
      if (calcAllRunning) {
        stopCalcAll();
      } else {
        clearCalcCacheAndRecalc();
      }
    });
    document.body.appendChild(clearBtn);
  }

  // =========================================================================
  // MutationObserver: inject buttons on DOM changes
  // =========================================================================

  const observer = new MutationObserver((mutations) => {
    let shouldInject = false;
    for (const mutation of mutations) {
      if (mutation.addedNodes.length > 0) {
        for (const node of mutation.addedNodes) {
          if (node.nodeType === 1) {
            if (node.hasAttribute && node.hasAttribute('data-id')) {
              shouldInject = true;
            }
            if (node.querySelector && node.querySelector('[data-id]')) {
              shouldInject = true;
            }
          }
        }
      }
    }
    if (shouldInject) {
      setTimeout(() => injectCalcButtons(), 300);
    }
  });

  observer.observe(document.body, {
    childList: true,
    subtree: true,
  });

  // =========================================================================
  // Settings panel
  // =========================================================================

  function showSettings() {
    const existing = document.querySelector('.pob-settings-overlay');
    if (existing) existing.remove();

    const config = getConfig();

    const overlay = document.createElement('div');
    overlay.className = 'pob-settings-overlay';
    overlay.addEventListener('click', (e) => {
      if (e.target === overlay) overlay.remove();
    });

    const panel = document.createElement('div');
    panel.className = 'pob-settings-panel';

    panel.innerHTML = `
      <h2>⚡ POB DPS Calculator Settings</h2>

      <label for="pob-api-url">API URL</label>
      <input type="text" id="pob-api-url" value="${config.apiUrl}" placeholder="http://localhost:8080" />
      <div class="pob-hint">POB Recalc API server address (conversion + calculation are done server-side)</div>

      <label for="pob-code">POB Code (share code)</label>
      <textarea id="pob-code" placeholder="Paste your Path of Building export code here (eNrtPdl2...)">${config.pobCode}</textarea>
      <div class="pob-hint">Export your build share code from Path of Building and paste it here</div>

      <label for="pob-slot">Default Equipment Slot</label>
      <select id="pob-slot">
        <option value="auto" ${config.defaultSlot === 'auto' ? 'selected' : ''}>Auto Detect</option>
        <option value="Helmet" ${config.defaultSlot === 'Helmet' ? 'selected' : ''}>Helmet</option>
        <option value="Body Armour" ${config.defaultSlot === 'Body Armour' ? 'selected' : ''}>Body Armour</option>
        <option value="Gloves" ${config.defaultSlot === 'Gloves' ? 'selected' : ''}>Gloves</option>
        <option value="Boots" ${config.defaultSlot === 'Boots' ? 'selected' : ''}>Boots</option>
        <option value="Belt" ${config.defaultSlot === 'Belt' ? 'selected' : ''}>Belt</option>
        <option value="Amulet" ${config.defaultSlot === 'Amulet' ? 'selected' : ''}>Amulet</option>
        <option value="Ring 1" ${config.defaultSlot === 'Ring 1' ? 'selected' : ''}>Ring 1</option>
        <option value="Ring 2" ${config.defaultSlot === 'Ring 2' ? 'selected' : ''}>Ring 2</option>
        <option value="Weapon 1" ${config.defaultSlot === 'Weapon 1' ? 'selected' : ''}>Weapon 1</option>
        <option value="Weapon 2" ${config.defaultSlot === 'Weapon 2' ? 'selected' : ''}>Weapon 2</option>
      </select>
      <div class="pob-hint">"Auto Detect" lets the server determine slot from item type</div>

      <label for="pob-concurrency">Batch Concurrency</label>
      <input type="number" id="pob-concurrency" value="${config.concurrency}" min="1" max="20" step="1" />
      <div class="pob-hint">Number of parallel calculations (1=serial, recommended 2~8, default 4)</div>

      <div class="pob-btn-row">
        <button id="pob-test-btn">🔗 Test Connection</button>
        <button id="pob-cancel-btn">Cancel</button>
        <button id="pob-save-btn" class="pob-primary">Save</button>
      </div>
    `;

    overlay.appendChild(panel);
    document.body.appendChild(overlay);

    document.getElementById('pob-save-btn').addEventListener('click', () => {
      const apiUrl = document.getElementById('pob-api-url').value.trim().replace(/\/$/, '');
      const pobCode = document.getElementById('pob-code').value.trim();
      const slot = document.getElementById('pob-slot').value;
      const concurrency = Math.max(1, Math.min(20, parseInt(document.getElementById('pob-concurrency').value, 10) || 4));

      GM_setValue('apiUrl', apiUrl);
      GM_setValue('pobCode', pobCode);
      GM_setValue('defaultSlot', slot);
      GM_setValue('concurrency', concurrency);

      overlay.remove();
      updateFloatButton();
    });

    document.getElementById('pob-cancel-btn').addEventListener('click', () => {
      overlay.remove();
    });

    document.getElementById('pob-test-btn').addEventListener('click', async () => {
      const btn = document.getElementById('pob-test-btn');
      const testUrl = document.getElementById('pob-api-url').value.trim().replace(/\/$/, '');
      btn.textContent = '⏳ Testing...';
      btn.disabled = true;

      const ok = await new Promise((resolve) => {
        GM_xmlhttpRequest({
          method: 'GET',
          url: `${testUrl}/health`,
          timeout: 5000,
          onload: (r) => resolve(r.status === 200),
          onerror: () => resolve(false),
          ontimeout: () => resolve(false),
        });
      });

      btn.textContent = ok ? '✅ Connected!' : '❌ Connection Failed';
      btn.disabled = false;
      setTimeout(() => { btn.textContent = '🔗 Test Connection'; }, 2000);
    });
  }

  // =========================================================================
  // Float entry button
  // =========================================================================

  function createFloatButton() {
    const btn = document.createElement('div');
    btn.className = 'pob-float-entry';
    btn.id = 'pob-float-entry';
    btn.innerHTML = '<span class="pob-status-dot disconnected"></span>POB DPS';
    btn.addEventListener('click', showSettings);
    document.body.appendChild(btn);
    updateFloatButton();
  }

  async function updateFloatButton() {
    const btn = document.getElementById('pob-float-entry');
    if (!btn) return;

    const dot = btn.querySelector('.pob-status-dot');
    const config = getConfig();

    if (!config.pobCode) {
      dot.className = 'pob-status-dot disconnected';
      btn.title = 'Click to set POB Code';
      return;
    }

    const healthy = await checkApiHealth();
    dot.className = `pob-status-dot ${healthy ? 'connected' : 'disconnected'}`;
    btn.title = healthy ? 'POB API Connected' : 'POB API Disconnected';
  }

  // =========================================================================
  // Tampermonkey menu commands
  // =========================================================================

  GM_registerMenuCommand('⚙️ POB DPS Calculator Settings', showSettings);
  GM_registerMenuCommand('🔄 Refresh Calc Buttons', injectCalcButtons);
  GM_registerMenuCommand('⚡ Calc All DPS', calcAllItems);
  GM_registerMenuCommand('🗑️ Clear Cache & Recalc', clearCalcCacheAndRecalc);

  // =========================================================================
  // Initialize
  // =========================================================================

  console.log('[POB DPS Calc] Userscript v1.0.0 loaded (Global server, no translation mode)');

  createFloatButton();
  createCalcAllButton();
  setTimeout(() => injectCalcButtons(), 1000);
  setInterval(() => updateFloatButton(), 30000);

})();
