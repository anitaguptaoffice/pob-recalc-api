#!/usr/bin/env python3
"""Test script for /replace-item endpoint"""
import json
import urllib.request
import sys

# Read POB code from file
with open("/data/workspace/pob-recalc-api/pob", "r") as f:
    pob_code = f.read().strip()

# New item to replace into the Helmet slot
# Current: Crown of the Inward Eye (Unique)
# Replace with: A rare helmet with high ES and resistances
new_item = (
    "Rarity: RARE\n"
    "Apocalypse Crown\n"
    "Prophet Crown\n"
    "Armour: 1400\n"
    "Energy Shield: 250\n"
    "Item Level: 86\n"
    "Quality: 20\n"
    "LevelReq: 63\n"
    "Implicits: 0\n"
    "400% increased Armour and Energy Shield\n"
    "+80 to maximum Life\n"
    "+60 to maximum Mana\n"
    "30% increased maximum Energy Shield\n"
    "+40% to Fire Resistance\n"
    "+35% to Cold Resistance"
)

payload = {
    "pob_code": pob_code,
    "slot": "Helmet",
    "item_text": new_item,
}

print(f"Sending request to /replace-item...")
print(f"  POB code length: {len(pob_code)} chars")
print(f"  Slot: {payload['slot']}")
print(f"  New item:\n    {new_item.replace(chr(10), chr(10) + '    ')}")
print()

data = json.dumps(payload).encode("utf-8")
req = urllib.request.Request(
    "http://localhost:8080/replace-item",
    data=data,
    headers={"Content-Type": "application/json"},
    method="POST",
)

try:
    with urllib.request.urlopen(req, timeout=120) as resp:
        result = json.loads(resp.read().decode("utf-8"))
        print("=== Response ===")
        print(json.dumps(result, indent=2, ensure_ascii=False))
        
        # Summary
        print("\n=== Summary ===")
        if "before" in result and "after" in result:
            before = result["before"]
            after = result["after"]
            diff = result.get("diff", {})
            
            print(f"Slot: {result.get('slot', '?')}")
            print(f"Old Item ID: {result.get('old_item_id', '?')}")
            print(f"New Item ID: {result.get('new_item_id', '?')}")
            print()
            print(f"{'Stat':<25} {'Before':>15} {'After':>15} {'Diff':>15}")
            print("-" * 72)
            for key in sorted(before.keys()):
                b = before.get(key, 0)
                a = after.get(key, 0)
                d = diff.get(key, 0)
                if d != 0:
                    sign = "+" if d > 0 else ""
                    print(f"  {key:<23} {b:>15.1f} {a:>15.1f} {sign}{d:>14.1f}")
            print()
            print("(Only showing stats that changed)")
except urllib.error.HTTPError as e:
    print(f"HTTP Error {e.code}: {e.read().decode()}")
except Exception as e:
    print(f"Error: {e}")
