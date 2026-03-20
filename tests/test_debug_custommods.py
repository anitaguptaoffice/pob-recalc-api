import base64, zlib, re, requests

with open("/data/workspace/pob-recalc-api/pob") as f:
    pob_code = f.read().strip()

b64std = pob_code.replace('-', '+').replace('_', '/')
decoded = base64.b64decode(b64std + '==')
xml = zlib.decompress(decoded).decode('utf-8', errors='replace')

# Inject customMods with &#10; (XML entity for newline)
custom = '10% increased maximum Life&#10;+100 to all Attributes&#10;50% increased Spell Damage'
modified = xml.replace('</ConfigSet>', '<Input name="customMods" string="' + custom + '"/>\n</ConfigSet>', 1)

compressed = zlib.compress(modified.encode('utf-8'))
pob2 = base64.b64encode(compressed).decode('ascii').replace('+', '-').replace('/', '_').rstrip('=')

# Check: what does configTab.input.customMods look like after loading?
# Let's also try with literal \n instead of &#10;
custom2 = '10% increased maximum Life\n+100 to all Attributes\n50% increased Spell Damage'
modified2 = xml.replace('</ConfigSet>', '<Input name="customMods" string="' + custom2 + '"/>\n</ConfigSet>', 1)

compressed2 = zlib.compress(modified2.encode('utf-8'))
pob3 = base64.b64encode(compressed2).decode('ascii').replace('+', '-').replace('/', '_').rstrip('=')

item = "Rarity: RARE\nTest\nProphet Crown\nImplicits: 0\n+80 to maximum Life\n+40% to Fire Resistance"

# Test 1: With &#10;
r1 = requests.post('http://localhost:8080/replace-item', json={
    'pob_code': pob2, 'slot': 'Helmet', 'item_text': item
})
d1 = r1.json()

# Test 2: With literal \n
r2 = requests.post('http://localhost:8080/replace-item', json={
    'pob_code': pob3, 'slot': 'Helmet', 'item_text': item
})
d2 = r2.json()

# Test 3: baseline (no mods)
r3 = requests.post('http://localhost:8080/replace-item', json={
    'pob_code': pob_code, 'slot': 'Helmet', 'item_text': item
})
d3 = r3.json()

print("=== Baseline (no customMods) ===")
print(f"  Before TotalDPS: {d3['before']['TotalDPS']}")
print(f"  Before Life: {d3['before']['Life']}")
print(f"  Before ES: {d3['before']['EnergyShield']}")

print("\n=== With &#10; separator ===")
print(f"  Before TotalDPS: {d1['before']['TotalDPS']}")
print(f"  Before Life: {d1['before']['Life']}")
print(f"  Before ES: {d1['before']['EnergyShield']}")

print("\n=== With literal newline ===")
print(f"  Before TotalDPS: {d2['before']['TotalDPS']}")
print(f"  Before Life: {d2['before']['Life']}")
print(f"  Before ES: {d2['before']['EnergyShield']}")
