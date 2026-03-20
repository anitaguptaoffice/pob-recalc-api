import duckdb
from common import CLIENT_GLOBAL, CLIENT_TENCENT, LANG_CHS, LANG_EN, SERVER_GLOBAL, SERVER_TENCENT, at, save_ndjson
from db.utils import check_duplicate_zhs
from export import game, tree

ANOINTED_PATH = "db/passive_skills/anointed.ndjson"
KEYSTONES_PATH = "db/passive_skills/keystones.ndjson"
ASCENDANT_PATH = "db/passive_skills/ascendant.ndjson"


def select_anointed_nodes(en_nodes, zh_nodes):
    node_list = []
    node_idx = {}
    for id, node in en_nodes.items():
        if 'name' not in node:
            continue
        en = node['name']
        # 有涂油配方的天赋
        if 'recipe' in node:
            data = {"id": id, "zh": "", "en": en}
            node_list.append(data)
            node_idx[id] = data

    for id, node in zh_nodes.items():
        if id not in node_idx:
            continue
        zh = node['name']
        node_idx[id]["zh"] = zh

    check_duplicate_zhs(node_list, ANOINTED_PATH)

    return node_list


def select_keystone_nodes(en_nodes, zh_nodes):
    node_list = []
    node_idx = {}
    for id, node in en_nodes.items():
        if 'name' not in node:
            continue
        en = node['name']
        if 'isKeystone' in node and node['isKeystone']:
            data = {"id": id, "zh": "", "en": en}
            node_list.append(data)
            node_idx[id] = data

    for id, node in zh_nodes.items():
        if id not in node_idx:
            continue
        zh = node['name']
        node_idx[id]['zh'] = zh

    check_duplicate_zhs(node_list, KEYSTONES_PATH, raise_error=True)

    return node_list


def select_hidden_ascendant_nodes():
    table1 = (CLIENT_TENCENT, LANG_CHS, "PassiveSkills")
    table2 = (CLIENT_GLOBAL, LANG_EN, "PassiveSkills")

    game.load_table(*table1)
    game.load_table(*table2)

    duck_name1 = game.duck_table_name(*table1)
    duck_name2 = game.duck_table_name(*table2)

    game.create_index(duck_name1, "Id")

    # 隐藏升华的Id目前拥有前缀`AscendancySpecialEldritch`
    rows = duckdb.sql(f"""SELECT {duck_name1}.PassiveSkillGraphId, {duck_name1}.Name, {duck_name2}.Name FROM {duck_name1}
            INNER JOIN {duck_name2} ON {duck_name1}.Id = {duck_name2}.Id
            WHERE {duck_name1}.Id LIKE 'AscendancySpecialEldritch%'
        """).fetchall()
    # id使用字符串格式存储
    return [{"id": f"{r[0]}", "zh": r[1], "en": r[2]} for r in rows]


# 重命名节点中文的元数据：
# 冲突节点1的id，冲突节点2的id，冲突的中文，节点2的中文的重命名
# 详情见本项目data/README.md
NODE_ZH_RENAME_LIST = [
    ("19083", "43122", "暗影", "暗影（贵族）")
]


def select_ascendant_nodes(en_nodes, zh_nodes):
    data_list = []
    data_idx = {}

    viewed_ids = set()
    # 从根节点遍历树（图实现）
    todo_view_ids = []
    for id in en_nodes["root"]["out"]:
        todo_view_ids.append(id)
    while len(todo_view_ids) > 0:
        id = todo_view_ids.pop()
        # 跳过已查看的节点
        if id in viewed_ids:
            continue
        viewed_ids.add(id)

        node = en_nodes[id]

        # 跳过血族升华节点
        if "isBloodline" in node and node["isBloodline"]:
            continue

        # 跳过非角色入口节点，跳过非升华入口节点，遇到非升华节点返回
        if "classStartIndex" not in node and "isAscendancyStart" not in node and "ascendancyName" not in node:
            continue

        # 多选项节点自身不做记录（仅记录子选项节点）
        # 贵族的连接角色入口节点的节点不做记录，如“野蛮人之道”
        if "isMultipleChoice" in node:
            pass
        elif "ascendancyName" in node and node["ascendancyName"] == "Ascendant" and "grantedPassivePoints" in node:
            pass
        elif "isNotable" in node or "isMultipleChoiceOption" in node:
            data = {"id": id, "zh": zh_nodes[id]["name"], "en": node["name"]}
            data_list.append(data)
            data_idx[id] = data

        for out_id in node["out"]:
            if not out_id in viewed_ids:
                todo_view_ids.append(out_id)
        for in_id in node["in"]:
            if not in_id in viewed_ids:
                todo_view_ids.append(in_id)

    for entry in NODE_ZH_RENAME_LIST:
        id1, id2, zh1, zh2 = entry
        data1, data2 = data_idx[id1], data_idx[id2]
        if data1["zh"] != zh1 or data2["zh"] != zh1:
            raise Exception(f"error: 重命名升华大点的中文失败，请更新元数据: {entry}")
        data2["zh"] = zh2

    hidden_ascendant_list = select_hidden_ascendant_nodes()
    data_list.extend(hidden_ascendant_list)

    check_duplicate_zhs(data_list, ASCENDANT_PATH, raise_error=True)

    return data_list


def create_nodes():
    en_tree, zh_tree = tree.passive_skill_tree(
        SERVER_GLOBAL), tree.passive_skill_tree(SERVER_TENCENT)
    en_nodes, zh_nodes = en_tree["nodes"], zh_tree["nodes"]

    anointed = select_anointed_nodes(en_nodes, zh_nodes)
    keystones = select_keystone_nodes(en_nodes, zh_nodes)
    ascendant = select_ascendant_nodes(en_nodes, zh_nodes)

    print(f"info: 创建 {ANOINTED_PATH}...")
    save_ndjson(at(ANOINTED_PATH), anointed)
    print(f"info: 创建 {KEYSTONES_PATH}...")
    save_ndjson(at(KEYSTONES_PATH), keystones)
    print(f"info: 创建 {ASCENDANT_PATH}...")
    save_ndjson(at(ASCENDANT_PATH), ascendant)
