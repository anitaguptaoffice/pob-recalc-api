// ==UserScript==
// @name         POE 中文市集 DPS 计算器
// @namespace    https://github.com/pob-recalc-api
// @version      2.0.0
// @description  在中文流放之路市集 (poe.game.qq.com) 每件装备旁插入DPS计算按钮，调用 POB Recalc API 对比装备替换前后属性变化。翻译在服务端完成，支持 12800+ 词缀模板。
// @author       POB Recalc API
// @match        https://poe.game.qq.com/trade/*
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
  // 配置
  // =========================================================================

  const DEFAULT_CONFIG = {
    apiUrl: 'http://localhost:8080',
    pobCode: '',
    defaultSlot: 'auto', // auto = 由服务端根据物品类型自动判断
  };

  function getConfig() {
    return {
      apiUrl: GM_getValue('apiUrl', DEFAULT_CONFIG.apiUrl),
      pobCode: GM_getValue('pobCode', DEFAULT_CONFIG.pobCode),
      defaultSlot: GM_getValue('defaultSlot', DEFAULT_CONFIG.defaultSlot),
    };
  }

  // =========================================================================
  // 样式注入
  // =========================================================================

  GM_addStyle(`
    /* 计算按钮 */
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

    /* DPS 结果面板 */
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

    /* 设置面板 */
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

    /* 浮动入口按钮 */
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

    /* 快速摘要 */
    .pob-quick-summary {
      display: inline-block;
      margin-left: 6px;
      font-size: 11px;
      font-weight: bold;
      font-family: "FontinSmallCaps", Verdana, sans-serif;
      vertical-align: middle;
    }
  `);

  // =========================================================================
  // API 调用 — 所有翻译在服务端完成
  // =========================================================================

  /**
   * 调用 /translate-item: 将单个中文物品 JSON 翻译为英文 POB item text
   * @param {Object} itemData - 国服 trade API 返回的物品 JSON 对象
   * @returns {Promise<{item_text: string, slot: string}>}
   */
  function callTranslateItemAPI(itemData) {
    return new Promise((resolve, reject) => {
      const config = getConfig();
      GM_xmlhttpRequest({
        method: 'POST',
        url: `${config.apiUrl}/translate-item`,
        headers: { 'Content-Type': 'application/json' },
        data: JSON.stringify(itemData),
        timeout: 15000,
        onload: function (response) {
          if (response.status === 200) {
            try {
              resolve(JSON.parse(response.responseText));
            } catch (e) {
              reject(new Error('JSON 解析失败: ' + e.message));
            }
          } else {
            reject(new Error(`翻译 API 错误 (${response.status}): ${response.responseText}`));
          }
        },
        onerror: function () {
          reject(new Error('网络错误: 无法连接到翻译 API'));
        },
        ontimeout: function () {
          reject(new Error('翻译请求超时 (15s)'));
        },
      });
    });
  }

  /**
   * 调用 /replace-item: 用英文 item text 替换 POB build 中的装备并对比
   * @param {string} pobCode
   * @param {string} slot
   * @param {string} itemText - 已翻译的英文 POB item text
   * @returns {Promise<Object>} - DPS 对比结果
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
              reject(new Error('JSON 解析失败: ' + e.message));
            }
          } else {
            reject(new Error(`API 错误 (${response.status}): ${response.responseText}`));
          }
        },
        onerror: function () {
          reject(new Error('网络错误: 无法连接到 POB API。请确认服务在 ' + config.apiUrl + ' 运行'));
        },
        ontimeout: function () {
          reject(new Error('请求超时 (60s)'));
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
  // 结果面板渲染
  // =========================================================================

  function renderResultPanel(result, container) {
    // 清除旧面板
    const old = container.querySelector('.pob-result-panel');
    if (old) old.remove();

    const panel = document.createElement('div');
    panel.className = 'pob-result-panel';

    const closeBtn = document.createElement('span');
    closeBtn.className = 'pob-close-btn';
    closeBtn.textContent = '✕';
    closeBtn.onclick = () => panel.remove();
    panel.appendChild(closeBtn);

    // 重要 DPS 统计
    const dpsStats = [
      { key: 'CombinedDPS', label: '综合 DPS' },
      { key: 'TotalDPS', label: '总 DPS' },
      { key: 'TotalDot', label: 'DoT DPS' },
      { key: 'AverageDamage', label: '平均伤害' },
      { key: 'Speed', label: '攻速/施法' },
      { key: 'CritChance', label: '暴击率' },
      { key: 'CritMultiplier', label: '暴击伤害' },
    ];

    const defStats = [
      { key: 'Life', label: '生命' },
      { key: 'EnergyShield', label: '能量护盾' },
      { key: 'Mana', label: '魔力' },
      { key: 'ManaUnreserved', label: '可用魔力' },
      { key: 'Armour', label: '护甲' },
      { key: 'Evasion', label: '闪避' },
      { key: 'BlockChance', label: '格挡率' },
      { key: 'SpellBlockChance', label: '法术格挡' },
    ];

    const resStats = [
      { key: 'FireResist', label: '火抗' },
      { key: 'ColdResist', label: '冰抗' },
      { key: 'LightningResist', label: '电抗' },
      { key: 'ChaosResist', label: '混沌抗' },
    ];

    function addSection(title, stats) {
      const titleEl = document.createElement('div');
      titleEl.className = 'pob-section-title';
      titleEl.textContent = title;
      panel.appendChild(titleEl);

      for (const stat of stats) {
        const diff = result.diff[stat.key];
        if (diff === undefined || diff === null) continue;
        // 只显示有变化的行，或者是关键指标
        const isKey = ['CombinedDPS', 'TotalDPS', 'Life', 'EnergyShield'].includes(stat.key);
        if (!isKey && Math.abs(diff) < 0.01) continue;

        const row = document.createElement('div');
        row.className = 'pob-stat-row';

        const nameSpan = document.createElement('span');
        nameSpan.className = 'pob-stat-name';
        nameSpan.textContent = stat.label;

        const valSpan = document.createElement('span');
        valSpan.className = 'pob-stat-val';

        // 格式化数值
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

        // 颜色
        if (Math.abs(diff) < 0.01) {
          valSpan.classList.add('neutral');
          formatted = '—';
        } else if (diff > 0) {
          valSpan.classList.add('positive');
        } else {
          valSpan.classList.add('negative');
        }

        // 附加 before/after 信息
        const beforeVal = result.before[stat.key] || 0;
        const afterVal = result.after[stat.key] || 0;
        valSpan.title = `${numberFormat(beforeVal)} → ${numberFormat(afterVal)}`;
        valSpan.textContent = formatted;

        row.appendChild(nameSpan);
        row.appendChild(valSpan);
        panel.appendChild(row);
      }
    }

    addSection('⚔️ 伤害', dpsStats);
    addSection('🛡️ 防御', defStats);
    addSection('🔥 抗性', resStats);

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
  // 拦截 fetch 获取物品 API 数据
  // =========================================================================

  // 存储从 API 获取的物品数据 (item id → item data)
  const itemDataCache = {};

  // 拦截 fetch 以捕获市集 API 响应
  const origFetch = window.fetch;
  window.fetch = async function (...args) {
    const response = await origFetch.apply(this, args);
    const url = typeof args[0] === 'string' ? args[0] : (args[0] && args[0].url) || '';

    // 捕获 /api/trade/fetch 或 /api/trade2/fetch 的响应 (包含物品详情)
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
          console.log(`[POB DPS Calc] 缓存了 ${data.result.length} 个物品数据`);
          // 数据到位后，尝试注入按钮
          setTimeout(() => injectCalcButtons(), 500);
        }
      } catch (e) {
        // 忽略解析错误
      }
    }

    return response;
  };

  // 同样拦截 XMLHttpRequest (作为后备)
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
          // 忽略
        }
      });
    }
    return origXHRSend.apply(this, args);
  };

  // =========================================================================
  // 按钮注入逻辑
  // =========================================================================

  function injectCalcButtons() {
    // 查找所有搜索结果中的物品行
    const itemRows = document.querySelectorAll(
      '.resultset .row, [class*="search-result"], [class*="result-row"], [data-id]'
    );

    if (itemRows.length === 0) {
      return;
    }

    for (const row of itemRows) {
      // 避免重复注入
      if (row.querySelector('.pob-calc-btn')) continue;

      const itemId = row.getAttribute('data-id') || row.id || '';

      // 创建计算按钮
      const btn = document.createElement('button');
      btn.className = 'pob-calc-btn';
      btn.innerHTML = '⚡ 计算DPS';
      btn.title = 'POB DPS 计算 (替换对比)';

      btn.addEventListener('click', async (e) => {
        e.preventDefault();
        e.stopPropagation();

        const config = getConfig();
        if (!config.pobCode) {
          showSettings();
          return;
        }

        // 从缓存中获取原始物品 JSON (来自国服 trade API)
        const rawItemData = itemId ? itemDataCache[itemId] : null;
        if (!rawItemData) {
          renderErrorPanel('未找到该物品的 API 数据，请刷新页面后重试', row);
          return;
        }

        console.log('[POB DPS Calc] 原始物品 JSON:', rawItemData);

        // 显示加载状态
        btn.classList.add('loading');
        btn.innerHTML = '翻译中';
        btn.disabled = true;

        try {
          // Step 1: 服务端翻译 — 发送原始中文物品 JSON
          const translateResult = await callTranslateItemAPI(rawItemData);
          const itemText = translateResult.item_text;
          const slot = config.defaultSlot === 'auto' ? translateResult.slot : config.defaultSlot;

          console.log('[POB DPS Calc] 翻译结果:', { slot, itemText });

          // Step 2: 替换对比 — 用翻译后的英文 item text
          btn.innerHTML = '计算中';
          const result = await callReplaceItemAPI(config.pobCode, slot, itemText);
          console.log('[POB DPS Calc] 对比结果:', JSON.stringify(result, null, 2));

          // 在按钮旁边显示简短摘要，方便确认结果
          const dpsChange = result.diff && result.diff.TotalDPS !== undefined ? result.diff.TotalDPS : null;
          const lifeChange = result.diff && result.diff.Life !== undefined ? result.diff.Life : null;
          let summary = '';
          if (dpsChange !== null) {
            summary += `DPS: ${dpsChange >= 0 ? '+' : ''}${numberFormat(dpsChange)}`;
          }
          if (lifeChange !== null && Math.abs(lifeChange) >= 1) {
            summary += ` | 生命: ${lifeChange >= 0 ? '+' : ''}${Math.round(lifeChange)}`;
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

          renderResultPanel(result, row);
        } catch (err) {
          console.error('[POB DPS Calc] 错误:', err);
          renderErrorPanel(err.message, row);
        } finally {
          btn.classList.remove('loading');
          btn.innerHTML = '⚡ 计算DPS';
          btn.disabled = false;
        }
      });

      // 将按钮插入到物品行的合适位置
      const header = row.querySelector('.middle, .itemPopupContainer, [class*="item-info"], [class*="details"]');
      if (header) {
        header.appendChild(btn);
      } else {
        row.insertBefore(btn, row.firstChild);
      }
    }
  }

  // =========================================================================
  // MutationObserver: 监听DOM变化，持续注入按钮
  // =========================================================================

  const observer = new MutationObserver((mutations) => {
    let shouldInject = false;
    for (const mutation of mutations) {
      if (mutation.addedNodes.length > 0) {
        for (const node of mutation.addedNodes) {
          if (node.nodeType === 1) {
            if (node.matches && (
              node.matches('[data-id]') ||
              node.matches('.row') ||
              node.matches('[class*="result"]')
            )) {
              shouldInject = true;
            }
            if (node.querySelector && node.querySelector('[data-id], .row')) {
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
  // 设置面板
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
      <h2>⚡ POB DPS 计算器 设置</h2>

      <label for="pob-api-url">API 地址</label>
      <input type="text" id="pob-api-url" value="${config.apiUrl}" placeholder="http://localhost:8080" />
      <div class="pob-hint">POB Recalc API 服务的地址（翻译+计算均在服务端完成）</div>

      <label for="pob-code">POB Code (分享码)</label>
      <textarea id="pob-code" placeholder="从 Path of Building 导出的分享码 (eNrtPdl2...)">${config.pobCode}</textarea>
      <div class="pob-hint">在 Path of Building 中导出你的 build 分享码，粘贴到这里</div>

      <label for="pob-slot">默认装备槽位</label>
      <select id="pob-slot">
        <option value="auto" ${config.defaultSlot === 'auto' ? 'selected' : ''}>自动检测</option>
        <option value="Helmet" ${config.defaultSlot === 'Helmet' ? 'selected' : ''}>头盔 (Helmet)</option>
        <option value="Body Armour" ${config.defaultSlot === 'Body Armour' ? 'selected' : ''}>胸甲 (Body Armour)</option>
        <option value="Gloves" ${config.defaultSlot === 'Gloves' ? 'selected' : ''}>手套 (Gloves)</option>
        <option value="Boots" ${config.defaultSlot === 'Boots' ? 'selected' : ''}>鞋子 (Boots)</option>
        <option value="Belt" ${config.defaultSlot === 'Belt' ? 'selected' : ''}>腰带 (Belt)</option>
        <option value="Amulet" ${config.defaultSlot === 'Amulet' ? 'selected' : ''}>项链 (Amulet)</option>
        <option value="Ring 1" ${config.defaultSlot === 'Ring 1' ? 'selected' : ''}>戒指 1 (Ring 1)</option>
        <option value="Ring 2" ${config.defaultSlot === 'Ring 2' ? 'selected' : ''}>戒指 2 (Ring 2)</option>
        <option value="Weapon 1" ${config.defaultSlot === 'Weapon 1' ? 'selected' : ''}>主手 (Weapon 1)</option>
        <option value="Weapon 2" ${config.defaultSlot === 'Weapon 2' ? 'selected' : ''}>副手 (Weapon 2)</option>
      </select>
      <div class="pob-hint">"自动检测" 由服务端根据物品类型自动判断</div>

      <div class="pob-btn-row">
        <button id="pob-test-btn">🔗 测试连接</button>
        <button id="pob-cancel-btn">取消</button>
        <button id="pob-save-btn" class="pob-primary">保存</button>
      </div>
    `;

    overlay.appendChild(panel);
    document.body.appendChild(overlay);

    // 事件绑定
    document.getElementById('pob-save-btn').addEventListener('click', () => {
      const apiUrl = document.getElementById('pob-api-url').value.trim().replace(/\/$/, '');
      const pobCode = document.getElementById('pob-code').value.trim();
      const slot = document.getElementById('pob-slot').value;

      GM_setValue('apiUrl', apiUrl);
      GM_setValue('pobCode', pobCode);
      GM_setValue('defaultSlot', slot);

      overlay.remove();
      updateFloatButton();
    });

    document.getElementById('pob-cancel-btn').addEventListener('click', () => {
      overlay.remove();
    });

    document.getElementById('pob-test-btn').addEventListener('click', async () => {
      const btn = document.getElementById('pob-test-btn');
      const testUrl = document.getElementById('pob-api-url').value.trim().replace(/\/$/, '');
      btn.textContent = '⏳ 测试中...';
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

      btn.textContent = ok ? '✅ 连接成功!' : '❌ 连接失败';
      btn.disabled = false;
      setTimeout(() => { btn.textContent = '🔗 测试连接'; }, 2000);
    });
  }

  // =========================================================================
  // 浮动入口按钮
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
      btn.title = '点击设置 POB Code';
      return;
    }

    const healthy = await checkApiHealth();
    dot.className = `pob-status-dot ${healthy ? 'connected' : 'disconnected'}`;
    btn.title = healthy ? 'POB API 已连接' : 'POB API 未连接';
  }

  // =========================================================================
  // Tampermonkey 菜单命令
  // =========================================================================

  GM_registerMenuCommand('⚙️ POB DPS 计算器设置', showSettings);
  GM_registerMenuCommand('🔄 刷新计算按钮', injectCalcButtons);

  // =========================================================================
  // 初始化
  // =========================================================================

  console.log('[POB DPS Calc] 油猴脚本 v2.0.0 已加载 (服务端翻译模式)');

  // 创建浮动入口
  createFloatButton();

  // 初次尝试注入按钮
  setTimeout(() => injectCalcButtons(), 1000);

  // 定期检查连接状态
  setInterval(() => updateFloatButton(), 30000);

})();
