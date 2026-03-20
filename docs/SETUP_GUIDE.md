# PoB MCP 环境搭建指南

> 本文档记录了 pob-mcp 和 PathOfBuilding 两个仓库相对于上游的差异，以及如何从零复现当前工作状态。

## 一、仓库概览

| 仓库 | 上游地址 | Fork 地址 | 本地路径 |
|------|---------|----------|---------|
| **pob-mcp** | - | `https://github.com/ianderse/pob-mcp.git` | `/data/workspace/pob-recalc-api/pob-mcp` |
| **PathOfBuilding** | `https://github.com/PathOfBuildingCommunity/PathOfBuilding.git` | `https://github.com/ianderse/PathOfBuilding.git` | `/root/test/PathOfBuilding` |

---

## 二、pob-mcp 仓库差异

### 相对上游：无源码变更

当前位于 `main` 分支最新提交 `7ecd92e fix merge conflicts`。

唯一的工作区变更是 `package-lock.json`，由 `npm install` 自动生成（将 `@types/node-fetch` 等几个包从 `dependencies` 调整到 `devDependencies`），**没有任何源代码改动**。

---

## 三、PathOfBuilding 仓库差异

### 3.1 分支结构

```
当前分支: api-stdio (基于 ianderse/PathOfBuilding 的 fork)

远程仓库:
  origin   → https://github.com/ianderse/PathOfBuilding.git
  upstream → https://github.com/PathOfBuildingCommunity/PathOfBuilding.git
```

### 3.2 已提交的变更（api-stdio 分支 vs upstream/dev）

`api-stdio` 分支在 `upstream/dev` 的基础上新增了 **11 个文件变更（+2295 行 / -1 行）**，核心是为 PoB 添加了一套 **headless stdio JSON-RPC API**：

#### 新增文件（6 个）

| 文件 | 行数 | 说明 |
|------|------|------|
| `src/API/Server.lua` | +87 | stdio JSON-RPC 主循环，读 stdin 写 stdout，分发 action 到 handler |
| `src/API/Handlers.lua` | +247 | 所有 API handler（get_stats、load_build、get_tree、set_tree、search_nodes 等） |
| `src/API/BuildOps.lua` | +733 | PoB build 操作的薄封装层（获取 stats、修改 tree、装备物品、技能宝石操作等） |
| `src/utf8.lua` | +28 | headless 模式的 UTF-8 最小桩实现 |
| `src/lua-utf8.lua` | +27 | 同上的备用路径 |
| `src/sha1.lua` | +92 | 纯 Lua SHA-1 实现（用于 build hash 等） |

#### 新增测试（2 个）

| 文件 | 说明 |
|------|------|
| `spec/API/Stdio_spec.lua` | stdio API 的集成测试 |
| `API_README.md` | API 使用文档（695 行） |

#### 修改文件（2 个）

| 文件 | 说明 |
|------|------|
| `src/HeadlessWrapper.lua` | **核心改动**（+130/-1）：添加了 `--stdio` / `POB_API_STDIO=1` 入口，自动初始化 build 系统后启动 API Server；新增 `get_script_dir()` 路径自动发现逻辑；设置 `package.path` 以支持外部 luajit 调用 |
| `src/Modules/Main.lua` | 仅格式调整（+1 空行） |

### 3.3 合并操作

在 `api-stdio` 分支上执行了一次 **merge upstream/dev**：

```
5853293c Merge remote-tracking branch 'upstream/dev' into api-stdio
```

这将 upstream 的最新 dev 内容（包括 v2.63.0 发布、#9648~#9663 等十几个 bugfix）合并到了 api-stdio 分支，涉及 **426 个文件、+763,592 / -82,093 行**。

主要引入的上游更新：
- 游戏数据导出更新（TreeData、技能数据等）
- Generals Cry、Animate Weapon、Molten Strike 等技能计算修复
- Lethal Dose、Black Scythe Training、Eternal Youth Life Recharge 等新特性支持
- Resistance Shrine、Balance of Terror、Perfidy 等物品/天赋修复
- CI/测试改进（busted 配置、docker 支持）

### 3.4 未提交的工作区变更

`src/HeadlessWrapper.lua` 有额外的未提交修改（+38/-10），实现了以下关键功能：

#### ① Deflate/Inflate 真实实现（替换空桩）

```lua
-- 之前（空桩）：
function Deflate(data) return "" end
function Inflate(data) return "" end

-- 现在（FFI 调用 zlib）：
do
    local ffi = require("ffi")
    ffi.cdef[[
        unsigned long compressBound(unsigned long sourceLen);
        int compress(uint8_t *dest, unsigned long *destLen,
                     const uint8_t *source, unsigned long sourceLen);
        int uncompress(uint8_t *dest, unsigned long *destLen,
                       const uint8_t *source, unsigned long sourceLen);
    ]]
    local zlib = ffi.load("z")

    function Deflate(data) ... end   -- 调用 zlib.compress
    function Inflate(data) ... end   -- 调用 zlib.uncompress，带自动缓冲区扩展
end
```

**意义**：没有这个实现，PoB 无法处理压缩的 build 数据（pastebin 导入/导出、build 编码等）。

#### ② 路径函数修复

```lua
-- 之前：
function GetScriptPath() return "" end
function GetRuntimePath() return "" end
function MakeDir(path) end

-- 现在：
function GetScriptPath() return "." end
function GetRuntimePath() return "." end
function MakeDir(path) os.execute("mkdir -p " .. path) end
```

---

## 四、从零复现步骤

### 4.1 系统依赖

```bash
# LuaJIT + zlib（Deflate/Inflate 需要）
apt-get update && apt-get install -y luajit libluajit-5.1-dev zlib1g-dev

# Node.js（pob-mcp 运行需要）
# 确保 node >= 18
node --version
```

### 4.2 克隆 PathOfBuilding fork 并切换到 api-stdio 分支

```bash
mkdir -p /root/test
cd /root/test

# 克隆 ianderse 的 fork
git clone https://github.com/ianderse/PathOfBuilding.git
cd PathOfBuilding

# 添加上游远程
git remote add upstream https://github.com/PathOfBuildingCommunity/PathOfBuilding.git
git fetch upstream

# 切换到 api-stdio 分支
git checkout api-stdio

# 合并上游最新 dev（获取最新的技能/物品计算修复）
git merge upstream/dev
```

### 4.3 应用 HeadlessWrapper.lua 的未提交修改

对 `src/HeadlessWrapper.lua` 做以下修改：

**① 替换 Deflate/Inflate 空桩为 zlib FFI 实现：**

找到：
```lua
function Deflate(data)
	-- TODO: Might need this
	return ""
end
function Inflate(data)
	-- TODO: And this
	return ""
end
```

替换为：
```lua
do
	local ffi = require("ffi")
	ffi.cdef[[
		unsigned long compressBound(unsigned long sourceLen);
		int compress(uint8_t *dest, unsigned long *destLen,
		             const uint8_t *source, unsigned long sourceLen);
		int uncompress(uint8_t *dest, unsigned long *destLen,
		               const uint8_t *source, unsigned long sourceLen);
	]]
	local zlib = ffi.load("z")

	function Deflate(data)
		if not data or #data == 0 then return "" end
		local sourceLen = #data
		local destLen = ffi.new("unsigned long[1]", zlib.compressBound(sourceLen))
		local dest = ffi.new("uint8_t[?]", destLen[0])
		local ret = zlib.compress(dest, destLen, data, sourceLen)
		if ret ~= 0 then return nil end
		return ffi.string(dest, destLen[0])
	end

	function Inflate(data)
		if not data or #data == 0 then return nil end
		local sourceLen = #data
		for mult = 4, 256, 4 do
			local destLen = ffi.new("unsigned long[1]", sourceLen * mult)
			local dest = ffi.new("uint8_t[?]", destLen[0])
			local ret = zlib.uncompress(dest, destLen, data, sourceLen)
			if ret == 0 then return ffi.string(dest, destLen[0])
			elseif ret ~= -5 then return nil end  -- -5 = Z_BUF_ERROR, retry
		end
		return nil
	end
end
```

**② 修复路径函数：**

```lua
-- GetScriptPath: "" → "."
function GetScriptPath()
	return "."
end

-- GetRuntimePath: "" → "."
function GetRuntimePath()
	return "."
end

-- MakeDir: 空函数 → 实际创建目录
function MakeDir(path)
	os.execute("mkdir -p " .. path)
end
```

### 4.4 克隆并构建 pob-mcp

```bash
cd /data/workspace/pob-recalc-api

# 克隆（如果还没有的话，它是 submodule 或独立目录）
git clone https://github.com/ianderse/pob-mcp.git pob-mcp
cd pob-mcp

# 安装依赖并编译
npm install
npm run build
```

### 4.5 准备 builds 目录

```bash
# 放置你的 PoB build XML 文件
mkdir -p /root/test/pob-mcp/builds
# 将 .xml build 文件复制到此目录
cp /path/to/your/builds/*.xml /root/test/pob-mcp/builds/
```

### 4.6 配置 CodeBuddy MCP

编辑 `/root/.codebuddy/mcp.json`：

```json
{
  "mcpServers": {
    "pob-mcp": {
      "command": "node",
      "args": ["/data/workspace/pob-recalc-api/pob-mcp/build/index.js"],
      "env": {
        "POB_DIRECTORY": "/root/test/pob-mcp/builds",
        "POB_LUA_ENABLED": "true",
        "POB_FORK_PATH": "/root/test/PathOfBuilding/src",
        "POB_CMD": "luajit"
      }
    }
  }
}
```

### 4.7 验证

```bash
# 测试 Lua Bridge 是否正常
cd /root/test/PathOfBuilding/src
echo '{"action":"get_version"}' | POB_API_STDIO=1 luajit HeadlessWrapper.lua --stdio

# 测试 MCP Server 是否正常
cd /data/workspace/pob-recalc-api/pob-mcp
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' | \
  POB_DIRECTORY=/root/test/pob-mcp/builds \
  POB_LUA_ENABLED=true \
  POB_FORK_PATH=/root/test/PathOfBuilding/src \
  POB_CMD=luajit \
  node build/index.js
```

---

## 五、架构说明

```
┌─────────────────────────────────────────────────────┐
│                   CodeBuddy IDE                      │
│                  (MCP Client)                        │
└──────────────────────┬──────────────────────────────┘
                       │ stdio (JSON-RPC over MCP)
                       ▼
┌─────────────────────────────────────────────────────┐
│              pob-mcp (Node.js)                       │
│         MCP Server - 第二层                          │
│  · XML 解析 (fast-xml-parser)                        │
│  · 工具路由 (80+ tools)                              │
│  · Build 分析/验证/优化                              │
└──────────────────────┬──────────────────────────────┘
                       │ stdio (JSON line protocol)
                       ▼
┌─────────────────────────────────────────────────────┐
│         PathOfBuilding (LuaJIT)                      │
│        Lua Bridge - 第一层                           │
│  · HeadlessWrapper.lua → API/Server.lua              │
│  · 真实 PoB 计算引擎 (DPS/EHP/Tree)                 │
│  · zlib Deflate/Inflate (FFI)                        │
└─────────────────────────────────────────────────────┘
```

**两层通信**：CodeBuddy ↔ pob-mcp（MCP 协议）↔ PathOfBuilding（JSON line 协议）
