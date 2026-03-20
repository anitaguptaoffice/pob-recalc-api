import os
from common import at, read_json, read_ndjson
from db import skill, pair, passive_skill, stat, unique


checked_non_ascii_types = {"Maelström Staff"}
checked_non_ascii_names = {"Doppelgänger Guise", "Mjölner"}


def check_non_ascii_names():
    non_ascii_types = set()
    non_ascii_names = set()

    for file_name in os.listdir(at("db/items")):
        source = at("db/items", file_name)
        if os.path.isfile(source) and file_name.endswith(".json"):
            data = read_json(source)
            for item in data:
                basetype = item["en"]
                if not basetype.isascii():
                    non_ascii_types.add(basetype)

    uniques = read_ndjson(at(unique.UNIQUES_PATH))
    for u in uniques:
        name = u["en"]
        if not name.isascii():
            non_ascii_names.add(name)

    deprecated_types = checked_non_ascii_types-non_ascii_types
    deprecated_names = checked_non_ascii_names-non_ascii_names
    new_types = non_ascii_types-checked_non_ascii_types
    new_names = non_ascii_names-checked_non_ascii_names

    if len(deprecated_types) != 0:
        print(f"warning: deprecated non-ascii basetypes: {deprecated_types}")
    if len(deprecated_names) != 0:
        print(f"warning: deprecated non-ascii uniques: {deprecated_names}")
    if len(new_types) != 0:
        print(f"warning: new non-ascii basetypes: {new_types}")
    if len(new_names) != 0:
        print(f"warning: new non-ascii uniques: {new_names}")


def snake_to_camel(name: str):
    result = ''
    capitalize_next = False
    for char in name:
        if char == '_':
            capitalize_next = True
        else:
            if capitalize_next:
                result += char.upper()
                capitalize_next = False
            else:
                result += char
    return result


def remain_fields(obj: dict, fields: set[str]):
    keys = list(obj.keys())
    for key in keys:
        if key not in fields:
            del obj[key]
        else:
            val = obj[key]
            if type(val) is dict:
                remain_fields(val, fields)
            elif type(val) is list:
                for item in val:
                    if type(item) is dict:
                        remain_fields(item, fields)


def remain_fields_of_each(arr: list[dict], fields: set[str]) -> list[dict]:
    for item in arr:
        remain_fields(item, fields)

    return arr


def get_attributes() -> list:
    data: list = read_json(at(pair.ATTRIBUTES_PATH))
    return remain_fields_of_each(data, {"zh", "en", "values"})


def get_properties() -> list:
    data = []
    data.extend(read_json(at(pair.PROPERTIES_PATH)))
    data.extend(read_json(at(pair.PROPERTIES2_PATH)))

    return remain_fields_of_each(data, {"zh", "en", "values"})


def get_requirements() -> list:
    data: list = read_json(at(pair.REQUIREMENTS_PATH))
    return remain_fields_of_each(data, {"zh", "en", "values"})


def get_requirements_suffixes() -> list:
    data: list = read_json(at(pair.REQUIREMENT_SUFFIXES_PATH))
    return remain_fields_of_each(data, {"zh", "en", "values"})


def get_strings() -> list:
    data: list = read_json(at(pair.STRINGS_PATH))
    return remain_fields_of_each(data, {"id", "zh", "en", "type"})


def get_items() -> dict[str, list]:
    check_non_ascii_names()

    uniques = read_ndjson(at(unique.UNIQUES_PATH))
    uniques_base_type_idx = {}
    for u in uniques:
        base_type = u["baseType"]
        if base_type not in uniques_base_type_idx:
            uniques_base_type_idx[base_type] = []
        uniques_base_type_idx[base_type].append(u)

    items = {}

    for file_name in os.listdir(at("db/items")):
        source = at("db/items", file_name)
        if os.path.isfile(source) and file_name.endswith(".json"):
            data = read_json(source)
            for item in data:
                base_type = item["en"]
                if base_type in uniques_base_type_idx:
                    item["uniques"] = uniques_base_type_idx[base_type]

            name = file_name[:-5]
            remain_fields_of_each(data, {"zh", "en", "uniques"})
            items[snake_to_camel(name)] = data

    return items


def get_skills() -> dict[str, list]:
    skills = {}

    skills["gemSkills"] = remain_fields_of_each(
        read_ndjson(at(skill.GEM_SKILLS_PATH)), {"zh", "en"})
    skills["transfiguredSkills"] = remain_fields_of_each(
        read_ndjson(at(skill.TRANSFIGURED_SKILLS_PATH)), {"zh", "en"})
    skills["hybridSkills"] = remain_fields_of_each(
        read_ndjson(at(skill.HYBRID_SKILLS_PATH)), {"zh", "en"})
    skills["indexableSupports"] = remain_fields_of_each(
        read_ndjson(at(skill.INDEXABLE_SUPPORTS_PATH)), {"zh", "en"})

    return skills


def get_passive_skills() -> dict[str, list]:
    skills = {}
    skills["anointed"] = remain_fields_of_each(
        read_ndjson(at(passive_skill.ANOINTED_PATH)), {"zh", "en"})
    skills["keystones"] = remain_fields_of_each(
        read_ndjson(at(passive_skill.KEYSTONES_PATH)), {"zh", "en"})
    skills["ascendant"] = remain_fields_of_each(
        read_ndjson(at(passive_skill.ASCENDANT_PATH)), {"zh", "en"})
    return skills


def remove_repeats(stats):
    stat_list = []
    stat_map = {}
    for stat in stats:
        zh = stat["zh"]
        en = stat["en"]
        if zh in stat_map:
            old_en = stat_map[zh]["en"]
            if en.casefold() != old_en.casefold():
                print("warning: same zh but diff en")
                print(f"{zh}")
                print(f"{old_en}")  # old
                print(f"{en}")  # old
            continue
        stat_list.append(stat)
        stat_map[zh] = stat
    return stat_list


def get_stats() -> list:
    stats = []
    stats.extend(read_json(at(stat.DESC_STATS_PATH)))
    stats.extend(read_json(at(stat.TRADE_STATS_PATH)))

    stats = remove_repeats(stats)

    # 还需要保留refs字段以及refs中的参数索引，这里假设索引最大为5
    return remain_fields_of_each(stats, {"zh", "en", "refs", "0", "1", "2", "3", "4", "5"})


def get_all() -> dict[str, list]:
    all = {}
    all["attributes"] = get_attributes()
    all["properties"] = get_properties()
    all["requirements"] = get_requirements()
    all["requirementSuffixes"] = get_requirements_suffixes()
    all["strings"] = get_strings()

    for name, array in get_items().items():
        all[name] = array

    for name, array in get_skills().items():
        all[name] = array

    for name, array in get_passive_skills().items():
        all[name] = array

    all["stats"] = get_stats()

    return all
