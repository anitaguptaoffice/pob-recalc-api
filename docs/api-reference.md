# POB Recalc API — 完整接口文档

> **Path of Building (POB)** 的无头计算引擎 HTTP API。通过 LuaJIT worker 进程池运行 POB 核心计算逻辑，提供装备替换对比、属性权重生成、最优涂油选择等功能。

---

## 目录

- [架构概览](#架构概览)
- [通用说明](#通用说明)
- [API 端点](#api-端点)
  - [1. POST /recalc — 重算 Build](#1-post-recalc--重算-build)
  - [2. POST /replace-item — 装备替换对比](#2-post-replace-item--装备替换对比)
  - [3. POST /generate-weights — 属性权重生成](#3-post-generate-weights--属性权重生成)
  - [4. POST /find-best-anoint — 最优涂油选择](#4-post-find-best-anoint--最优涂油选择)
  - [5. POST /build-cost — Build 造价估算](#5-post-build-cost--build-造价估算)
  - [6. GET /health — 健康检查](#6-get-health--健康检查)
  - [7. GET /version — 版本信息](#7-get-version--版本信息)
- [功能详解](#功能详解)
  - [全局自定义修正 (Custom Modifiers)](#全局自定义修正-custom-modifiers)
  - [物品词缀标记](#物品词缀标记)
- [常用 Stat 名称参考](#常用-stat-名称参考)
- [装备槽位名称](#装备槽位名称)
- [测试脚本](#测试脚本)

---

## 架构概览

```
┌──────────────┐     HTTP      ┌──────────────┐     stdin/stdout     ┌──────────────┐
│   客户端      │ ──────────── │  Go Server   │ ─────────────────── │ LuaJIT Worker │
│  (curl/MCP)  │   JSON/XML   │  (main.go)   │    自定义协议        │ (worker.lua)  │
└──────────────┘              └──────────────┘                      └──────────────┘
                                    │                                      │
                                    │  Worker Pool (默认2)                  │  POB 核心引擎
                                    │  Acquire → 计算 → Release            │  HeadlessWrapper
                                    │                                      │  CalcsTab / ItemsTab
```

- **Go Server** (`main.go`)：HTTP 入口，管理 LuaJIT worker 进程池，负责 POB code 解码
- **LuaJIT Worker** (`worker.lua`)：加载 POB 核心引擎，通过 stdin/stdout 自定义协议通信
- **协议格式**：`COMMAND <len1> [<len2> ...]\n` + 二进制 payload，响应 `OK <len>\n` / `ERR <len>\n` + body

---

## 通用说明

### 基础 URL

```
http://localhost:8080
```

### POB Code

所有需要 build 数据的接口都通过 `pob_code` 参数传入。POB Code 是 Path of Building 导出的分享码，本质上是 **base64url + zlib 压缩** 的 XML 文本。

### 响应头

所有计算接口都会返回以下性能计时头：

| Header | 说明 |
|--------|------|
| `X-Decode-Time-Ms` | POB code 解码耗时（毫秒） |
| `X-Calc-Time-Ms` / `X-Recalc-Time-Ms` | 核心计算耗时（毫秒） |
| `X-Total-Time-Ms` | 总耗时（毫秒） |

### 错误处理

- `400 Bad Request` — 参数缺失或 POB code 无法解码
- `405 Method Not Allowed` — 使用了错误的 HTTP 方法
- `500 Internal Server Error` — Worker 计算失败
- `503 Service Unavailable` — 无可用 worker（全忙或超时）

---

## API 端点

### 1. POST /recalc — 重算 Build

加载 POB code 对应的 build，执行完整重算，返回更新后的 XML。

**用途**：当你修改了 XML 内容（如注入 customMods、修改配置）后，需要重新计算面板数据。

#### 请求

```
POST /recalc
Content-Type: text/plain

<POB code 字符串>
```

> 注意：这个接口的请求体直接是 POB code 纯文本，不是 JSON。

#### 响应

```
HTTP 200
Content-Type: application/xml; charset=utf-8

<?xml version="1.0" encoding="UTF-8"?>
<PathOfBuilding>
  ... 完整的重算后 XML ...
</PathOfBuilding>
```

#### curl 示例

```bash
curl -X POST http://localhost:8080/recalc \
  -H "Content-Type: text/plain" \
  -d "eNrtPdl2..." \
  -o recalculated.xml
```

---

### 2. POST /replace-item — 装备替换对比

替换 build 中指定槽位的装备，对比替换前后的全部属性变化。

**用途**：评估某件装备对 build 的影响——它会提升多少 DPS？降低多少生存？

#### 请求

```json
POST /replace-item
Content-Type: application/json

{
  "pob_code": "eNrtPdl2...",
  "slot": "Helmet",
  "item_text": "Rarity: RARE\nApocalypse Crown\nProphet Crown\n..."
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `pob_code` | string | ✅ | POB 分享码 |
| `slot` | string | ✅ | 装备槽位名（见[装备槽位名称](#装备槽位名称)） |
| `item_text` | string | ✅ | POB 格式的装备文本（支持 `{custom}` 等标记） |

#### 响应

```json
{
  "slot": "Helmet",
  "old_item_id": 3,
  "new_item_id": 999,
  "before": {
    "TotalDPS": 1234567.89,
    "CombinedDPS": 2345678.90,
    "Life": 4500,
    "EnergyShield": 2800,
    "FireResist": 75,
    ...
  },
  "after": {
    "TotalDPS": 1300000.00,
    "CombinedDPS": 2400000.00,
    "Life": 4580,
    "EnergyShield": 3050,
    "FireResist": 75,
    ...
  },
  "diff": {
    "TotalDPS": 65432.11,
    "CombinedDPS": 54321.10,
    "Life": 80,
    "EnergyShield": 250,
    ...
  }
}
```

**返回字段说明**：

| 字段 | 说明 |
|------|------|
| `before` | 替换前的所有属性值 |
| `after` | 替换后的所有属性值 |
| `diff` | 差值 (`after - before`)，正值表示提升，负值表示下降 |

包含 25+ 属性指标，详见 [常用 Stat 名称参考](#常用-stat-名称参考)。

#### curl 示例

```bash
curl -X POST http://localhost:8080/replace-item \
  -H "Content-Type: application/json" \
  -d '{
    "pob_code": "'"$(cat pob)"'",
    "slot": "Helmet",
    "item_text": "Rarity: RARE\nTest Crown\nProphet Crown\nImplicits: 0\n+80 to maximum Life\n+40% to Fire Resistance"
  }'
```

#### 测试脚本

```
tests/test_replace.py      — 基础装备替换测试
tests/test_custom_mod.py   — {custom} 自定义词缀测试
```

---

### 3. POST /generate-weights — 属性权重生成

对指定装备槽位，遍历该槽位所有可能的词条（mod），计算每个词条对 build 的提升权重。

**用途**：生成 PoE Trade 搜索权重——帮你知道该优先搜哪些词条，以及每条词缀值多少"DPS 等价"。

#### 请求

```json
POST /generate-weights
Content-Type: application/json

{
  "pob_code": "eNrtPdl2...",
  "slot": "Helmet",
  "stat_weights": [
    {"stat": "FullDPS", "weightMult": 1.0},
    {"stat": "TotalEHP", "weightMult": 0.5}
  ],
  "include_corrupted": false,
  "include_eldritch": false,
  "include_scourge": false,
  "include_synthesis": false,
  "include_talisman": false
}
```

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `pob_code` | string | ✅ | — | POB 分享码 |
| `slot` | string | ✅ | — | 装备槽位名 |
| `stat_weights` | array | ❌ | `[{FullDPS:1.0}, {TotalEHP:0.5}]` | 关注的属性及其权重倍率 |
| `stat_weights[].stat` | string | ✅ | — | 属性名（如 `FullDPS`, `TotalEHP`, `Life`） |
| `stat_weights[].weightMult` | number | ✅ | — | 权重乘数，越大越重视 |
| `include_corrupted` | bool | ❌ | `false` | 是否包含腐化词缀 |
| `include_eldritch` | bool | ❌ | `false` | 是否包含异界尊师词缀 |
| `include_scourge` | bool | ❌ | `false` | 是否包含灾祸词缀 |
| `include_synthesis` | bool | ❌ | `false` | 是否包含综合词缀 |
| `include_talisman` | bool | ❌ | `false` | 是否包含护身符特殊词缀 |

#### 响应

```json
{
  "slot": "Helmet",
  "item_category": "Helmet",
  "current_item": "Crown of the Inward Eye",
  "current_stat_diff": 1.23,
  "mods_tested": 342,
  "mod_weights": [
    {
      "trade_mod_id": "pseudo.pseudo_total_life",
      "weight": 15.82,
      "mean_stat_diff": 0.045,
      "mod_text": "+# to maximum Life",
      "mod_type": "explicit",
      "test_value": 50,
      "invert": false
    },
    {
      "trade_mod_id": "explicit.stat_2974417149",
      "weight": 12.50,
      "mean_stat_diff": 0.038,
      "mod_text": "#% increased Spell Damage",
      "mod_type": "explicit",
      "test_value": 30,
      "invert": false
    }
    // ... 更多词条
  ]
}
```

**返回字段说明**：

| 字段 | 说明 |
|------|------|
| `item_category` | 物品类别（Helmet, Body Armour, Weapon 等） |
| `current_item` | 当前装备名称 |
| `current_stat_diff` | 当前装备相对空槽的加权提升 |
| `mods_tested` | 测试的词条总数 |
| `mod_weights` | 按影响力降序排列的词条权重列表 |
| `mod_weights[].trade_mod_id` | 可直接用于 pathofexile.com Trade API 的 mod ID |
| `mod_weights[].weight` | 权重值（越大越重要） |
| `mod_weights[].mean_stat_diff` | 平均属性提升量 |
| `mod_weights[].mod_text` | 词条文本模板（`#` 代表数值） |
| `mod_weights[].test_value` | 测试用的数值 |
| `mod_weights[].invert` | 是否取反（用于 Trade API 的 invert 参数） |

> ⏱ 计算耗时约 4-6 秒/槽位（需测试数百个 mod）。

#### curl 示例

```bash
curl -X POST http://localhost:8080/generate-weights \
  -H "Content-Type: application/json" \
  -d '{
    "pob_code": "'"$(cat pob)"'",
    "slot": "Helmet",
    "stat_weights": [
      {"stat": "FullDPS", "weightMult": 1.0},
      {"stat": "TotalEHP", "weightMult": 0.5}
    ]
  }'
```

#### 测试脚本

```
tests/test_generate_weights.py — 多槽位 + 多权重组合测试
```

---

### 4. POST /find-best-anoint — 最优涂油选择

遍历天赋树中所有可涂油的天赋节点（约 470 个），计算每个涂油对 build 指定属性的提升，排序返回最优选择。

**用途**：护身符涂油选哪个最好？该涂 DPS 还是生存？这个接口帮你一键算出来。

#### 请求

```json
POST /find-best-anoint
Content-Type: application/json

{
  "pob_code": "eNrtPdl2...",
  "stat": "CombinedDPS",
  "max_results": 20,
  "search": "",
  "slot_name": "Amulet"
}
```

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `pob_code` | string | ✅ | — | POB 分享码 |
| `stat` | string | ❌ | `CombinedDPS` | 排序依据的属性名 |
| `max_results` | int | ❌ | `30` | 返回前 N 个结果 |
| `search` | string | ❌ | `""` | 按节点名/描述文本过滤（不区分大小写） |
| `slot_name` | string | ❌ | `Amulet` | 涂油目标槽位 |

#### 响应

```json
{
  "stat": "CombinedDPS",
  "slot": "Amulet",
  "current_item": "Whispers of Infinity, Seaglass Amulet",
  "nodes_tested": 470,
  "nodes_skipped": 0,
  "total_anointable": 470,
  "current_anoints": ["Forethought"],
  "results": [
    {
      "node_id": 5574,
      "name": "Force of Darkness",
      "description": "Damage Penetrates 15% Chaos Resistance",
      "diff": 2102280,
      "new_value": 14155300,
      "base_value": 12053100,
      "is_allocated": false,
      "is_current_anoint": false,
      "oil_recipe": [7, 6, 6],
      "is_keystone": false,
      "is_notable": true
    },
    {
      "node_id": 1234,
      "name": "Corruption",
      "description": "30% increased Chaos Damage\n...",
      "diff": 1850000,
      "new_value": 13903100,
      "base_value": 12053100,
      "is_allocated": true,
      "is_current_anoint": false,
      "oil_recipe": [3, 5, 6],
      "is_keystone": false,
      "is_notable": true
    }
    // ... 更多结果
  ]
}
```

**返回字段说明**：

| 字段 | 说明 |
|------|------|
| `nodes_tested` | 实际测试的节点数 |
| `nodes_skipped` | 被 search 过滤跳过的节点数 |
| `current_anoints` | 当前已涂油的天赋节点名 |
| `results[].diff` | 属性提升量（`new_value - base_value`） |
| `results[].is_allocated` | 该节点是否已在天赋树上分配（涂油重复） |
| `results[].is_current_anoint` | 该节点是否是当前的涂油 |
| `results[].oil_recipe` | 涂油配方（油的 index 数组） |
| `results[].is_keystone` | 是否为基石天赋 |
| `results[].is_notable` | 是否为重要天赋 |

> ⏱ 计算耗时约 7-8 秒（全量 470 节点），使用 `search` 过滤可降至 <1 秒。

#### curl 示例

```bash
# 按 CombinedDPS 排序，取前 10
curl -X POST http://localhost:8080/find-best-anoint \
  -H "Content-Type: application/json" \
  -d '{
    "pob_code": "'"$(cat pob)"'",
    "stat": "CombinedDPS",
    "max_results": 10
  }'

# 搜索含 "chaos" 的节点（更快）
curl -X POST http://localhost:8080/find-best-anoint \
  -H "Content-Type: application/json" \
  -d '{
    "pob_code": "'"$(cat pob)"'",
    "stat": "CombinedDPS",
    "search": "chaos",
    "max_results": 5
  }'

# 按 EnergyShield 排序
curl -X POST http://localhost:8080/find-best-anoint \
  -H "Content-Type: application/json" \
  -d '{
    "pob_code": "'"$(cat pob)"'",
    "stat": "EnergyShield",
    "max_results": 10
  }'
```

#### 测试脚本

```
tests/test_find_best_anoint.py — 多排序指标 + 搜索过滤测试
```

---

### 5. POST /build-cost — Build 造价估算

解析 POB code 中的装备和宝石，通过国服 Trade API 查询价格，输出总造价。

**用途**：快速估算一个 build 的总成本。

#### 请求

```json
POST /build-cost
Content-Type: application/json

{
  "pob_code": "eNrtPdl2...",
  "poesessid": "你的国服POESESSID",
  "cn_league": "S29赛季"
}
```

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `pob_code` | string | ✅ | — | POB 分享码 |
| `poesessid` | string | ✅ | — | 国服 POESESSID cookie（Trade API 认证） |
| `cn_league` | string | ❌ | `S29赛季` | 国服联赛名 |

#### 响应

```json
{
  "divine_rate": 150,
  "total_chaos": 5000,
  "total_divine": 33.3,
  "items": [
    {
      "name": "Headhunter",
      "name_zh": "猎首",
      "base_type": "Leather Belt",
      "slot": "Belt",
      "rarity": "UNIQUE",
      "price_chaos": 500,
      "confidence": "high"
    }
  ],
  "gems": [
    {
      "name": "Cyclone",
      "name_zh": "旋风斩",
      "gem_level": 21,
      "gem_quality": 20,
      "price_chaos": 30,
      "confidence": "medium"
    }
  ]
}
```

> ⏱ 耗时较长（每件物品约 6 秒查询间隔，受国服 Trade API 限流约束）。

#### curl 示例

```bash
curl -X POST http://localhost:8080/build-cost \
  -H 'Content-Type: application/json' \
  -d '{
    "pob_code": "eNrtPdl2...",
    "poesessid": "你的POESESSID",
    "cn_league": "S29赛季"
  }'
```

---

### 6. GET /health — 健康检查

```
GET /health
```

响应：

```
ok
workers_available: 2
```

---

### 7. GET /version — 版本信息

```
GET /version
```

响应：POB 源码版本信息（如果 `pob_version` 文件存在）。

---

## 功能详解

### 全局自定义修正 (Custom Modifiers)

POB 支持通过 Configuration 页面的 "Custom Modifiers" 文本框注入全局词缀。在 API 层面，这些修正嵌入在 POB code 的 XML 中。

**详细用法请参考**：[pob-custom-modifiers-guide.md](./pob-custom-modifiers-guide.md)

**快速入门**：

```python
import base64, zlib, re

def inject_custom_mods(pob_code: str, mods: list[str]) -> str:
    """向 POB code 注入全局自定义修正"""
    # 解码
    b64std = pob_code.strip().replace('-', '+').replace('_', '/')
    decoded = base64.b64decode(b64std + '==')
    xml = zlib.decompress(decoded).decode('utf-8')
    
    # 注入（用 &#10; 作为换行分隔符）
    mods_str = "&#10;".join(mods)
    custom_input = f'<Input name="customMods" string="{mods_str}"/>'
    
    if 'name="customMods"' in xml:
        xml = re.sub(r'<Input name="customMods"[^/]*/>', custom_input, xml)
    else:
        xml = xml.replace('</ConfigSet>', custom_input + '\n</ConfigSet>', 1)
    
    # 重新编码
    compressed = zlib.compress(xml.encode('utf-8'))
    return base64.b64encode(compressed).decode('ascii').replace('+', '-').replace('/', '_').rstrip('=')
```

注入后的 POB code 可用于所有 API 接口。

**测试脚本**：

```
tests/test_debug_custommods.py   — customMods XML 注入调试
tests/test_global_custom_mods.py — 全局 customMods 对比验证
```

---

### 物品词缀标记

在 `item_text` 中，可以使用以下 `{tag}` 前缀标记特殊词缀：

| 标记 | 含义 | 示例 |
|------|------|------|
| `{custom}` | 自定义词缀（纯模拟） | `{custom}30% increased Spell Damage` |
| `{crafted}` | 工艺台词缀 | `{crafted}+25% to Cold Resistance` |
| `{fractured}` | 裂隙词缀 | `{fractured}+80 to maximum Life` |
| `{implicit}` | 隐性词缀 | `{implicit}+1 to Level of all Skill Gems` |
| `{enchant}` | 附魔 | `{enchant}40% increased Damage` |
| `{crucible}` | 坩埚词缀 | — |
| `{scourge}` | 灾祸词缀 | — |
| `{eater}` | 异界吞噬者词缀 | — |
| `{exarch}` | 灼炎尊师词缀 | — |
| `{synthesis}` | 综合词缀 | — |
| `{mutated}` | 变异词缀 | — |

---

## 常用 Stat 名称参考

以下是 POB 输出中可用于 `stat` 参数和 `stat_weights` 的属性名：

### 伤害类

| Stat 名称 | 说明 |
|-----------|------|
| `CombinedDPS` | 综合 DPS（POB 最常用的总伤指标） |
| `TotalDPS` | 总 DPS（hit） |
| `TotalDot` | 总 DoT DPS |
| `FullDPS` | 完整 DPS（包含所有伤害源） |
| `AverageDamage` | 平均单次伤害 |
| `Speed` | 攻击/施法速度 |
| `CritChance` | 暴击率 |
| `CritMultiplier` | 暴击伤害加成 |
| `ActiveMinionLimit` | 召唤物上限 |

### 防御/生存类

| Stat 名称 | 说明 |
|-----------|------|
| `Life` | 生命值 |
| `EnergyShield` | 能量护盾 |
| `Mana` | 法力值 |
| `TotalEHP` | 总有效生命（综合防御评估） |
| `Armour` | 护甲 |
| `Evasion` | 闪避 |
| `Ward` | 结界 |
| `BlockChance` | 格挡率 |
| `SpellBlockChance` | 法术格挡率 |
| `SuppressionChance` | 法术压制率 |

### 抗性类

| Stat 名称 | 说明 |
|-----------|------|
| `FireResist` | 火抗 |
| `ColdResist` | 冰抗 |
| `LightningResist` | 电抗 |
| `ChaosResist` | 混沌抗 |

### 召唤物类

| Stat 名称 | 说明 |
|-----------|------|
| `MinionTotalDPS` | 召唤物总 DPS |
| `MinionCombinedDPS` | 召唤物综合 DPS |
| `MinionLife` | 召唤物生命 |

> 提示：Minion 类属性会自动从 `output.Minion` 子表中读取。

---

## 装备槽位名称

以下是 `slot` 参数可用的值：

| 槽位名 | 说明 |
|--------|------|
| `Helmet` | 头盔 |
| `Body Armour` | 胸甲 |
| `Gloves` | 手套 |
| `Boots` | 鞋子 |
| `Belt` | 腰带 |
| `Amulet` | 护身符 |
| `Ring 1` | 戒指 1 |
| `Ring 2` | 戒指 2 |
| `Weapon 1` | 主手武器 |
| `Weapon 2` | 副手武器/盾牌 |
| `Flask 1` ~ `Flask 5` | 药剂 1-5 |

---

## 测试脚本

所有测试脚本位于 `tests/` 目录：

| 脚本 | 对应功能 | 说明 |
|------|---------|------|
| `test_replace.py` | `/replace-item` | 基础装备替换 + DPS 对比 |
| `test_custom_mod.py` | `/replace-item` + `{custom}` | 物品级自定义词缀 (`{custom}`) 效果验证 |
| `test_debug_custommods.py` | 全局 customMods | 调试 XML 注入 `&#10;` vs `\n` 换行分隔 |
| `test_global_custom_mods.py` | 全局 customMods | 全局自定义修正对 before/after 影响对比 |
| `test_generate_weights.py` | `/generate-weights` | 多槽位 (Helmet/Ring/Weapon) 权重生成 |
| `test_find_best_anoint.py` | `/find-best-anoint` | 涂油排序 + 搜索过滤 + 多 stat 测试 |

运行示例：

```bash
cd pob-recalc-api
python3 tests/test_replace.py
python3 tests/test_generate_weights.py
python3 tests/test_find_best_anoint.py
```

> 所有测试脚本需要：(1) 本地 `pob` 文件包含有效 POB code；(2) API 服务运行在 `localhost:8080`。
