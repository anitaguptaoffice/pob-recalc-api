-- test_recalc_diff.lua
-- Load XML build, recalc, compare before/after PlayerStat values

local _real_stdout = io.stdout
print = function(...)
	local args = {...}
	for i = 1, select("#", ...) do
		if i > 1 then io.stderr:write("\t") end
		io.stderr:write(tostring(args[i]))
	end
	io.stderr:write("\n")
end

function GetVirtualScreenSize()
	return 1920, 1080
end

io.stderr:write("[test] Loading HeadlessWrapper...\n")
dofile("HeadlessWrapper.lua")
ConPrintf = function(fmt, ...)
	io.stderr:write(string.format(fmt, ...) .. "\n")
end
io.stdout = _real_stdout

for i = 1, 20 do runCallback("OnFrame") end
io.stderr:write("[test] HeadlessWrapper loaded.\n")

-- Read pre-decoded XML
local xmlFile = "/tmp/酋长火刀阵.xml"
local f = io.open(xmlFile, "r")
if not f then error("cannot open " .. xmlFile) end
local xmlText = f:read("*a")
f:close()
io.stderr:write("[test] Input XML: " .. #xmlText .. " bytes\n")

-- Extract Build stats from XML
local function extractBuildStats(xml)
	local dbXML, err = common.xml.ParseXML(xml)
	if err or not dbXML or not dbXML[1] then return nil, tostring(err) end
	for _, node in ipairs(dbXML[1]) do
		if type(node) == "table" and node.elem == "Build" then
			local stats = { _attribs = node.attrib or {} }
			for _, child in ipairs(node) do
				if type(child) == "table" and child.elem == "PlayerStat" then
					local a = child.attrib or {}
					if a.stat and a.value then
						stats[a.stat] = tonumber(a.value) or a.value
					end
				end
			end
			return stats
		end
	end
	return nil, "No <Build> node"
end

local beforeStats = assert(extractBuildStats(xmlText))
io.stderr:write("[test] Before: className=" .. tostring(beforeStats._attribs.className)
	.. " ascendClassName=" .. tostring(beforeStats._attribs.ascendClassName) .. "\n")

-- Load build in POB
io.stderr:write("[test] Loading build into POB...\n")
loadBuildFromXML(xmlText, "test_recalc")

if build.buildFlag then
	for i = 1, 10 do
		runCallback("OnFrame")
		if not build.buildFlag then break end
	end
end
if not build.calcsTab.mainEnv then
	build.calcsTab:BuildOutput()
end

-- Tree diagnostic
if build.spec then
	local ac = 0
	if build.spec.allocNodes then for _ in pairs(build.spec.allocNodes) do ac = ac + 1 end end
	io.stderr:write(string.format("[test] Tree: class=%s ascend=%s allocNodes=%d\n",
		tostring(build.spec.curClassName), tostring(build.spec.curAscendClassName), ac))
end

-- Generate recalc: build:Save into newBuildNode, extract stats directly
local newBuildNode = { elem = "Build" }
build:Save(newBuildNode)

local afterStats = { _attribs = newBuildNode.attrib or {} }
for _, child in ipairs(newBuildNode) do
	if type(child) == "table" and child.elem == "PlayerStat" then
		local a = child.attrib or {}
		if a.stat and a.value then
			afterStats[a.stat] = tonumber(a.value) or a.value
		end
	end
end

io.stderr:write("[test] After: className=" .. tostring(afterStats._attribs.className)
	.. " ascendClassName=" .. tostring(afterStats._attribs.ascendClassName) .. "\n")

-- Compare
io.stderr:write("\n========== Build Attributes ==========\n")
for _, k in ipairs({"className", "ascendClassName", "level", "mainSocketGroup", "bandit"}) do
	local bv = beforeStats._attribs[k] or "(nil)"
	local av = afterStats._attribs[k] or "(nil)"
	local mark = (bv == av) and "OK" or "*** DIFF ***"
	io.stderr:write(string.format("  %-25s  before=%-20s  after=%-20s  %s\n", k, bv, av, mark))
end

io.stderr:write("\n========== PlayerStat Comparison ==========\n")
local allKeys, keySet = {}, {}
for k in pairs(beforeStats) do if k ~= "_attribs" then allKeys[#allKeys+1] = k; keySet[k] = true end end
for k in pairs(afterStats) do if k ~= "_attribs" and not keySet[k] then allKeys[#allKeys+1] = k; keySet[k] = true end end
table.sort(allKeys)

local sigDiffs = 0
for _, k in ipairs(allKeys) do
	local bv = beforeStats[k] or 0
	local av = afterStats[k] or 0
	local diff, pct, mark = 0, "", "OK"
	if type(bv) == "number" and type(av) == "number" then
		diff = av - bv
		if bv ~= 0 then pct = string.format("(%+.2f%%)", (diff / math.abs(bv)) * 100) end
		if math.abs(diff) > 10 or (bv ~= 0 and math.abs(diff / bv) > 0.01) then
			mark = "*** DIFF ***"; sigDiffs = sigDiffs + 1
		end
	elseif tostring(bv) ~= tostring(av) then
		mark = "*** DIFF ***"; sigDiffs = sigDiffs + 1
	end
	io.stderr:write(string.format("  %-40s before=%-15s after=%-15s diff=%-12s %-12s %s\n",
		k,
		type(bv) == "number" and string.format("%.2f", bv) or tostring(bv),
		type(av) == "number" and string.format("%.2f", av) or tostring(av),
		type(diff) == "number" and string.format("%+.2f", diff) or "",
		pct, mark))
end

io.stderr:write(string.format("\n[test] Total stats: %d, Significant diffs: %d\n", #allKeys, sigDiffs))
if sigDiffs == 0 then
	io.stderr:write("[test] ✅ PASS — recalc output matches original XML!\n")
else
	io.stderr:write("[test] ⚠️  Some stats differ — review above.\n")
end
