-- End-to-end recalc test: simulates the full RECALC pipeline
dofile("HeadlessWrapper.lua")

-- Load the test build
local f = io.open("/data/workspace/pob-recalc-api/builds/酋长火刀阵.xml", "r")
local xmlText = f:read("*a")
f:close()

print("=== Loading build ===")
loadBuildFromXML(xmlText, "test")

-- Run extra frames to ensure calculation is complete
for i = 1, 10 do
    if not build.buildFlag then break end
    runCallback("OnFrame")
end

print("Build loaded. spec exists: " .. tostring(build.spec ~= nil))
print("className: " .. tostring(build.spec and build.spec.curClassName))
print("ascendClassName: " .. tostring(build.spec and build.spec.curAscendClassName))
local allocCount = 0
if build.spec and build.spec.allocNodes then
    for _ in pairs(build.spec.allocNodes) do allocCount = allocCount + 1 end
end
print("allocNodes: " .. allocCount)

-- Check calcsTab output
if build.calcsTab and build.calcsTab.mainOutput then
    local out = build.calcsTab.mainOutput
    print("\nRecalculated CombinedDPS: " .. tostring(out.CombinedDPS))
    print("Recalculated Life: " .. tostring(out.Life))
end

-- Test generateRecalcXML
print("\n=== Testing generateRecalcXML ===")
local newBuildNode = { elem = "Build" }
build:Save(newBuildNode)

-- Check what className is in the saved Build node
print("Saved Build attribs:")
if newBuildNode.attrib then
    print("  className: " .. tostring(newBuildNode.attrib.className))
    print("  ascendClassName: " .. tostring(newBuildNode.attrib.ascendClassName))
end

-- Check PlayerStat values
local statCount = 0
for _, child in ipairs(newBuildNode) do
    if child.elem == "PlayerStat" and child.attrib then
        statCount = statCount + 1
        local name = child.attrib.stat
        local val = child.attrib.value
        if name == "CombinedDPS" or name == "Life" or name == "Str" then
            print("  " .. name .. " = " .. tostring(val))
        end
    end
end
print("Total PlayerStat count: " .. statCount)

-- Full XML compose test
local dbXML, errMsg = common.xml.ParseXML(xmlText)
if dbXML and dbXML[1] then
    -- Find and replace <Build> node
    for i, node in ipairs(dbXML[1]) do
        if type(node) == "table" and node.elem == "Build" then
            dbXML[1][i] = newBuildNode
            break
        end
    end
    local outXML, composeErr = common.xml.ComposeXML(dbXML)
    if outXML then
        print("\nOutput XML length: " .. #outXML)
        -- Check key values in output
        local checkClass = outXML:match('className="([^"]*)"')
        local checkAscend = outXML:match('ascendClassName="([^"]*)"')
        local checkDPS = outXML:match('stat="CombinedDPS" value="([^"]*)"')
        print("Output className: " .. tostring(checkClass))
        print("Output ascendClassName: " .. tostring(checkAscend))
        print("Output CombinedDPS: " .. tostring(checkDPS))
    else
        print("Compose error: " .. tostring(composeErr))
    end
end

print("\n=== TEST COMPLETE ===")
