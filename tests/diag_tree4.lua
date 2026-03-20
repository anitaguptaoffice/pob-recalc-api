-- Diagnostic: deep check of cluster jewel graph and allocNodes

function GetVirtualScreenSize()
    return 1920, 1080
end

dofile("HeadlessWrapper.lua")

local f = io.open("/data/workspace/pob-recalc-api/builds/酋长火刀阵.xml", "r")
local xml = f:read("*a")
f:close()

loadBuildFromXML(xml, "test")

if build and build.spec then
    io.stderr:write("\n=== After loadBuildFromXML ===\n")
    local allocCount = 0
    for _ in pairs(build.spec.allocNodes) do allocCount = allocCount + 1 end
    io.stderr:write("allocNodes: " .. allocCount .. "\n")
    io.stderr:write("curClassId: " .. tostring(build.spec.curClassId) .. "\n")
    io.stderr:write("curClassName: " .. tostring(build.spec.curClassName) .. "\n")

    -- Check allocSubgraphNodes
    io.stderr:write("allocSubgraphNodes: " .. #build.spec.allocSubgraphNodes .. "\n")
    for i, id in ipairs(build.spec.allocSubgraphNodes) do
        if i <= 20 then
            io.stderr:write("  subgraph node: " .. id .. "\n")
        end
    end

    -- Check jewels
    io.stderr:write("\nJewels:\n")
    for nodeId, itemId in pairs(build.spec.jewels) do
        io.stderr:write("  socket " .. nodeId .. " = item " .. tostring(itemId) .. "\n")
    end

    -- Check subGraphs
    io.stderr:write("\nSubgraphs: " .. (build.spec.subGraphs and "exists" or "nil") .. "\n")
    if build.spec.subGraphs then
        local sgCount = 0
        for _ in pairs(build.spec.subGraphs) do sgCount = sgCount + 1 end
        io.stderr:write("subGraph count: " .. sgCount .. "\n")
    end

    -- Now try running more OnFrame calls
    io.stderr:write("\n=== Running additional OnFrame calls ===\n")
    for i = 1, 5 do
        runCallback("OnFrame")
    end
    
    allocCount = 0
    for _ in pairs(build.spec.allocNodes) do allocCount = allocCount + 1 end
    io.stderr:write("allocNodes after 5 more frames: " .. allocCount .. "\n")
    io.stderr:write("curClassId: " .. tostring(build.spec.curClassId) .. "\n")
    io.stderr:write("curClassName: " .. tostring(build.spec.curClassName) .. "\n")
    
    -- Check calcsTab
    if build.calcsTab then
        if build.calcsTab.mainOutput then
            io.stderr:write("CombinedDPS: " .. tostring(build.calcsTab.mainOutput.CombinedDPS) .. "\n")
            io.stderr:write("Life: " .. tostring(build.calcsTab.mainOutput.Life) .. "\n")
            io.stderr:write("Str: " .. tostring(build.calcsTab.mainOutput.Str) .. "\n")
        else
            io.stderr:write("mainOutput is nil\n")
        end
    end
end
