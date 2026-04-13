import requests
import json
import re
import base64
import zlib

# 读取 POB code
with open("tests/testdata/fixture.txt", "r") as f:
    pob_code = f.read().strip()

def decode_pob(pob_code):
    pob_code = pob_code.strip()
    b64std = pob_code.replace('-', '+').replace('_', '/')
    try:
        decoded = base64.b64decode(b64std + '==')
    except:
        decoded = base64.b64decode(b64std)
    return zlib.decompress(decoded).decode('utf-8', errors='replace')

def encode_pob(xml_str):
    compressed = zlib.compress(xml_str.encode('utf-8'))
    b64 = base64.b64encode(compressed).decode('ascii')
    return b64.replace('+', '-').replace('/', '_').rstrip('=')

# 解码原始 XML 并注入 customMods
original_xml = decode_pob(pob_code)
custom_mods = "10% increased maximum Life&#10;+100 to all Attributes&#10;50% increased Spell Damage"
modified_xml = original_xml.replace(
    '</ConfigSet>',
    '<Input name="customMods" string="' + custom_mods + '"/>\n</ConfigSet>',
    1
)
modified_pob = encode_pob(modified_xml)

# 测试装备
item_text = """Rarity: RARE
Test Helmet
Prophet Crown
Implicits: 0
+80 to maximum Life
+40% to Fire Resistance
+30% to Lightning Resistance"""

# 1. 无 customMods
print("=== 1. /replace-item WITHOUT customMods ===")
r1 = requests.post("http://localhost:8080/replace-item", json={
    "pob_code": pob_code,
    "slot": "Helmet",
    "item_text": item_text
})
d1 = r1.json()

# 2. 有 customMods
print("=== 2. /replace-item WITH customMods ===")
r2 = requests.post("http://localhost:8080/replace-item", json={
    "pob_code": modified_pob,
    "slot": "Helmet",
    "item_text": item_text
})
d2 = r2.json()

# 对比 before 值（应该不同，因为 customMods 作为全局加成影响了 baseline）
print(f"\n=== Comparison ===")
print(f"{'Stat':<25} {'No Mods Before':>15} {'With Mods Before':>17} {'Diff':>10}")
print("-" * 70)

keys = ['TotalDPS', 'CombinedDPS', 'Life', 'EnergyShield',
        'MinionTotalDPS', 'MinionCombinedDPS']

for k in keys:
    v1 = d1.get('before', {}).get(k, 0)
    v2 = d2.get('before', {}).get(k, 0)
    if v1 or v2:
        diff = v2 - v1
        sign = '+' if diff >= 0 else ''
        print(f"  {k:<23} {v1:>15.0f} {v2:>17.0f} {sign}{diff:>9.0f}")

print(f"\n{'Stat':<25} {'No Mods After':>15} {'With Mods After':>17} {'Diff':>10}")
print("-" * 70)

for k in keys:
    v1 = d1.get('after', {}).get(k, 0)
    v2 = d2.get('after', {}).get(k, 0)
    if v1 or v2:
        diff = v2 - v1
        sign = '+' if diff >= 0 else ''
        print(f"  {k:<23} {v1:>15.0f} {v2:>17.0f} {sign}{diff:>9.0f}")

print("\n=== Conclusion ===")
before_no = d1.get('before', {}).get('TotalDPS', 0)
before_yes = d2.get('before', {}).get('TotalDPS', 0)
if before_yes > before_no:
    print("✅ customMods IS working! Global custom modifiers affect the calculation.")
    print(f"   TotalDPS increased from {before_no:.0f} to {before_yes:.0f} ({(before_yes-before_no)/before_no*100:.1f}% boost)")
else:
    print("❌ customMods did NOT affect the calculation.")
