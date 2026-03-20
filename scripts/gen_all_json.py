#!/usr/bin/env python3
"""
Generate all.json for cn-poe-utils Go translator.
This is a standalone script that reads from data/db/ and outputs the combined
translation data in the format expected by poe.Data.

Usage:
    python3 scripts/gen_all_json.py
"""

import json
import os
import sys


DB_ROOT = os.path.join(os.path.dirname(__file__), "..", "cn-poe-utils", "data", "db")
OUTPUT = os.path.join(os.path.dirname(__file__), "..", "translate_data", "all.json")


def read_json(path):
    with open(path, "rt", encoding="utf-8") as f:
        return json.load(f)


def read_ndjson(path):
    result = []
    with open(path, "rt", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line:
                result.append(json.loads(line))
    return result


def snake_to_camel(name: str) -> str:
    result = ""
    capitalize_next = False
    for char in name:
        if char == "_":
            capitalize_next = True
        else:
            if capitalize_next:
                result += char.upper()
                capitalize_next = False
            else:
                result += char
    return result


def remain_fields(obj, fields):
    if isinstance(obj, dict):
        keys = list(obj.keys())
        for key in keys:
            if key not in fields:
                del obj[key]
            else:
                remain_fields(obj[key], fields)
    elif isinstance(obj, list):
        for item in obj:
            remain_fields(item, fields)


def remain_fields_of_each(arr, fields):
    for item in arr:
        remain_fields(item, fields)
    return arr


def get_attributes():
    data = read_json(os.path.join(DB_ROOT, "attributes.json"))
    return remain_fields_of_each(data, {"zh", "en", "values"})


def get_properties():
    data = []
    data.extend(read_json(os.path.join(DB_ROOT, "properties.json")))
    p2 = os.path.join(DB_ROOT, "properties2.json")
    if os.path.exists(p2):
        data.extend(read_json(p2))
    return remain_fields_of_each(data, {"zh", "en", "values"})


def get_requirements():
    data = read_json(os.path.join(DB_ROOT, "requirements.json"))
    return remain_fields_of_each(data, {"zh", "en", "values"})


def get_requirement_suffixes():
    data = read_json(os.path.join(DB_ROOT, "requirement_suffixes.json"))
    return remain_fields_of_each(data, {"zh", "en", "values"})


def get_strings():
    data = read_json(os.path.join(DB_ROOT, "strings.json"))
    return remain_fields_of_each(data, {"id", "zh", "en", "type"})


def get_items():
    uniques_path = os.path.join(DB_ROOT, "uniques.ndjson")
    uniques = read_ndjson(uniques_path) if os.path.exists(uniques_path) else []
    uniques_base_type_idx = {}
    for u in uniques:
        base_type = u.get("baseType", u.get("en", ""))
        if base_type not in uniques_base_type_idx:
            uniques_base_type_idx[base_type] = []
        uniques_base_type_idx[base_type].append(u)

    items = {}
    items_dir = os.path.join(DB_ROOT, "items")
    for file_name in sorted(os.listdir(items_dir)):
        if not file_name.endswith(".json"):
            continue
        source = os.path.join(items_dir, file_name)
        data = read_json(source)
        for item in data:
            base_type = item.get("en", "")
            if base_type in uniques_base_type_idx:
                item["uniques"] = uniques_base_type_idx[base_type]

        name = file_name[:-5]
        remain_fields_of_each(data, {"zh", "en", "uniques"})
        items[snake_to_camel(name)] = data

    return items


def get_skills():
    skills_dir = os.path.join(DB_ROOT, "skills")
    skills = {}
    skills["gemSkills"] = remain_fields_of_each(
        read_ndjson(os.path.join(skills_dir, "gem_skills.ndjson")), {"zh", "en"}
    )
    skills["transfiguredSkills"] = remain_fields_of_each(
        read_ndjson(os.path.join(skills_dir, "transfigured_skills.ndjson")), {"zh", "en"}
    )
    skills["hybridSkills"] = remain_fields_of_each(
        read_ndjson(os.path.join(skills_dir, "hybrid_skills.ndjson")), {"zh", "en"}
    )
    skills["indexableSupports"] = remain_fields_of_each(
        read_ndjson(os.path.join(skills_dir, "indexable_supports.ndjson")), {"zh", "en"}
    )
    return skills


def get_passive_skills():
    ps_dir = os.path.join(DB_ROOT, "passive_skills")
    skills = {}
    skills["anointed"] = remain_fields_of_each(
        read_ndjson(os.path.join(ps_dir, "anointed.ndjson")), {"zh", "en"}
    )
    skills["keystones"] = remain_fields_of_each(
        read_ndjson(os.path.join(ps_dir, "keystones.ndjson")), {"zh", "en"}
    )
    skills["ascendant"] = remain_fields_of_each(
        read_ndjson(os.path.join(ps_dir, "ascendant.ndjson")), {"zh", "en"}
    )
    return skills


def remove_repeats(stats):
    stat_list = []
    stat_map = {}
    for stat in stats:
        zh = stat["zh"]
        if zh in stat_map:
            continue
        stat_list.append(stat)
        stat_map[zh] = stat
    return stat_list


def get_stats():
    stats_dir = os.path.join(DB_ROOT, "stats")
    stats = []
    stats.extend(read_json(os.path.join(stats_dir, "desc.json")))
    trade_path = os.path.join(stats_dir, "trade.json")
    if os.path.exists(trade_path):
        stats.extend(read_json(trade_path))
    stats = remove_repeats(stats)
    return remain_fields_of_each(
        stats, {"zh", "en", "refs", "0", "1", "2", "3", "4", "5"}
    )


def main():
    all_data = {}
    all_data["attributes"] = get_attributes()
    all_data["properties"] = get_properties()
    all_data["requirements"] = get_requirements()
    all_data["requirementSuffixes"] = get_requirement_suffixes()
    all_data["strings"] = get_strings()

    for name, array in get_items().items():
        all_data[name] = array

    for name, array in get_skills().items():
        all_data[name] = array

    for name, array in get_passive_skills().items():
        all_data[name] = array

    all_data["stats"] = get_stats()

    os.makedirs(os.path.dirname(OUTPUT), exist_ok=True)
    with open(OUTPUT, "wt", encoding="utf-8") as f:
        json.dump(all_data, f, ensure_ascii=False, indent=2)

    size_mb = os.path.getsize(OUTPUT) / (1024 * 1024)
    print(f"Generated {OUTPUT} ({size_mb:.1f} MB)")
    print(f"  attributes: {len(all_data['attributes'])}")
    print(f"  properties: {len(all_data['properties'])}")
    print(f"  requirements: {len(all_data['requirements'])}")
    print(f"  stats: {len(all_data['stats'])}")
    keys = [k for k in all_data.keys() if k not in (
        "attributes", "properties", "requirements", "requirementSuffixes",
        "strings", "stats", "gemSkills", "transfiguredSkills", "hybridSkills",
        "indexableSupports", "anointed", "keystones", "ascendant"
    )]
    for k in keys:
        if isinstance(all_data[k], list):
            print(f"  {k}: {len(all_data[k])}")


if __name__ == "__main__":
    main()
