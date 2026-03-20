import re
from typing import Any
from common import CLIENT_GLOBAL, LANG_EN, at_pob, read_json
from db import pob
from export import game


def get_slim_classes(tree):
    slim_classes = []

    for c in tree["classes"]:
        slim_c = {"name": c["name"]}
        slim_ascendancy_list = []
        for ascendancy in c["ascendancies"]:
            slim_ascendancy_list.append({"name": ascendancy["name"]})
        slim_c["ascendancies"] = slim_ascendancy_list
        slim_classes.append(slim_c)

    return slim_classes


def get_slim_nodes(tree):
    slim_nodes = {}

    nodes: dict = tree["nodes"]
    for id, node in nodes.items():
        if "expansionJewel" in node:
            slim_nodes[id] = {
                "expansionJewel": node["expansionJewel"],
                "orbit": node["orbit"],
                "orbitIndex": node["orbitIndex"]
            }
            proxyNodeId = node["expansionJewel"]["proxy"]
            proxyNode = nodes[proxyNodeId]

            slim_nodes[proxyNodeId] = {
                "orbit": proxyNode["orbit"],
                "orbitIndex": proxyNode["orbitIndex"]
            }

    return slim_nodes


def get_slim_tree():
    tree = pob.get_tree(pob.latest_tree_version())

    return {
        "classes": get_slim_classes(tree),
        "nodes": get_slim_nodes(tree),
        "jewelSlots": tree["jewelSlots"],
        "constants": tree["constants"]
    }


def get_phrecia_ascendancy_map():
    m = {}

    table = read_json(game.table_path(CLIENT_GLOBAL, LANG_EN, "Ascendancy"))
    for record in table:
        name = record["Name"]
        league_name = record["LeagueName"]
        m[name] = league_name

    return m


RARITY_MAP_IN_IMPORT_TAB_PATTERN = r"local rarityMap = {[^}]+}"
SLOT_MAP_IN_IMPORT_TAB_PATTERN = r"local slotMap = {[^}]+}"


def get_rarity_map_and_slot_map():
    with open(at_pob("Classes/ImportTab.lua"), 'r', encoding='utf-8') as f:
        content = f.read()

        regex = re.compile(RARITY_MAP_IN_IMPORT_TAB_PATTERN)
        code = regex.findall(content)[0]
        rarity_map_json = pob.local2json(code, "rarityMap")

        regex = re.compile(SLOT_MAP_IN_IMPORT_TAB_PATTERN)
        code = regex.findall(content)[0]
        slot_map_json = pob.local2json(code, "slotMap")

        return (rarity_map_json, slot_map_json)


def get_slim_cluster_jewel_metadata():
    metadata = pob.get_cluster_jewel_metadata()

    jewels: dict = metadata["jewels"]
    slim_jewels = {}

    for key, value in jewels.items():
        slim_jewels[key] = {
            "size": value["size"],
            "sizeIndex": value["sizeIndex"],
            "smallIndicies": value["smallIndicies"],
            "notableIndicies": value["notableIndicies"],
            "socketIndicies": value["socketIndicies"],
            "totalIndicies": value["totalIndicies"],
        }

    return {
        "jewels": slim_jewels,
    }


def get_all() -> dict[str, Any]:
    (rarityMap, slotMap) = get_rarity_map_and_slot_map()

    return {
        "tree": get_slim_tree(),
        "phreciaAscendancyMap": get_phrecia_ascendancy_map(),
        "rarityMap": rarityMap,
        "slotMap": slotMap,
        "clusterJewels": get_slim_cluster_jewel_metadata(),
    }
