import json
from typing import Any

from common import at, must_parent
from make import pob, poe


TS_POB_DATA_PATH = "../src/data/pob/data.ts"
TS_POE_DATA_PATH = "../src/data/poe/data.ts"

GO_POB_DATA_PATH = "../go/data/pob/data.go"
GO_POE_DATA_PATH = "../go/data/poe/testdata/all.json"


def json_to_js(data, name: str) -> str:
    """
    将JSON文件转换为JavaScript代码。

    :param data: 数据
    :param name: 变量名
    """
    return f"export const {name} = {json.dumps(data, ensure_ascii=False, indent=2)};"


def pob_make_for_ts(all: dict[str, Any]):

    codes = []
    for name, data in all.items():
        codes.append(json_to_js(data, name))

    must_parent(at(TS_POB_DATA_PATH))
    print(f"saved {at(TS_POB_DATA_PATH)}")
    with open(at(TS_POB_DATA_PATH), 'wt', encoding="utf-8", newline="\n") as f:
        f.write("\n".join(codes))


def poe_make_for_ts(all: dict[str, list]):
    codes = [json_to_js(data, name) for name, data in all.items()]

    must_parent(at(TS_POE_DATA_PATH))
    print(f"saved {at(TS_POE_DATA_PATH)}")
    with open(at(TS_POE_DATA_PATH), 'wt', encoding="utf-8", newline="\n") as f:
        f.write("\n".join(codes))


def pob_make_for_go(pob_all: dict[str, Any], poe_all: dict[str, list]):
    must_parent(at(GO_POB_DATA_PATH))
    print(f"saved {at(GO_POB_DATA_PATH)}")

    # Go版本的pob数据需要包含poe的transfiguredSkills数据
    pob_all["transfiguredSkills"] = poe_all["transfiguredSkills"]

    with open(at(GO_POB_DATA_PATH), 'wt', encoding="utf-8", newline="\n") as f:
        f.write("package pob\n\n")
        f.write("const dataStr = `")
        json.dump(pob_all, f, ensure_ascii=False, separators=(',', ':'))
        f.write("`")

    del pob_all["transfiguredSkills"]


def poe_make_for_go(all: dict[str, list]):
    must_parent(at(GO_POE_DATA_PATH))
    print(f"saved {at(GO_POE_DATA_PATH)}")
    with open(at(GO_POE_DATA_PATH), 'wt', encoding="utf-8", newline="\n") as f:
        json.dump(all, f, ensure_ascii=False, indent=2)


def make():
    print("info: making...")
    pob_all = pob.get_all()
    poe_all = poe.get_all()

    pob_make_for_ts(pob_all)
    poe_make_for_ts(poe_all)

    pob_make_for_go(pob_all, poe_all)
    poe_make_for_go(poe_all)
