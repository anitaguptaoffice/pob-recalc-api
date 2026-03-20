-- Test the actual generateRecalcXML function
dofile("HeadlessWrapper.lua")

local f = io.open("/data/workspace/pob-recalc-api/builds/酋长火刀阵.xml", "r")
local xmlText = f:read("*a")
f:close()

print("=== Loading build ===")
loadBuildFromXML(xmlText, "test")

-- Run extra frames for calculation
for i = 1, 10 do
    if not build.buildFlag then break end
    runCallback("OnFrame")
end

print("className: " .. tostring(build.spec.curClassName))
print("allocNodes: " .. (function() local c=0; for _ in pairs(build.spec.allocNodes) do c=c+1 end; return c end)())

-- Test generateRecalcXML directly (loaded from worker.lua functions)
local dbXML, errMsg = common.xml.ParseXML(xmlText)
local origBuildAttribs = {}
for _, node in ipairs(dbXML[1]) do
    if type(node) == "table" and node.elem == "Build" then
        origBuildAttribs = node.attrib or {}
        break
    end
end

local newBuildNode = { elem = "Build" }
build:Save(newBuildNode)
print("\nnewBuildNode.attrib.className: " .. tostring(newBuildNode.attrib and newBuildNode.attrib.className))
print("newBuildNode.attrib.ascendClassName: " .. tostring(newBuildNode.attrib and newBuildNode.attrib.ascendClassName))
print("newBuildNode children count: " .. #newBuildNode)

-- Replace in DOM
for i, node in ipairs(dbXML[1]) do
    if type(node) == "table" and node.elem == "Build" then
        dbXML[1][i] = newBuildNode
        break
    end
end

local outXML, composeErr = common.xml.ComposeXML(dbXML)
if outXML then
    print("\nOutput XML length: " .. #outXML)
    local checkClass = outXML:match('className="([^"]*)"')
    local checkAscend = outXML:match('ascendClassName="([^"]*)"')
    local checkDPS = outXML:match('stat="CombinedDPS" value="([^"]*)"')
    print("className: " .. tostring(checkClass))
    print("ascendClassName: " .. tostring(checkAscend))
    print("CombinedDPS: " .. tostring(checkDPS))

    -- Write output for comparison
    local of = io.open("/data/workspace/pob-recalc-api/builds/酋长火刀阵_fixed_output.xml", "w")
    of:write(outXML)
    of:close()
    print("\nOutput written to 酋长火刀阵_fixed_output.xml")
else
    print("Compose error: " .. tostring(composeErr))
end
