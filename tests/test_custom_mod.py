#!/usr/bin/env python3
"""Test: replace item with {custom} modifiers (extra mods to simulate uncalculated bonuses)"""
import json
import urllib.request

# Read POB code from file
with open("tests/testdata/fixture.txt", "r") as f:
    pob_code = f.read().strip()

# Test 1: 普通稀有头盔 (baseline，不带 custom mod)
item_without_custom = (
    "Rarity: RARE\n"
    "Test Helmet Base\n"
    "Prophet Crown\n"
    "Energy Shield: 200\n"
    "Item Level: 86\n"
    "Quality: 20\n"
    "Implicits: 0\n"
    "+80 to maximum Life\n"
    "+60 to maximum Mana\n"
    "+40% to Fire Resistance\n"
    "+35% to Cold Resistance"
)

# Test 2: 同样的头盔，加上 {custom} 自定义词缀模拟额外增伤
item_with_custom = (
    "Rarity: RARE\n"
    "Test Helmet Custom\n"
    "Prophet Crown\n"
    "Energy Shield: 200\n"
    "Item Level: 86\n"
    "Quality: 20\n"
    "Implicits: 0\n"
    "+80 to maximum Life\n"
    "+60 to maximum Mana\n"
    "+40% to Fire Resistance\n"
    "+35% to Cold Resistance\n"
    "{custom}30% increased Spell Damage\n"
    "{custom}+2 to Level of all Minion Skill Gems"
)

def test_replace(label, item_text):
    payload = {
        "pob_code": pob_code,
        "slot": "Helmet",
        "item_text": item_text,
    }
    data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(
        "http://localhost:8080/replace-item",
        data=data,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=120) as resp:
        return json.loads(resp.read().decode("utf-8"))

print("=" * 70)
print("Test 1: 普通稀有头盔 (无 custom mod)")
print("=" * 70)
r1 = test_replace("no-custom", item_without_custom)

print(f"  TotalDPS:    {r1['before']['TotalDPS']:>12.0f}  ->  {r1['after']['TotalDPS']:>12.0f}  (diff: {r1['diff']['TotalDPS']:+.0f})")
print(f"  CombinedDPS: {r1['before']['CombinedDPS']:>12.0f}  ->  {r1['after']['CombinedDPS']:>12.0f}  (diff: {r1['diff']['CombinedDPS']:+.0f})")
print(f"  Life:        {r1['before']['Life']:>12.0f}  ->  {r1['after']['Life']:>12.0f}  (diff: {r1['diff']['Life']:+.0f})")
print(f"  ES:          {r1['before']['EnergyShield']:>12.0f}  ->  {r1['after']['EnergyShield']:>12.0f}  (diff: {r1['diff']['EnergyShield']:+.0f})")

print()
print("=" * 70)
print("Test 2: 同样头盔 + {custom} 自定义词缀")
print("  额外: 30% increased Spell Damage")
print("  额外: +2 to Level of all Minion Skill Gems")
print("=" * 70)
r2 = test_replace("with-custom", item_with_custom)

print(f"  TotalDPS:    {r2['before']['TotalDPS']:>12.0f}  ->  {r2['after']['TotalDPS']:>12.0f}  (diff: {r2['diff']['TotalDPS']:+.0f})")
print(f"  CombinedDPS: {r2['before']['CombinedDPS']:>12.0f}  ->  {r2['after']['CombinedDPS']:>12.0f}  (diff: {r2['diff']['CombinedDPS']:+.0f})")
print(f"  Life:        {r2['before']['Life']:>12.0f}  ->  {r2['after']['Life']:>12.0f}  (diff: {r2['diff']['Life']:+.0f})")
print(f"  ES:          {r2['before']['EnergyShield']:>12.0f}  ->  {r2['after']['EnergyShield']:>12.0f}  (diff: {r2['diff']['EnergyShield']:+.0f})")

print()
print("=" * 70)
print("对比: custom mod 带来的增益")
print("=" * 70)
dps_gain = r2['after']['TotalDPS'] - r1['after']['TotalDPS']
cdps_gain = r2['after']['CombinedDPS'] - r1['after']['CombinedDPS']
print(f"  TotalDPS 额外增益:    {dps_gain:+.0f}")
print(f"  CombinedDPS 额外增益: {cdps_gain:+.0f}")

if dps_gain != 0 or cdps_gain != 0:
    print("\n  ✅ {custom} 词缀生效了！POB 正确计算了自定义词缀的影响。")
else:
    print("\n  ❌ {custom} 词缀似乎没有生效，DPS 没有变化。")
