# HeadlessWrapper 关键 Bug 修复记录

## 问题概述

POB Recalc API 的 headless 模式计算结果严重偏差：CombinedDPS 仅为 2500 万，而正确值应为 1.52 亿。天赋树未正确加载，className 回退为默认的 Scion/None，allocNodes 仅 1 个（应为 128 个）。

## 根因分析

`HeadlessWrapper.lua` 中有三个关键函数是空壳实现，导致了一系列连锁故障：

### Bug 1：Inflate / Deflate 返回空字符串

**文件**：`PathOfBuilding/src/HeadlessWrapper.lua`

**原始代码**：
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

**影响**：

POB 的 Timeless Jewel（永恒珠宝）数据以 zlib 压缩格式存储在 `.zip` 文件中（如 `ElegantHubris.zip`，2.49 MB）。`DataLegionLookUpTableHelper.lua` 在加载时调用 `Inflate()` 解压数据。空壳实现返回 `""`，导致：

1. Timeless Jewel 查找表加载失败
2. 天赋树节点解析不完整
3. `build.spec` 中的 `curClassName` / `curAscendClassName` 回退为默认值（Scion / None）
4. `allocNodes` 仅包含起始节点（1 个），而非实际分配的 128 个节点
5. 所有依赖天赋树的计算全部失效，DPS、Life、ES 等数值严重偏低

### Bug 2：GetScriptPath 返回空字符串

**原始代码**：
```lua
function GetScriptPath()
    return ""
end
```

**影响**：

`DataLegionLookUpTableHelper.lua` 第 10 行定义 `jewelTypeName = "/Data/TimelessJewelData/" .. jewelTypeName`（注意开头的 `/`），然后用 `GetScriptPath() .. jewelTypeName .. ".zip"` 拼接文件路径。

- `GetScriptPath()` 返回 `""` → 拼接结果为 `"/Data/TimelessJewelData/ElegantHubris.zip"` — 这是一个**绝对路径**
- 工作目录是 `/data/workspace/PathOfBuilding/src/`，文件实际位于 `./Data/TimelessJewelData/ElegantHubris.zip`
- 绝对路径 `/Data/TimelessJewelData/ElegantHubris.zip` 在文件系统中不存在 → `io.open` 失败

### Bug 3：MakeDir 为空操作

**原始代码**：
```lua
function MakeDir(path) end
```

**影响**：

`DataLegionLookUpTableHelper.lua` 在解压 `.zip` 后会将数据缓存为 `.bin` 文件，写入前需要 `MakeDir` 创建目录。空操作导致缓存写入失败（非致命，但每次启动都要重新解压）。

## 修复方案

### 修复 1：用 LuaJIT FFI + 系统 libz 实现 Inflate / Deflate

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

**技术说明**：
- POB 的压缩数据使用标准 zlib 格式（魔数 `78 da`，windowBits=15），与 Go 端 `compress/zlib` 一致
- 系统已安装 `libz.so`，`ffi.load("z")` 可直接加载
- `Inflate` 采用渐进式缓冲区策略（4x → 256x），处理不同压缩比的数据

### 修复 2：GetScriptPath / GetRuntimePath 返回 `"."`

```lua
function GetScriptPath()
    return "."
end
function GetRuntimePath()
    return "."
end
```

**说明**：返回 `"."` 使路径拼接为 `"./Data/TimelessJewelData/..."` — 相对路径，从工作目录正确解析。

### 修复 3：MakeDir 实际创建目录

```lua
function MakeDir(path)
    os.execute("mkdir -p " .. path)
end
```

### 附带清理：移除 worker.lua 中的 workaround

`generateRecalcXML` 中原有一段强制从原始 XML 保留 `className` / `ascendClassName` 的 workaround 代码，这是之前为了掩盖天赋树加载失败的症状而加的。根因修复后，`build:Save()` 输出的值本来就是正确的，该 workaround 已移除。

### 修复 4：ComposeXML 调用参数错误

`generateRecalcXML` 中 `common.xml.ComposeXML(dbXML)` 传入了 `ParseXML` 返回的包装数组，而 `ComposeXML` 需要的是带 `.elem` 属性的根节点。

**问题**：`ParseXML` 返回的结构为 `dbXML = { [1] = { elem="PathOfBuilding", ... } }`，其中 `dbXML` 本身没有 `.elem`，真正的根节点是 `dbXML[1]`。

**修复**：
```lua
-- 修复前（错误）
local xmlText, composeErr = common.xml.ComposeXML(dbXML)

-- 修复后（正确）
local xmlText, composeErr = common.xml.ComposeXML(dbXML[1])
```

**文件**：`pob-recalc-api/worker.lua`

## 验证结果

### 单元测试验证（本地环境）

使用「酋长火刀阵」build 进行 recalc 前后对比，96 项 PlayerStat **全部一致**：

| 指标 | 修复前 | 修复后 | 原始 XML |
|------|--------|--------|----------|
| className | Scion ❌ | Marauder ✅ | Marauder |
| ascendClassName | None ❌ | Chieftain ✅ | Chieftain |
| allocNodes | 1 ❌ | 128 ✅ | 128 |
| CombinedDPS | 25,158,002 ❌ | 152,140,053 ✅ | 152,140,053 |
| TotalDPS | — | 136,707,712 ✅ | 136,707,712 |
| Life | 1,571 ❌ | 4,173 ✅ | 4,173 |
| TotalEHP | — | 196,033 ✅ | 196,033 |

> **注意**：以上验证在本地直接运行（`go run main.go` + 本地修改后的 `HeadlessWrapper.lua`）时进行，结果正确。

### Docker 镜像验证（未修复前）

Docker 镜像通过 `git clone` 从 PathOfBuilding 上游仓库克隆源码，使用的是**原版未修复的 `HeadlessWrapper.lua`**。在未打补丁的 Docker 容器中，日志明确显示 bug：

```
Failed to load /Data/TimelessJewelData/ElegantHubris.bin, or data is out of date, falling back to compressed file
Failed to load either file: /Data/TimelessJewelData/ElegantHubris.zip, /Data/TimelessJewelData/ElegantHubris.bin
Stats: DPS=22599531, Life=1571, ES=68
```

| 指标 | Docker API（未修复）| 正确值 |
|------|---------------------|--------|
| TotalDPS | 22,599,531 ❌ | 136,707,712 |
| Life | 1,571 ❌ | 4,173 |
| CombinedDPS | ~25,000,000 ❌ | 152,140,053 |

### Docker 镜像热修方案

由于修复涉及 `HeadlessWrapper.lua`（属于上游 PathOfBuilding 仓库），且上游尚未合并这些修复，Dockerfile 中通过 `builds/patch-headless.sh` 脚本在构建时自动打补丁：

```dockerfile
# Hotfix: Patch HeadlessWrapper.lua to fix Inflate/Deflate, GetScriptPath,
# GetRuntimePath, and MakeDir stubs that break Timeless Jewel data loading.
COPY builds/patch-headless.sh /tmp/patch-headless.sh
RUN cd /app/src && sh /tmp/patch-headless.sh HeadlessWrapper.lua && rm /tmp/patch-headless.sh
```

补丁脚本对原版 `HeadlessWrapper.lua` 进行以下替换：
1. `Deflate`/`Inflate` 空壳 → LuaJIT FFI + 系统 libz 实现
2. `GetScriptPath()`/`GetRuntimePath()` 返回 `""` → 返回 `"."`
3. `MakeDir(path)` 空操作 → `os.execute("mkdir -p " .. path)`

重新构建镜像后需再次验证 API 计算结果是否正确。

## 涉及的文件

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `PathOfBuilding/src/HeadlessWrapper.lua` | 修改 | Inflate/Deflate 实现、GetScriptPath/GetRuntimePath 返回值、MakeDir 实现（仅本地环境） |
| `pob-recalc-api/worker.lua` | 修改 | 移除 className/ascendClassName workaround；修复 ComposeXML 调用参数（dbXML → dbXML[1]）；改进 pcall 错误信息捕获 |
| `pob-recalc-api/builds/patch-headless.sh` | **新增** | Docker 构建时自动修补 HeadlessWrapper.lua 的 shell 脚本 |
| `pob-recalc-api/Dockerfile` | 修改 | 添加 `zlib-dev` 依赖和热修补丁步骤 |

## 前置依赖

- 系统需安装 `libz.so`（zlib 库）：Docker 镜像中通过 `apk add zlib-dev` 安装
- LuaJIT 需支持 FFI（标准 LuaJIT 2.x 均支持）

## 待办

- [ ] 重新构建 Docker 镜像验证补丁生效：`docker build -t pob-recalc .`
- [ ] 如果上游 PathOfBuilding 合并了相关修复，可移除 `builds/patch-headless.sh` 和 Dockerfile 中的热修步骤
