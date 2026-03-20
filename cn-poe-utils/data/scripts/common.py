import json
import os
from pathlib import Path
from typing import Any

import ndjson

from config import DATA_ROOT, POB_PATH, PROJECT_ROOT

SERVER_GLOBAL = "global"
SERVER_TENCENT = "tencent"

CLIENT_GLOBAL = "global"
CLIENT_TENCENT = "tencent"

LANG_CHS = "Simplified Chinese"
LANG_EN = "English"


def must_parent(path: str | Path) -> None:
    '''确保文件所在路径存在, 常用于写入文件时，避免父目录不存在而抛出异常'''
    Path(path).parent.mkdir(parents=True, exist_ok=True)


def read_json(file: str | Path) -> Any:
    '''读取json文件'''
    with open(file, 'rt', encoding='utf-8') as f:
        return json.load(f)


def save_json(file: str, data) -> None:
    '''保存json文件'''
    must_parent(file)
    with open(file, 'wt', encoding='utf-8') as f:
        json.dump(data, f, ensure_ascii=False, indent=2)
        f.write('\n')


def read_ndjson(file: str) -> Any:
    '''读取ndjson文件'''
    with open(file, 'rt', encoding='utf-8') as f:
        return ndjson.load(f)


def is_any_json(file: str):
    return os.path.isfile(file) and (file.endswith(".json") or file.endswith(".ndjson"))


def read_any_json(file: str):
    '''读取json文件或者ndjson文件'''
    if file.endswith(".ndjson"):
        return read_ndjson(file)
    elif file.endswith(".json"):
        return read_json(file)
    else:
        raise Exception(f"unsupported file: {file}")


def save_ndjson(file: str, data: list) -> None:
    '''保存ndjson文件'''
    must_parent(file)
    with open(file, 'wt', encoding='utf-8') as f:
        ndjson.dump(data, f, ensure_ascii=False)
        f.write('\n')


def at(*paths: str) -> str:
    '''获取相对于DATA目录的路径，支持多个路径参数'''
    return os.path.join(DATA_ROOT, *paths)


def at_proj(*paths: str) -> str:
    '''获取相对于本项目根目录的路径，支持多个路径参数'''
    return os.path.join(PROJECT_ROOT, *paths)


def at_pob(*paths: str) -> str:
    '''获取相对于POB根目录的路径，支持多个路径参数'''
    return os.path.join(POB_PATH, *paths)


def is_number(s):
    """检查字符串是否是数字"""
    try:
        float(s)
        return True
    except ValueError:
        return False


def sql_escape(s: str) -> str:
    """SQL字符串转义"""
    return s.replace("'", "''")
