import duckdb
from common import CLIENT_GLOBAL, CLIENT_TENCENT, LANG_CHS, LANG_EN, at, read_ndjson, save_ndjson
from db.pair import update_pairs
from export import game, trade

UNIQUES_PATH = "db/uniques.ndjson"


# 遗产的传奇
LEGACY_UNIQUES_BEFORE_TENCENT = [
    "Deshret's Vise|Steel Gauntlets",
    "Dusktoe|Leatherscale Boots",
    "Hellbringer|Conjurer Gloves",
    "Agnerod|Imperial Staff",  # 交易网站上帝国长杖的索引名，实际是4个独立的暗金
]


def unique_names_in_stash_layout():
    """根据UniqueStashLayout表，从Words表中导出传奇名称"""
    table1 = (CLIENT_TENCENT, LANG_CHS, "UniqueStashLayout")
    table2 = (CLIENT_TENCENT, LANG_CHS, "Words")
    table3 = (CLIENT_GLOBAL, LANG_EN, "Words")
    table4 = (CLIENT_TENCENT, LANG_CHS, "UniqueStashTypes")

    game.load_table(*table1)
    game.load_table(*table2)
    game.load_table(*table3)
    game.load_table(*table4)

    duck_name1 = game.duck_table_name(*table1)
    duck_name2 = game.duck_table_name(*table2)
    duck_name3 = game.duck_table_name(*table3)
    duck_name4 = game.duck_table_name(*table4)

    game.create_index(duck_name1, "UniqueStashTypesKey")
    game.create_index(duck_name1, "WordsKey")
    game.create_index(duck_name2, "_index")
    game.create_index(duck_name2, "Text")
    game.create_index(duck_name2, "Wordlist")
    game.create_index(duck_name3, "Text")
    game.create_index(duck_name3, "Wordlist")
    game.create_index(duck_name4, "_index")

    # 过滤掉守望石、地图、盗贼契约等非装备类暗金
    rows = duckdb.sql(f"""SELECT {duck_name2}.Text2, {duck_name3}.Text2, {duck_name2}.Text FROM {duck_name1}
            INNER JOIN {duck_name2} ON {duck_name1}.WordsKey = {duck_name2}._index
            INNER JOIN {duck_name3} ON {duck_name2}.Text = {duck_name3}.Text and {duck_name2}.Wordlist = {duck_name3}.Wordlist
            INNER JOIN {duck_name4} ON {duck_name1}.UniqueStashTypesKey = {duck_name4}._index
            WHERE {duck_name4}.Id not in {"Watchstone", "Map", "HeistContract"}
        """).fetchall()
    return [{"zh": r[0], "en": r[1], "key": r[2]} for r in rows]


def update_uniques_inner(array: list):
    # 先更新已有暗金
    update_pairs(array, "db/uniques", table_info="Words,Text,Text2",
                 join_fields={"Wordlist"}, filter={"Wordlist": 6})  # Words表中Wordlist为6的记录是传奇的名称

    # 找出新的暗金
    dat_unique_names = unique_names_in_stash_layout()
    dat_unique_names_idx = {u["zh"]+"|"+u["en"]: u for u in dat_unique_names}
    new_uniques_names = dat_unique_names_idx.keys(
    )-{item["zh"]+"|"+item["en"] for item in array}

    new_unique_array = []
    for name in new_uniques_names:
        print("info: [uniques] 发现新的暗金：", name)
        u = dat_unique_names_idx[name]
        new_unique_array.append(
            {"zh": u["zh"], "en": u["en"], "baseType": "", "key": u["key"]})

    # 使用交易数据来更新暗金的baseType字段
    global_trade_uniques = trade.equipment_uniques(CLIENT_GLOBAL)
    global_trade_uniques_name_idx = {}
    for u in global_trade_uniques:
        name = u["name"]
        if name not in global_trade_uniques_name_idx:
            global_trade_uniques_name_idx[name] = []
        global_trade_uniques_name_idx[name].append(u)
    for u in new_unique_array:
        if not u["baseType"]:
            if u["en"] in global_trade_uniques_name_idx:
                matches = global_trade_uniques_name_idx[u["en"]]
                if len(matches) > 1:
                    print("warning: [uniques] 无法更新暗金的基底类型，因为在交易数据中匹配到多个：",
                          u["zh"], u["en"], matches[0]["type"], matches[1]["type"], "...")
                else:
                    u["baseType"] = matches[0]["type"]

    # 将新增的暗金加入到array中
    array.extend(new_unique_array)

    # 使用交易数据来检查数据库中的数据是否缺失或过时
    trade_uniques_fullname_list = [
        f"{u['name']}|{u['type']}" for u in global_trade_uniques]
    exist_uniques_fullname_list = [
        f"{u['en']}|{u['baseType']}" for u in array if u['baseType']]
    for u in array:
        if not u["baseType"]:
            continue
        full_name = f"{u['en']}|{u['baseType']}"
        if full_name not in trade_uniques_fullname_list:
            print(f"warning: [uniques] 暗金在交易数据中不存在：{full_name}")
    for u in global_trade_uniques:
        full_name = f"{u['name']}|{u['type']}"
        # 跳过遗产传奇，因为它们在本地数据中不存在
        if full_name in LEGACY_UNIQUES_BEFORE_TENCENT:
            continue
        if full_name not in exist_uniques_fullname_list:
            print(f"warning: [uniques] 暗金在本地数据中不存在：{full_name}")


def check_repeated(uniques: list[dict]):
    names = set()
    for u in uniques:
        zh = u["zh"]
        en = u["en"]
        base_type = u["baseType"]
        name = f"{zh}|{en}|{base_type}"
        if name in names:
            print(f"warning: [uniques] 重复的暗金数据：{name}")
        else:
            names.add(name)


def update():
    print(f"info: 更新 {UNIQUES_PATH}...")
    uniques = read_ndjson(at(UNIQUES_PATH))
    update_uniques_inner(uniques)
    check_repeated(uniques)
    save_ndjson(at(UNIQUES_PATH), uniques)
