// ==UserScript==
// @name         POE 中文市集 DPS 计算器
// @namespace    https://github.com/pob-recalc-api
// @version      1.0.0
// @description  在中文流放之路市集 (poe.game.qq.com) 每件装备旁插入DPS计算按钮，调用 POB Recalc API 对比装备替换前后属性变化
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
    defaultSlot: 'auto', // auto = 根据物品类型自动判断
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
  `);

  // =========================================================================
  // 中文 → 英文 翻译映射
  // =========================================================================
  //
  // 参考 awakened-poe-trade (SnosMe/awakened-poe-trade) 的数据结构
  // 这里提供一个精简但覆盖面足够的映射表
  // 模式: 中文 mod 文本中的 # 代表数值占位符
  //

  // --- 稀有度 ---
  const RARITY_ZH_TO_EN = {
    '普通': 'NORMAL',
    '魔法': 'MAGIC',
    '稀有': 'RARE',
    '传奇': 'UNIQUE',
  };

  // --- 物品类别 (Class) → 英文 ---
  const ITEM_CLASS_ZH_TO_EN = {
    // 武器
    '爪': 'Claws',
    '匕首': 'Daggers',
    '符文匕首': 'Rune Daggers',
    '单手剑': 'One Hand Swords',
    '细剑': 'Thrusting One Hand Swords',
    '单手斧': 'One Hand Axes',
    '单手锤': 'One Hand Maces',
    '权杖': 'Sceptres',
    '法杖': 'Wands',
    '双手剑': 'Two Hand Swords',
    '双手斧': 'Two Hand Axes',
    '双手锤': 'Two Hand Maces',
    '弓': 'Bows',
    '长杖': 'Staves',
    '战杖': 'Warstaves',
    // 护甲
    '头部': 'Helmets',
    '胸甲': 'Body Armours',
    '手套': 'Gloves',
    '鞋子': 'Boots',
    '盾': 'Shields',
    // 配饰
    '项链': 'Amulets',
    '戒指': 'Rings',
    '腰带': 'Belts',
    '箭袋': 'Quivers',
    // 药剂
    '药剂': 'Flasks',
    // 珠宝
    '珠宝': 'Jewels',
    '深渊珠宝': 'Abyss Jewels',
  };

  // --- 物品类别 → 装备槽位 ---
  const CLASS_TO_SLOT = {
    'Claws': 'Weapon 1',
    'Daggers': 'Weapon 1',
    'Rune Daggers': 'Weapon 1',
    'One Hand Swords': 'Weapon 1',
    'Thrusting One Hand Swords': 'Weapon 1',
    'One Hand Axes': 'Weapon 1',
    'One Hand Maces': 'Weapon 1',
    'Sceptres': 'Weapon 1',
    'Wands': 'Weapon 1',
    'Two Hand Swords': 'Weapon 1',
    'Two Hand Axes': 'Weapon 1',
    'Two Hand Maces': 'Weapon 1',
    'Bows': 'Weapon 1',
    'Staves': 'Weapon 1',
    'Warstaves': 'Weapon 1',
    'Helmets': 'Helmet',
    'Body Armours': 'Body Armour',
    'Gloves': 'Gloves',
    'Boots': 'Boots',
    'Shields': 'Weapon 2',
    'Amulets': 'Amulet',
    'Rings': 'Ring 1',
    'Belts': 'Belt',
    'Quivers': 'Weapon 2',
    'Flasks': 'Flask 1',
    'Jewels': 'Jewel 1',
    'Abyss Jewels': 'Jewel 1',
  };

  // --- 基底名称 中 → 英 (常见基底精选) ---
  // 这是一个大表，涵盖常用基底。如果无法匹配，会保留原名。
  const BASE_TYPE_ZH_TO_EN = {
    // 头盔
    '铁帽': 'Iron Hat',
    '锥盔': 'Cone Helmet',
    '远古盔': 'Great Helmet',
    '铁冠': 'Iron Circlet',
    '骨盔': 'Bone Helmet',
    '远古面罩': 'Praetor Crown',
    '终末头盔': 'Hubris Circlet',
    '预言冠': 'Prophet Crown',
    '黑暗兜帽': 'Sinner Tricorne',
    '皮兜帽': 'Leather Hood',
    '雄狮盔': 'Lion Pelt',
    '永恒铸铁头盔': 'Eternal Burgonet',
    '战士面甲': 'Solaris Circlet',
    '刺客帽': 'Assassin\'s Garb',
    // 胸甲
    '朴素长袍': 'Simple Robe',
    '执政官铠甲': 'Vaal Regalia',
    '光辉战铠': 'Glorious Plate',
    '杀手外衣': 'Assassin\'s Garb',
    '精工锁甲': 'Full Wyrmscale',
    '圣战甲': 'Crusader Plate',
    '执法者外衣': 'Carnal Armour',
    '刺客之衣': 'Assassin\'s Garb',
    '天灾铠甲': 'Astral Plate',
    '幽影外衣': 'Sadist Garb',
    '至尊法袍': 'Vaal Regalia',
    // 手套
    '铁手套': 'Iron Gauntlets',
    '丝绸手套': 'Silk Gloves',
    '黄金手套': 'Sorcerer Gloves',
    '暗影手套': 'Gripped Gloves',
    '指骨手套': 'Fingerless Silk Gloves',
    '刺客手套': 'Slink Gloves',
    '铸铁手套': 'Titan Gauntlets',
    '炼金手套': 'Apothecary\'s Gloves',
    // 鞋子
    '铁靴': 'Iron Greaves',
    '丝绸鞋': 'Silk Slippers',
    '暗影长靴': 'Two-Toned Boots',
    '铸铁长靴': 'Titan Greaves',
    '黄金战靴': 'Sorcerer Boots',
    '刺客靴': 'Slink Boots',
    // 盾牌
    '松木盾': 'Pine Buckler',
    '精工塔盾': 'Titanium Spirit Shield',
    '战灵盾': 'Fossilised Spirit Shield',
    '泰坦盾': 'Ezomyte Tower Shield',
    // 腰带
    '皮带': 'Leather Belt',
    '重型腰带': 'Heavy Belt',
    '布质腰带': 'Cloth Belt',
    '锈蚀腰带': 'Rustic Sash',
    '猎首腰带': 'Stygian Vise',
    '水晶腰带': 'Crystal Belt',
    // 戒指
    '铁戒指': 'Iron Ring',
    '珊瑚戒指': 'Coral Ring',
    '红玉戒指': 'Ruby Ring',
    '蓝玉戒指': 'Sapphire Ring',
    '黄玉戒指': 'Topaz Ring',
    '黄金戒指': 'Gold Ring',
    '钻石戒指': 'Diamond Ring',
    '棱光戒指': 'Prismatic Ring',
    '紫晶戒指': 'Amethyst Ring',
    '双石戒指': 'Two-Stone Ring',
    '牛角戒指': 'Vermillion Ring',
    '偏转戒指': 'Cerulean Ring',
    '月石戒指': 'Moonstone Ring',
    '钢戒指': 'Steel Ring',
    '蛋白石戒指': 'Opal Ring',
    // 项链
    '珊瑚护身符': 'Coral Amulet',
    '玛瑙护身符': 'Onyx Amulet',
    '翡翠护身符': 'Jade Amulet',
    '青玉护身符': 'Lapis Amulet',
    '琥珀护身符': 'Amber Amulet',
    '黄金护身符': 'Gold Amulet',
    '海玻璃护身符': 'Seaglass Amulet',
    '柑晶护身符': 'Citrine Amulet',
    '朱砂护身符': 'Marble Amulet',
    '涡流护身符': 'Vaal Amulet',
    '碧玉护身符': 'Turquoise Amulet',
    '黑曜石护身符': 'Agate Amulet',
    '结晶护身符': 'Simplex Amulet',
    '天青石护身符': 'Blue Pearl Amulet',
    // 箭袋
    '利箭箭袋': 'Broadhead Arrow Quiver',
    '穿刺箭袋': 'Penetrating Arrow Quiver',
    '火焰箭袋': 'Fire Arrow Quiver',
    '冰霜箭袋': 'Void Arrow Quiver',
    '闪电箭袋': 'Spike-Point Arrow Quiver',
    '幽影箭袋': 'Primal Arrow Quiver',
  };

  // --- 词缀 中 → 英 映射 (核心翻译表) ---
  // # 代表数值占位符，翻译时把数字回填进去
  // 格式: { zh: '中文模板', en: '英文模板' }
  const MOD_TRANSLATIONS = [
    // === 生命/魔力/能量护盾 ===
    { zh: '+# 最大生命', en: '+# to maximum Life' },
    { zh: '+# 最大魔力', en: '+# to maximum Mana' },
    { zh: '+# 最大能量护盾', en: '+# to maximum Energy Shield' },
    { zh: '每秒回复 # 生命', en: 'Regenerate # Life per second' },
    { zh: '每秒回复 #% 生命', en: 'Regenerate #% of Life per second' },
    { zh: '每秒回复 # 魔力', en: 'Regenerate # Mana per second' },
    { zh: '每秒回复 #% 魔力', en: 'Regenerate #% of Mana per second' },
    { zh: '每秒回复 # 能量护盾', en: 'Regenerate # Energy Shield per second' },
    { zh: '每秒回复 #% 能量护盾', en: 'Regenerate #% of Energy Shield per second' },
    { zh: '生命回复速度提高 #%', en: '#% increased Life Recovery rate' },
    { zh: '生命回复速度加快 #%', en: '#% increased Life Recovery rate' },

    // === 属性 ===
    { zh: '+# 力量', en: '+# to Strength' },
    { zh: '+# 敏捷', en: '+# to Dexterity' },
    { zh: '+# 智慧', en: '+# to Intelligence' },
    { zh: '+# 全属性', en: '+# to all Attributes' },
    { zh: '+# 力量和敏捷', en: '+# to Strength and Dexterity' },
    { zh: '+# 力量和智慧', en: '+# to Strength and Intelligence' },
    { zh: '+# 敏捷和智慧', en: '+# to Dexterity and Intelligence' },

    // === 抗性 ===
    { zh: '+#% 火焰抗性', en: '+#% to Fire Resistance' },
    { zh: '+#% 冰霜抗性', en: '+#% to Cold Resistance' },
    { zh: '+#% 闪电抗性', en: '+#% to Lightning Resistance' },
    { zh: '+#% 混沌抗性', en: '+#% to Chaos Resistance' },
    { zh: '+#% 所有元素抗性', en: '+#% to all Elemental Resistances' },
    { zh: '+#% 火焰与冰霜抗性', en: '+#% to Fire and Cold Resistances' },
    { zh: '+#% 火焰与闪电抗性', en: '+#% to Fire and Lightning Resistances' },
    { zh: '+#% 冰霜与闪电抗性', en: '+#% to Cold and Lightning Resistances' },

    // === 伤害 (物理) ===
    { zh: '攻击附加 # - # 基础物理伤害', en: 'Adds # to # Physical Damage to Attacks' },
    { zh: '物理伤害提高 #%', en: '#% increased Physical Damage' },
    { zh: '全域物理伤害提高 #%', en: '#% increased Global Physical Damage' },
    { zh: '近战物理伤害提高 #%', en: '#% increased Melee Physical Damage' },

    // === 伤害 (元素) ===
    { zh: '攻击附加 # - # 基础火焰伤害', en: 'Adds # to # Fire Damage to Attacks' },
    { zh: '攻击附加 # - # 基础冰霜伤害', en: 'Adds # to # Cold Damage to Attacks' },
    { zh: '攻击附加 # - # 基础闪电伤害', en: 'Adds # to # Lightning Damage to Attacks' },
    { zh: '攻击附加 # - # 基础混沌伤害', en: 'Adds # to # Chaos Damage to Attacks' },
    { zh: '法术附加 # - # 基础火焰伤害', en: 'Adds # to # Fire Damage to Spells' },
    { zh: '法术附加 # - # 基础冰霜伤害', en: 'Adds # to # Cold Damage to Spells' },
    { zh: '法术附加 # - # 基础闪电伤害', en: 'Adds # to # Lightning Damage to Spells' },
    { zh: '法术附加 # - # 基础混沌伤害', en: 'Adds # to # Chaos Damage to Spells' },
    { zh: '附加 # - # 基础火焰伤害', en: 'Adds # to # Fire Damage' },
    { zh: '附加 # - # 基础冰霜伤害', en: 'Adds # to # Cold Damage' },
    { zh: '附加 # - # 基础闪电伤害', en: 'Adds # to # Lightning Damage' },
    { zh: '附加 # - # 基础混沌伤害', en: 'Adds # to # Chaos Damage' },
    { zh: '附加 # - # 基础物理伤害', en: 'Adds # to # Physical Damage' },

    // === 伤害提高 ===
    { zh: '火焰伤害提高 #%', en: '#% increased Fire Damage' },
    { zh: '冰霜伤害提高 #%', en: '#% increased Cold Damage' },
    { zh: '闪电伤害提高 #%', en: '#% increased Lightning Damage' },
    { zh: '混沌伤害提高 #%', en: '#% increased Chaos Damage' },
    { zh: '元素伤害提高 #%', en: '#% increased Elemental Damage' },
    { zh: '法术伤害提高 #%', en: '#% increased Spell Damage' },
    { zh: '攻击伤害提高 #%', en: '#% increased Attack Damage' },
    { zh: '伤害提高 #%', en: '#% increased Damage' },
    { zh: '范围伤害提高 #%', en: '#% increased Area Damage' },
    { zh: '投射物伤害提高 #%', en: '#% increased Projectile Damage' },
    { zh: '击中和持续伤害总增 #%', en: '#% more Damage with Hits and Ailments' },
    { zh: '持续伤害提高 #%', en: '#% increased Damage over Time' },
    { zh: '持续伤害总增 #%', en: '#% more Damage over Time' },

    // === 暴击 ===
    { zh: '暴击率提高 #%', en: '#% increased Critical Strike Chance' },
    { zh: '全域暴击率提高 #%', en: '#% increased Global Critical Strike Chance' },
    { zh: '法术暴击率提高 #%', en: '#% increased Critical Strike Chance for Spells' },
    { zh: '+#% 暴击伤害加成', en: '+#% to Critical Strike Multiplier' },
    { zh: '+#% 全域暴击伤害加成', en: '+#% to Global Critical Strike Multiplier' },
    { zh: '全域暴击伤害加成 +#%', en: '+#% to Global Critical Strike Multiplier' },
    { zh: '法术暴击伤害加成 +#%', en: '+#% to Critical Strike Multiplier for Spells' },

    // === 攻速/施法速度 ===
    { zh: '攻击速度加快 #%', en: '#% increased Attack Speed' },
    { zh: '施法速度加快 #%', en: '#% increased Cast Speed' },
    { zh: '攻击和施法速度加快 #%', en: '#% increased Attack and Cast Speed' },

    // === 护甲/闪避/格挡 ===
    { zh: '+# 护甲', en: '+# to Armour' },
    { zh: '护甲提高 #%', en: '#% increased Armour' },
    { zh: '+# 闪避值', en: '+# to Evasion Rating' },
    { zh: '闪避值提高 #%', en: '#% increased Evasion Rating' },
    { zh: '全域闪避值提高 #%', en: '#% increased Global Evasion Rating' },
    { zh: '#% 额外格挡率', en: '+#% Chance to Block' },
    { zh: '+#% 格挡攻击伤害几率', en: '+#% Chance to Block Attack Damage' },
    { zh: '+#% 格挡法术伤害几率', en: '+#% Chance to Block Spell Damage' },

    // === 穿透/减抗 ===
    { zh: '伤害穿透 #% 火焰抗性', en: 'Damage Penetrates #% Fire Resistance' },
    { zh: '伤害穿透 #% 冰霜抗性', en: 'Damage Penetrates #% Cold Resistance' },
    { zh: '伤害穿透 #% 闪电抗性', en: 'Damage Penetrates #% Lightning Resistance' },
    { zh: '伤害穿透 #% 元素抗性', en: 'Damage Penetrates #% Elemental Resistances' },

    // === 移速 ===
    { zh: '移动速度加快 #%', en: '#% increased Movement Speed' },

    // === 宝石等级 ===
    { zh: '所有火焰技能石等级 +#', en: '+# to Level of all Fire Skill Gems' },
    { zh: '所有冰霜技能石等级 +#', en: '+# to Level of all Cold Skill Gems' },
    { zh: '所有闪电技能石等级 +#', en: '+# to Level of all Lightning Skill Gems' },
    { zh: '所有混沌技能石等级 +#', en: '+# to Level of all Chaos Skill Gems' },
    { zh: '所有物理技能石等级 +#', en: '+# to Level of all Physical Skill Gems' },
    { zh: '所有技能石等级 +#', en: '+# to Level of all Skill Gems' },
    { zh: '所有法术技能石等级 +#', en: '+# to Level of all Spell Skill Gems' },
    { zh: '所有召唤生物技能石等级 +#', en: '+# to Level of all Minion Skill Gems' },
    { zh: '装备中的技能石等级 +#', en: '+# to Level of Socketed Gems' },
    { zh: '装备中的辅助技能石等级 +#', en: '+# to Level of Socketed Support Gems' },
    { zh: '插槽中的技能石等级 +#', en: '+# to Level of Socketed Gems' },

    // === 命中/精准 ===
    { zh: '+# 命中值', en: '+# to Accuracy Rating' },
    { zh: '命中值提高 #%', en: '#% increased Accuracy Rating' },
    { zh: '全域命中值提高 #%', en: '#% increased Global Accuracy Rating' },

    // === 法术压制 ===
    { zh: '+#% 压制法术伤害几率', en: '+#% chance to Suppress Spell Damage' },

    // === 偷取 ===
    { zh: '攻击伤害的 #% 转化为生命偷取', en: '#% of Attack Damage Leeched as Life' },
    { zh: '攻击伤害的 #% 转化为魔力偷取', en: '#% of Attack Damage Leeched as Mana' },
    { zh: '物理攻击伤害的 #% 转化为生命偷取', en: '#% of Physical Attack Damage Leeched as Life' },
    { zh: '物理攻击伤害的 #% 转化为魔力偷取', en: '#% of Physical Attack Damage Leeched as Mana' },

    // === 武器属性 ===
    { zh: '攻击速度加快 #%（局域）', en: '#% increased Attack Speed' },
    { zh: '物理伤害提高 #%（局域）', en: '#% increased Physical Damage' },
    { zh: '暴击率提高 #%（局域）', en: '#% increased Critical Strike Chance' },
    { zh: '局域物理伤害提高 #%', en: '#% increased Physical Damage' },

    // === 其他常见 ===
    { zh: '效果区域扩大 #%', en: '#% increased Area of Effect' },
    { zh: '光环效果提高 #%', en: '#% increased effect of Non-Curse Auras from your Skills' },
    { zh: '技能效果持续时间延长 #%', en: '#% increased Skill Effect Duration' },
    { zh: '药剂效果提高 #%', en: '#% increased Flask Effect Duration' },
    { zh: '冷却回复速度加快 #%', en: '#% increased Cooldown Recovery Rate' },
    { zh: '减少 #% 魔力保留效能', en: '#% reduced Mana Reservation Efficiency of Skills' },
    { zh: '魔力保留效能提高 #%', en: '#% increased Mana Reservation Efficiency of Skills' },

    // === 结界 ===
    { zh: '+# 结界', en: '+# to Ward' },
  ];

  // =========================================================================
  // 翻译引擎
  // =========================================================================

  /**
   * 将中文 mod 行翻译为英文
   * 采用模板匹配: 先把数字替换为 #，然后在映射表查找
   */
  function translateModLine(zhLine) {
    const trimmed = zhLine.trim();
    if (!trimmed) return '';

    // 提取所有数字 (包括小数和负数)
    const numbers = [];
    // 把数字替换为 # 来匹配模板
    const template = trimmed.replace(/[+-]?\d+\.?\d*/g, (match) => {
      numbers.push(match);
      return '#';
    });

    // 在映射表中查找
    for (const entry of MOD_TRANSLATIONS) {
      if (entry.zh === template) {
        // 找到匹配，回填数字
        let idx = 0;
        const result = entry.en.replace(/#/g, () => {
          return numbers[idx++] || '#';
        });
        return result;
      }
    }

    // 没找到匹配 — 尝试模糊匹配 (去掉前后的 +/- 符号再试)
    const templateNoSign = template.replace(/^[+-]\s*/, '');
    for (const entry of MOD_TRANSLATIONS) {
      const entryNoSign = entry.zh.replace(/^[+-]\s*/, '');
      if (entryNoSign === templateNoSign) {
        let idx = 0;
        const result = entry.en.replace(/#/g, () => {
          return numbers[idx++] || '#';
        });
        return result;
      }
    }

    // 翻译失败，返回原文并标记
    console.warn(`[POB DPS Calc] 未翻译的 mod: "${trimmed}" (template: "${template}")`);
    return trimmed; // 返回中文原文，POB会尝试解析
  }

  /**
   * 翻译物品基底名
   */
  function translateBaseType(zhBase) {
    return BASE_TYPE_ZH_TO_EN[zhBase] || zhBase;
  }

  /**
   * 将中文市集上的物品信息转换为 POB item_text 格式
   */
  function buildPobItemText(itemInfo) {
    const lines = [];

    // Rarity
    const rarity = RARITY_ZH_TO_EN[itemInfo.rarity] || 'RARE';
    lines.push(`Rarity: ${rarity}`);

    // Name (传奇物品有特殊名称)
    if (itemInfo.name && rarity === 'UNIQUE') {
      lines.push(itemInfo.name);
    } else if (itemInfo.name) {
      lines.push(itemInfo.name);
    }

    // Base type
    lines.push(translateBaseType(itemInfo.baseType));

    // Quality
    if (itemInfo.quality) {
      lines.push(`Quality: +${itemInfo.quality}%`);
    }

    // Item Level
    if (itemInfo.itemLevel) {
      lines.push(`Item Level: ${itemInfo.itemLevel}`);
    }

    // Sockets
    if (itemInfo.sockets) {
      lines.push(`Sockets: ${itemInfo.sockets}`);
    }

    // Implicits
    if (itemInfo.implicits && itemInfo.implicits.length > 0) {
      lines.push(`Implicits: ${itemInfo.implicits.length}`);
      for (const mod of itemInfo.implicits) {
        const translated = translateModLine(mod);
        lines.push(`{implicit}${translated}`);
      }
    } else {
      lines.push('Implicits: 0');
    }

    // Explicit mods
    if (itemInfo.explicits && itemInfo.explicits.length > 0) {
      for (const mod of itemInfo.explicits) {
        const translated = translateModLine(mod);
        lines.push(translated);
      }
    }

    // Crafted mods
    if (itemInfo.crafted && itemInfo.crafted.length > 0) {
      for (const mod of itemInfo.crafted) {
        const translated = translateModLine(mod);
        lines.push(`{crafted}${translated}`);
      }
    }

    // Enchantments
    if (itemInfo.enchants && itemInfo.enchants.length > 0) {
      for (const mod of itemInfo.enchants) {
        const translated = translateModLine(mod);
        lines.push(`{enchant}${translated}`);
      }
    }

    // Fractured
    if (itemInfo.fractured && itemInfo.fractured.length > 0) {
      for (const mod of itemInfo.fractured) {
        const translated = translateModLine(mod);
        lines.push(`{fractured}${translated}`);
      }
    }

    return lines.join('\n');
  }

  /**
   * 根据物品类别确定装备槽位
   */
  function detectSlot(itemInfo) {
    const config = getConfig();
    if (config.defaultSlot !== 'auto') {
      return config.defaultSlot;
    }
    const enClass = ITEM_CLASS_ZH_TO_EN[itemInfo.itemClass] || '';
    return CLASS_TO_SLOT[enClass] || 'Helmet'; // fallback
  }

  // =========================================================================
  // 中文市集页面 DOM 解析
  // =========================================================================

  /**
   * 从商品搜索结果的单个物品 DOM 节点中提取信息
   * 兼容 poe.game.qq.com/trade 的页面结构
   */
  function extractItemInfo(itemEl) {
    const info = {
      rarity: '稀有',
      name: '',
      baseType: '',
      itemClass: '',
      itemLevel: 0,
      quality: 0,
      sockets: '',
      implicits: [],
      explicits: [],
      crafted: [],
      enchants: [],
      fractured: [],
    };

    try {
      // --- 尝试从 data-* 属性或结构化数据获取 ---
      // 中文市集通常使用 React/Vue 渲染，DOM 结构如下参考:
      // .resultset .row 每行一个物品
      // 物品头部: 物品名称、基底类型、稀有度
      // 物品属性区: 各种 mod

      // 检测物品名称和基底
      const headerEl = itemEl.querySelector('.itemHeader, [class*="itemHeader"], [class*="item-header"]');
      if (headerEl) {
        // 稀有度来自 CSS class 或 frame type
        const frameType = headerEl.getAttribute('data-frame-type') ||
          headerEl.className || '';
        if (frameType.includes('unique') || frameType.includes('3')) {
          info.rarity = '传奇';
        } else if (frameType.includes('rare') || frameType.includes('2')) {
          info.rarity = '稀有';
        } else if (frameType.includes('magic') || frameType.includes('1')) {
          info.rarity = '魔法';
        } else if (frameType.includes('normal') || frameType.includes('0')) {
          info.rarity = '普通';
        }

        // 名称和基底
        const nameSpans = headerEl.querySelectorAll('span.lc, .typeLine, [class*="name"], [class*="typeLine"]');
        if (nameSpans.length >= 2) {
          info.name = nameSpans[0].textContent.trim();
          info.baseType = nameSpans[nameSpans.length - 1].textContent.trim();
        } else if (nameSpans.length === 1) {
          info.baseType = nameSpans[0].textContent.trim();
        }
      }

      // --- 解析 mods ---
      const modGroups = itemEl.querySelectorAll('.property, .mod, [class*="mod"], [class*="affix"]');
      modGroups.forEach(modEl => {
        const text = modEl.textContent.trim();
        if (!text) return;

        const classes = modEl.className || '';
        if (classes.includes('implicit')) {
          info.implicits.push(text);
        } else if (classes.includes('crafted')) {
          info.crafted.push(text);
        } else if (classes.includes('enchant')) {
          info.enchants.push(text);
        } else if (classes.includes('fractured')) {
          info.fractured.push(text);
        } else {
          info.explicits.push(text);
        }
      });

      // --- 从 API 数据获取 (如果能拿到) ---
      // 中文市集的物品数据也可能存储在 __NEXT_DATA__ 或 window._searchResults 中

    } catch (e) {
      console.error('[POB DPS Calc] 解析物品信息失败:', e);
    }

    return info;
  }

  /**
   * 尝试从市集 API 响应中提取物品的 raw text
   * 中文市集通常会在 fetch 响应里包含物品文本
   */
  function extractItemFromApiData(itemData) {
    const info = {
      rarity: '稀有',
      name: '',
      baseType: '',
      itemClass: '',
      itemLevel: 0,
      quality: 0,
      sockets: '',
      implicits: [],
      explicits: [],
      crafted: [],
      enchants: [],
      fractured: [],
    };

    if (!itemData) return info;

    // 物品名和基底
    info.name = itemData.name || '';
    info.baseType = itemData.typeLine || itemData.baseType || '';

    // 稀有度
    const frameType = itemData.frameType;
    if (frameType === 3) info.rarity = '传奇';
    else if (frameType === 2) info.rarity = '稀有';
    else if (frameType === 1) info.rarity = '魔法';
    else if (frameType === 0) info.rarity = '普通';

    // 物品等级
    info.itemLevel = itemData.ilvl || 0;

    // 品质
    if (itemData.properties) {
      for (const prop of itemData.properties) {
        if (prop.name === '品质' || prop.name === 'Quality') {
          const val = prop.values && prop.values[0] && prop.values[0][0];
          if (val) info.quality = parseInt(String(val).replace(/[+%]/g, ''), 10) || 0;
        }
      }
    }

    // 插槽
    if (itemData.sockets && itemData.sockets.length > 0) {
      const groups = {};
      for (const s of itemData.sockets) {
        if (!groups[s.group]) groups[s.group] = [];
        groups[s.group].push(s.sColour || s.colour || 'W');
      }
      info.sockets = Object.values(groups).map(g => g.join('-')).join(' ');
    }

    // 隐性词缀
    if (itemData.implicitMods) {
      info.implicits = itemData.implicitMods.slice();
    }

    // 显性词缀
    if (itemData.explicitMods) {
      info.explicits = itemData.explicitMods.slice();
    }

    // 工艺词缀
    if (itemData.craftedMods) {
      info.crafted = itemData.craftedMods.slice();
    }

    // 附魔
    if (itemData.enchantMods) {
      info.enchants = itemData.enchantMods.slice();
    }

    // 裂隙词缀
    if (itemData.fracturedMods) {
      info.fractured = itemData.fracturedMods.slice();
    }

    // 物品类别
    if (itemData.extended && itemData.extended.category) {
      info.itemClass = itemData.extended.category;
    }

    return info;
  }

  // =========================================================================
  // API 调用
  // =========================================================================

  function callReplaceItemAPI(pobCode, slot, itemText) {
    return new Promise((resolve, reject) => {
      const config = getConfig();
      GM_xmlhttpRequest({
        method: 'POST',
        url: `${config.apiUrl}/replace-item`,
        headers: {
          'Content-Type': 'application/json',
        },
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
        onerror: function (err) {
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

    // 捕获 /api/trade/fetch 的响应 (包含物品详情)
    if (url.includes('/api/trade/fetch')) {
      try {
        const cloned = response.clone();
        const data = await cloned.json();
        if (data && data.result) {
          for (const item of data.result) {
            if (item && item.id && item.item) {
              itemDataCache[item.id] = item.item;
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
    if (this._pobUrl && this._pobUrl.includes('/api/trade/fetch')) {
      this.addEventListener('load', function () {
        try {
          const data = JSON.parse(this.responseText);
          if (data && data.result) {
            for (const item of data.result) {
              if (item && item.id && item.item) {
                itemDataCache[item.id] = item.item;
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
    // 中文市集的 DOM 结构: 每个搜索结果是一个 row，包含物品信息
    const itemRows = document.querySelectorAll(
      '.resultset .row, [class*="search-result"], [class*="result-row"], [data-id]'
    );

    if (itemRows.length === 0) {
      // 尝试其他选择器
      const altRows = document.querySelectorAll('[class*="row"]:not(.pob-processed)');
      // 如果DOM结构不确定，以MutationObserver持续观察
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

        // 获取物品数据
        let itemInfo;
        if (itemId && itemDataCache[itemId]) {
          itemInfo = extractItemFromApiData(itemDataCache[itemId]);
        } else {
          itemInfo = extractItemInfo(row);
        }

        if (!itemInfo.baseType && !itemInfo.name) {
          renderErrorPanel('无法识别该物品的信息', row);
          return;
        }

        // 构建 POB item text
        const itemText = buildPobItemText(itemInfo);
        const slot = detectSlot(itemInfo);

        console.log('[POB DPS Calc] 物品信息:', itemInfo);
        console.log('[POB DPS Calc] POB item text:\n' + itemText);
        console.log('[POB DPS Calc] Slot:', slot);

        // 显示加载状态
        btn.classList.add('loading');
        btn.innerHTML = '计算中';
        btn.disabled = true;

        try {
          const result = await callReplaceItemAPI(config.pobCode, slot, itemText);
          renderResultPanel(result, row);
        } catch (err) {
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
        // 如果找不到特定位置，就插到行首/行尾
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
            // 检查是否是新的搜索结果
            if (node.matches && (
              node.matches('[data-id]') ||
              node.matches('.row') ||
              node.matches('[class*="result"]')
            )) {
              shouldInject = true;
            }
            // 或者子节点中包含结果
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
    // 移除旧面板
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
      <div class="pob-hint">POB Recalc API 服务的地址</div>

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
      <div class="pob-hint">"自动检测" 会根据物品类型推断槽位</div>

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

      // 临时用自定义 URL 测试
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

  console.log('[POB DPS Calc] 油猴脚本已加载');

  // 创建浮动入口
  createFloatButton();

  // 初次尝试注入按钮
  setTimeout(() => injectCalcButtons(), 1000);

  // 定期检查连接状态
  setInterval(() => updateFloatButton(), 30000);

})();
