-- Diagnostic: check if tree loads correctly in headless mode

function GetVirtualScreenSize()
    return 1920, 1080
end

dofile("HeadlessWrapper.lua")
io.stderr:write("=== HeadlessWrapper loaded ===\n")

local f = io.open("/data/workspace/pob-recalc-api/builds/酋长火刀阵.xml", "r")
local xml = f:read("*a")
f:close()
io.stderr:write("XML length: " .. #xml .. "\n")

loadBuildFromXML(xml, "test")
io.stderr:write("=== Build loaded ===\n")

if build and build.spec then
    io.stderr:write("treeVersion: " .. tostring(build.spec.treeVersion) .. "\n")
    io.stderr:write("curClassId: " .. tostring(build.spec.curClassId) .. "\n")
    io.stderr:write("curClassName: " .. tostring(build.spec.curClassName) .. "\n")
    io.stderr:write("curAscendClassId: " .. tostring(build.spec.curAscendClassId) .. "\n")
    io.stderr:write("curAscendClassName: " .. tostring(build.spec.curAscendClassName) .. "\n")
    local count = 0
    for _ in pairs(build.spec.allocNodes) do count = count + 1 end
    io.stderr:write("allocNodes count: " .. count .. "\n")

    -- Check tree.nodes count
    local treeNodeCount = 0
    for _ in pairs(build.spec.tree.nodes) do treeNodeCount = treeNodeCount + 1 end
    io.stderr:write("tree.nodes count: " .. treeNodeCount .. "\n")
    
    -- Check if spec.nodes were populated
    local specNodeCount = 0
    for _ in pairs(build.spec.nodes) do specNodeCount = specNodeCount + 1 end
    io.stderr:write("spec.nodes count: " .. specNodeCount .. "\n")
else
    io.stderr:write("build.spec is nil!\n")
end

if build and build.calcsTab and build.calcsTab.mainOutput then
    local out = build.calcsTab.mainOutput
    io.stderr:write("CombinedDPS: " .. tostring(out.CombinedDPS) .. "\n")
    io.stderr:write("TotalDPS: " .. tostring(out.TotalDPS) .. "\n")
    io.stderr:write("Life: " .. tostring(out.Life) .. "\n")
    io.stderr:write("Str: " .. tostring(out.Str) .. "\n")
    io.stderr:write("EnergyShield: " .. tostring(out.EnergyShield) .. "\n")
else
    io.stderr:write("calcsTab.mainOutput is nil!\n")
    if build and build.calcsTab then
        io.stderr:write("calcsTab exists but mainOutput is nil\n")
        io.stderr:write("mainEnv: " .. tostring(build.calcsTab.mainEnv) .. "\n")
    end
end
