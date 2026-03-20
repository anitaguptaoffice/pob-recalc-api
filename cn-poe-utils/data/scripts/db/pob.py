import json
import os
import subprocess
from common import at
from config import POB_PATH


def latest_tree_version() -> str:
    wd = at("scripts/luajit")
    env = os.environ.copy()
    env["LUA_PATH"] = f"{POB_PATH}/?.lua;{POB_PATH}/lua/?.lua;"

    result = subprocess.run(["luajit/luajit.exe", "-e", "local m = require 'GameVersions'; print(latestTreeVersion)"],
                            capture_output=True,
                            text=True,
                            cwd=wd,
                            env=env)
    if result.stderr.strip() != "":
        raise Exception(f"POB: 获取最新天赋树版本失败 {result.stderr.strip()}")
    return result.stdout.strip()


def get_tree(version: str):
    wd = at("scripts/luajit")
    env = os.environ.copy()
    env["LUA_PATH"] = f"{POB_PATH}/lua/?.lua;{POB_PATH}/TreeData/{version}/?.lua;"

    result = subprocess.run(["luajit/luajit.exe", "-e", "local j = require 'dkjson';local t = require 'tree'; print(j.encode(t))"],
                            capture_output=True,
                            text=True,
                            cwd=wd,
                            env=env)
    if result.stderr.strip() != "":
        raise Exception(f"POB: 获取最新天赋树失败: {result.stderr.strip()}")
    return json.loads(result.stdout.strip())


def get_cluster_jewel_metadata():
    wd = at("scripts/luajit")
    env = os.environ.copy()
    env["LUA_PATH"] = f"{POB_PATH}/lua/?.lua;{POB_PATH}/Data/?.lua;"

    result = subprocess.run(["luajit/luajit.exe", "-e", "local j = require 'dkjson';local t = require 'ClusterJewels'; print(j.encode(t))"],
                            capture_output=True,
                            text=True,
                            cwd=wd,
                            env=env)
    if result.stderr.strip() != "":
        raise Exception(f"POB: 获取星团珠宝元数据失败: {result.stderr.strip()}")
    return json.loads(result.stdout.strip())


def local2json(code: str, variable: str):
    """将Lua变量转为JSON。

    :param code: Lua代码
    :param variable: 变量名
    """
    wd = at("scripts/luajit")
    env = os.environ.copy()
    env["LUA_PATH"] = f"{POB_PATH}/lua/?.lua;"

    result = subprocess.run(["luajit/luajit.exe", "-e", f"{code}; local j = require 'dkjson'; print(j.encode({variable}))"],
                            capture_output=True,
                            text=True,
                            cwd=wd,
                            env=env)
    if result.stderr.strip() != "":
        raise Exception(f"将Lua变量转化为JSON失败: {result.stderr.strip()}")
    return json.loads(result.stdout.strip())
