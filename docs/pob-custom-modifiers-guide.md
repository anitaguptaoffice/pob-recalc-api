# POB 全局额外修正 (Custom Modifiers) — 老手必备技能

> 这是 POB (Path of Building) 中最灵活也最容易被忽视的功能之一。掌握它，你就能模拟几乎任何 POB 无法自动识别的加成——队友光环、祭坛词缀、赛季机制、甚至尚未上线的词条。

---

## 一、什么是全局额外修正？

POB 桌面版有一个 **Configuration（配置）** 标签页，最底部有一个叫 **"Custom Modifiers"** 的自由文本框。在这里输入的每一行文字，POB 都会当作一条"全局词缀"注入到你的 build 计算中——效果等同于你的角色凭空获得了这条属性加成。

### 它能干什么？

| 场景 | 示例写法 | 为什么需要 |
|------|----------|-----------|
| 模拟队友光环 | `30% more Spell Damage` | POB 不知道你的辅助会给多少加成 |
| 模拟祭坛词缀 | `+2 to Level of all Spell Skill Gems` | 祭坛/圣坛效果无法自动导入 |
| 模拟赛季机制 | `50% increased Damage` | 新赛季机制 POB 可能还没支持 |
| 预演装备购买 | `+100 to maximum Life` | 想知道"如果我多 100 血会怎样" |
| 模拟天赋配点 | `Transfiguration of Mind` | 某些 keystone 效果可以直接写 |
| 测试伤害上限 | `100% increased Critical Strike Chance` | 快速验证某个属性的边际收益 |

---

## 二、语法规则

### 基本格式

每行一条词缀，写法和游戏内装备词缀**完全一致**：

```
+80 to maximum Life
30% increased Spell Damage
+2 to Level of all Spell Skill Gems
Adds 10 to 20 Fire Damage to Spells
+5% to Critical Strike Multiplier
```

### 颜色反馈

- **蓝色** 文字 = POB 识别成功，会纳入计算
- **红色** 文字 = POB 无法解析，不会生效（检查拼写！）

### 常见写法速查

```
# ===== 攻击/法术 =====
+100 to maximum Life
+50 to maximum Energy Shield
+30 to maximum Mana
30% increased Spell Damage
20% more Spell Damage                    # more 和 increased 是两回事！
Adds 10 to 20 Cold Damage to Spells
+5% to Critical Strike Multiplier
10% increased Cast Speed
10% increased Attack Speed

# ===== 防御 =====
+40% to Fire Resistance
+35% to Cold Resistance
+30% to Lightning Resistance
+20% to Chaos Resistance
+1000 to Armour
+500 to Evasion Rating
10% additional Physical Damage Reduction

# ===== 高级/特殊 =====
+2 to Level of all Spell Skill Gems
+1 to Maximum Power Charges
Transfiguration of Mind                   # Keystone 名直接写
Enemies have -10% to Total Physical Damage Reduction
Nearby Enemies have -9% to Fire Resistance
```

### 注意事项

1. **`more` vs `increased`**：这是 POE 的核心机制区别。`more` 是乘算，`increased` 是加算。写错了结果天差地别
2. **大小写敏感**：`Maximum Life` ✅ vs `maximum life` — POB 通常不区分大小写，但保持首字母大写是好习惯
3. **精确拼写**：必须和游戏内词缀拼写一致，否则 POB 不认。不确定时在 POB 里试——蓝色就是对的

---

## 三、在 POB 桌面版中使用

1. 打开 POB，导入你的 build
2. 点击顶部 **Configuration** 标签
3. 滚动到最底部，找到 **Custom Modifiers** 文本框
4. 直接输入词缀，每行一条
5. POB 自动重算，面板数字即时更新

---

## 四、通过 API 使用（编程方式）

在我们的 `pob-recalc-api` 中，自定义修正是**注入到 POB code（XML）里**的，不需要额外的 API 参数。

### 原理

POB code 本质上是一段 base64 + zlib 压缩的 XML。XML 中有一个 `<ConfigSet>` 节点，存储了所有 Configuration 页面的设置。我们只需要在解码后的 XML 中注入 `customMods` 字段，再重新编码即可。

### 注入方法（Python 示例）

```python
import base64, zlib

def decode_pob(pob_code: str) -> str:
    """将 POB code 解码为 XML 字符串"""
    b64std = pob_code.strip().replace('-', '+').replace('_', '/')
    # 补齐 padding
    padding = 4 - len(b64std) % 4
    if padding != 4:
        b64std += '=' * padding
    decoded = base64.b64decode(b64std)
    return zlib.decompress(decoded).decode('utf-8', errors='replace')

def encode_pob(xml_str: str) -> str:
    """将 XML 字符串编码为 POB code"""
    compressed = zlib.compress(xml_str.encode('utf-8'))
    b64 = base64.b64encode(compressed).decode('ascii')
    return b64.replace('+', '-').replace('/', '_').rstrip('=')

def inject_custom_mods(pob_code: str, mods: list[str]) -> str:
    """
    向 POB code 注入全局自定义修正词条。
    
    参数：
        pob_code: 原始 POB code
        mods: 词条列表，如 ["30% increased Spell Damage", "+100 to maximum Life"]
    
    返回：
        注入后的新 POB code
    """
    xml = decode_pob(pob_code)
    
    # 用 &#10; 作为换行分隔符（XML 实体）
    mods_str = "&#10;".join(mods)
    
    # 注入到 ConfigSet 节点中
    custom_input = f'<Input name="customMods" string="{mods_str}"/>'
    
    if 'name="customMods"' in xml:
        # 替换已有的 customMods
        import re
        xml = re.sub(
            r'<Input name="customMods"[^/]*/>', 
            custom_input, 
            xml
        )
    else:
        # 在 </ConfigSet> 前插入
        xml = xml.replace('</ConfigSet>', custom_input + '\n</ConfigSet>', 1)
    
    return encode_pob(xml)
```

### 使用示例

```python
import requests

# 原始 POB code
original_pob = "eNr..."  # 你的 POB code

# 注入自定义修正
modified_pob = inject_custom_mods(original_pob, [
    "10% increased maximum Life",
    "+100 to all Attributes",
    "50% increased Spell Damage",
])

# 调用 /replace-item 或 /recalc 接口 —— customMods 已经嵌在 POB code 里了
resp = requests.post("http://localhost:8080/replace-item", json={
    "pob_code": modified_pob,  # 注入了修正的 POB code
    "slot": "Helmet",
    "item_text": "Rarity: RARE\nTest\nProphet Crown\nImplicits: 0\n+80 to maximum Life",
})
print(resp.json())
```

### 关键细节

| 项目 | 说明 |
|------|------|
| **换行分隔** | XML 中用 `&#10;`（XML 实体），不是 `\n` |
| **注入位置** | `<ConfigSet>` 节点内，作为 `<Input name="customMods" string="..." />` |
| **对所有接口生效** | 注入后的 POB code 可以用于 `/recalc`、`/replace-item`、`/generate-weights` 任何接口 |
| **叠加规则** | customMods 和装备词缀互相独立，各自计算后叠加 |

---

## 五、物品级别的自定义词缀 `{custom}` 标记

除了全局 Custom Modifiers，POB 还支持在**单件装备**上添加自定义词缀。方法是在词条前加 `{custom}` 前缀：

```
Rarity: RARE
Test Helmet
Prophet Crown
Implicits: 0
+80 to maximum Life
+40% to Fire Resistance
{custom}30% increased Spell Damage
{custom}+2 to Level of all Minion Skill Gems
```

### 全局 vs 物品级别对比

| 特性 | 全局 Custom Modifiers | 物品 `{custom}` 标记 |
|------|----------------------|---------------------|
| 位置 | Configuration 标签页底部 | 装备编辑器中的词条行 |
| 作用范围 | 影响整个 build | 只影响该件装备 |
| API 注入方式 | 修改 XML 中 `<ConfigSet>` | 在 `item_text` 中加 `{custom}` 前缀 |
| 适用场景 | 模拟外部增益（光环、祭坛等） | 模拟装备上的额外词条（Craft、附魔等） |
| 替换装备后 | 仍然存在 | 随装备一起替换 |

### 物品 `{custom}` 用于 API 的示例

```python
# 通过 /replace-item 接口，物品文本中直接包含 {custom} 词缀
resp = requests.post("http://localhost:8080/replace-item", json={
    "pob_code": pob_code,
    "slot": "Helmet",
    "item_text": (
        "Rarity: RARE\n"
        "My Test Helmet\n"
        "Prophet Crown\n"
        "Implicits: 0\n"
        "+80 to maximum Life\n"
        "+40% to Fire Resistance\n"
        "{custom}30% increased Spell Damage\n"         # 自定义词缀
        "{custom}+2 to Level of all Minion Skill Gems"  # 自定义词缀
    ),
})
```

---

## 六、POB 内部实现原理（供开发参考）

了解底层有助于调试和扩展。

### 全局 customMods 的计算流程

```
配置页输入 customMods 文本
        ↓
ConfigTab:BuildModList()   ← 遍历所有 ConfigOption，调用 apply 函数
        ↓
apply 函数逐行解析：
    for line in val:gmatch("([^\n]*)\n?") do
        mods = modLib.parseMod(strippedLine)    ← 核心解析器
        modList:AddMod(mod)                     ← 加入全局 mod 列表
    end
        ↓
CalcSetup.lua:
    env.modDB:AddList(build.configTab.modList)    ← 注入到计算环境
    env.enemyDB:AddList(build.configTab.enemyModList)
        ↓
CalcPerform 执行完整计算 → 输出面板数字
```

### 关键源码文件

| 文件 | 职责 |
|------|------|
| `Modules/ConfigOptions.lua` L2302 | `customMods` 的定义和 `apply` 函数 |
| `Classes/ConfigTab.lua` L865 | `BuildModList()` — 构建全局 modList |
| `Modules/CalcSetup.lua` L561 | 将 configTab.modList 注入计算环境 |
| `Classes/Item.lua` L72 | `{custom}` 等 lineFlags 的定义 |
| `Classes/Item.lua` L1113 | 序列化时自动加回 `{custom}` 前缀 |

### XML 中的存储格式

```xml
<ConfigSet>
    <!-- 其他配置项... -->
    <Input name="customMods" string="30% increased Spell Damage&#10;+100 to maximum Life&#10;+2 to Level of all Spell Skill Gems"/>
</ConfigSet>
```

- 多行词条用 `&#10;`（XML 换行实体）分隔
- 每次 build 加载时，`ConfigTab:Load()` 读取该值，`BuildModList()` 解析并应用

---

## 七、实战技巧

### 1. 快速对比"有没有某条词缀"的影响

在 Custom Modifiers 里加一行 → 看面板变化 → 删掉。比换装备快 10 倍。

### 2. 模拟队友增益

组队打本时，队友的光环/诅咒效果可以写进去，算出真实的组队 DPS。

### 3. 评估升级优先级

想知道"加 50 血"和"加 10% 伤"哪个更值？各写一条，对比面板变化。

### 4. 配合 `/generate-weights` 接口使用

先用 customMods 模拟你的真实游戏环境（比如队友光环），再调用 `/generate-weights` 生成权重——这样算出来的权重更贴合你的实际需求。

```python
# 1. 注入队友光环等环境修正
env_pob = inject_custom_mods(original_pob, [
    "Determination",      # 队友开坚定
    "30% more Spell Damage",  # 队友增伤光环
])

# 2. 基于真实环境生成装备权重
resp = requests.post("http://localhost:8080/generate-weights", json={
    "pob_code": env_pob,
    "slot": "Helmet",
    "stat_weights": [
        {"stat": "FullDPS", "weightMult": 1.0},
        {"stat": "TotalEHP", "weightMult": 0.5},
    ],
})
```

### 5. 逆向验证装备价值

拿到一件装备后，把它的每条词缀分别写进 Custom Modifiers，看每条贡献多少 DPS/生存，帮你判断这件装备值不值。

---

## 附录：POB 支持的全部词缀标记

在物品文本中，以下 `{tag}` 前缀都是合法的：

| 标记 | 含义 |
|------|------|
| `{custom}` | 自定义词缀（不对应真实游戏 mod，纯模拟用） |
| `{crafted}` | 工艺台词缀 |
| `{fractured}` | 裂隙词缀（不可移除） |
| `{implicit}` | 隐性词缀 |
| `{enchant}` | 附魔 |
| `{crucible}` | 坩埚词缀 |
| `{scourge}` | 灾祸词缀 |
| `{eater}` | 异界吞噬者词缀 |
| `{exarch}` | 灼炎尊师词缀 |
| `{synthesis}` | 综合词缀 |
| `{mutated}` | 变异词缀 |
