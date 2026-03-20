#!/usr/bin/env python3
"""Test the /generate-weights endpoint."""
import requests
import json
import sys
import time

API = "http://localhost:8080"

# Read POB code
with open("pob", "r") as f:
    pob_code = f.read().strip()

print("=" * 70)
print("TEST: /generate-weights")
print("=" * 70)

# Test 1: Default stat weights (FullDPS + TotalEHP)
print("\n--- Test 1: Helmet slot, default weights (FullDPS=1.0, TotalEHP=0.5) ---")
start = time.time()
resp = requests.post(f"{API}/generate-weights", json={
    "pob_code": pob_code,
    "slot": "Helmet",
    "stat_weights": [
        {"stat": "FullDPS", "weightMult": 1.0},
        {"stat": "TotalEHP", "weightMult": 0.5},
    ],
    "include_corrupted": False,
    "include_eldritch": False,
})
elapsed = time.time() - start

if resp.status_code != 200:
    print(f"ERROR: status={resp.status_code}")
    print(resp.text[:500])
    sys.exit(1)

data = resp.json()
print(f"  Status: {resp.status_code} ({elapsed:.1f}s)")
print(f"  Slot: {data.get('slot')}")
print(f"  Item Category: {data.get('item_category')}")
print(f"  Current Item: {data.get('current_item')}")
print(f"  Current Stat Diff: {data.get('current_stat_diff', 0):.2f}")
print(f"  Mods Tested: {data.get('mods_tested')}")
print(f"  Weights Found: {len(data.get('mod_weights', []))}")

weights = data.get("mod_weights", [])
if weights:
    print(f"\n  Top 20 mod weights (sorted by impact):")
    print(f"  {'#':>3} {'Weight':>10} {'Impact':>10} {'Mod':50}")
    print(f"  {'---':>3} {'------':>10} {'------':>10} {'---':50}")
    for i, mw in enumerate(weights[:20]):
        print(f"  {i+1:>3} {mw['weight']:>10.2f} {mw['mean_stat_diff']:>10.2f} {mw['mod_text']:50} [{mw.get('mod_type','')}]")

# Test 2: Pure DPS weights on Ring slot
print("\n\n--- Test 2: Ring 1 slot, pure DPS weight ---")
start = time.time()
resp2 = requests.post(f"{API}/generate-weights", json={
    "pob_code": pob_code,
    "slot": "Ring 1",
    "stat_weights": [
        {"stat": "FullDPS", "weightMult": 1.0},
    ],
})
elapsed2 = time.time() - start

if resp2.status_code != 200:
    print(f"ERROR: status={resp2.status_code}")
    print(resp2.text[:500])
else:
    data2 = resp2.json()
    print(f"  Status: {resp2.status_code} ({elapsed2:.1f}s)")
    print(f"  Slot: {data2.get('slot')}")
    print(f"  Current Item: {data2.get('current_item')}")
    print(f"  Mods Tested: {data2.get('mods_tested')}")
    print(f"  Weights Found: {len(data2.get('mod_weights', []))}")

    weights2 = data2.get("mod_weights", [])
    if weights2:
        print(f"\n  Top 15 mod weights:")
        print(f"  {'#':>3} {'Weight':>10} {'Impact':>10} {'Mod':50}")
        print(f"  {'---':>3} {'------':>10} {'------':>10} {'---':50}")
        for i, mw in enumerate(weights2[:15]):
            print(f"  {i+1:>3} {mw['weight']:>10.2f} {mw['mean_stat_diff']:>10.2f} {mw['mod_text']:50}")

# Test 3: Weapon slot
print("\n\n--- Test 3: Weapon 1 slot ---")
start = time.time()
resp3 = requests.post(f"{API}/generate-weights", json={
    "pob_code": pob_code,
    "slot": "Weapon 1",
    "stat_weights": [
        {"stat": "FullDPS", "weightMult": 1.0},
    ],
})
elapsed3 = time.time() - start

if resp3.status_code != 200:
    print(f"ERROR: status={resp3.status_code}")
    print(resp3.text[:500])
else:
    data3 = resp3.json()
    print(f"  Status: {resp3.status_code} ({elapsed3:.1f}s)")
    print(f"  Slot: {data3.get('slot')}")
    print(f"  Item Category: {data3.get('item_category')}")
    print(f"  Current Item: {data3.get('current_item')}")
    print(f"  Mods Tested: {data3.get('mods_tested')}")
    print(f"  Weights Found: {len(data3.get('mod_weights', []))}")

    weights3 = data3.get("mod_weights", [])
    if weights3:
        print(f"\n  Top 15 mod weights:")
        print(f"  {'#':>3} {'Weight':>10} {'Impact':>10} {'Mod':50}")
        print(f"  {'---':>3} {'------':>10} {'------':>10} {'---':50}")
        for i, mw in enumerate(weights3[:15]):
            print(f"  {i+1:>3} {mw['weight']:>10.2f} {mw['mean_stat_diff']:>10.2f} {mw['mod_text']:50}")

print("\n" + "=" * 70)
print("ALL TESTS COMPLETE")
print("=" * 70)
