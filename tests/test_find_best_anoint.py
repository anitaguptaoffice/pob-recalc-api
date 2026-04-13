#!/usr/bin/env python3
"""Test script for /find-best-anoint endpoint"""
import json
import urllib.request
import sys
import time

# Read POB code from file
with open("tests/testdata/fixture.txt", "r") as f:
    pob_code = f.read().strip()

payload = {
    "pob_code": pob_code,
    "stat": "CombinedDPS",   # Sort by: CombinedDPS, TotalDPS, TotalDot, Life, EnergyShield, etc.
    "max_results": 20,
    # "search": "",          # Optional: filter by name or mod text
    # "slot_name": "Amulet", # Default: Amulet
}

print(f"Sending request to /find-best-anoint...")
print(f"  POB code length: {len(pob_code)} chars")
print(f"  Stat: {payload['stat']}")
print(f"  Max results: {payload['max_results']}")
print()

data = json.dumps(payload).encode("utf-8")
req = urllib.request.Request(
    "http://localhost:8080/find-best-anoint",
    data=data,
    headers={"Content-Type": "application/json"},
    method="POST",
)

start = time.time()

try:
    with urllib.request.urlopen(req, timeout=300) as resp:
        elapsed = time.time() - start
        result = json.loads(resp.read().decode("utf-8"))

        decode_ms = resp.headers.get("X-Decode-Time-Ms", "?")
        calc_ms = resp.headers.get("X-Calc-Time-Ms", "?")
        total_ms = resp.headers.get("X-Total-Time-Ms", "?")

        print(f"=== Response (took {elapsed:.1f}s) ===")
        print(f"  Decode: {decode_ms}ms, Calc: {calc_ms}ms, Total: {total_ms}ms")
        print(f"  Stat: {result.get('stat', '?')}")
        print(f"  Slot: {result.get('slot', '?')}")
        print(f"  Current item: {result.get('current_item', '?')}")
        print(f"  Nodes tested: {result.get('nodes_tested', '?')}")
        print(f"  Total anointable: {result.get('total_anointable', '?')}")
        print(f"  Current anoints: {result.get('current_anoints', [])}")
        print()

        results = result.get("results", [])
        if results:
            print(f"{'#':<4} {'Name':<40} {'Diff':>12} {'New Value':>14} {'Base Value':>14} {'Flags'}")
            print("-" * 100)
            for i, r in enumerate(results):
                flags = []
                if r.get("is_allocated"):
                    flags.append("ALLOC")
                if r.get("is_current_anoint"):
                    flags.append("CURRENT")
                if r.get("is_keystone"):
                    flags.append("KEY")
                flag_str = " ".join(flags)

                diff = r.get("diff", 0)
                sign = "+" if diff > 0 else ""
                print(f"{i+1:<4} {r.get('name', '?'):<40} {sign}{diff:>11.1f} {r.get('new_value', 0):>14.1f} {r.get('base_value', 0):>14.1f} {flag_str}")
            print()

            # Show top result details
            top = results[0]
            print(f"=== Top Result Details ===")
            print(f"  Name: {top.get('name', '?')}")
            print(f"  Node ID: {top.get('node_id', '?')}")
            print(f"  Oil recipe indices: {top.get('oil_recipe', [])}")
            print(f"  Description:")
            for line in top.get("description", "").split("\n"):
                if line.strip():
                    print(f"    {line}")
        else:
            print("No anoint results found.")

        # Optionally dump full JSON
        if "--json" in sys.argv:
            print("\n=== Full JSON ===")
            print(json.dumps(result, indent=2, ensure_ascii=False))

except urllib.error.HTTPError as e:
    print(f"HTTP Error {e.code}: {e.read().decode()}")
except Exception as e:
    print(f"Error: {e}")
