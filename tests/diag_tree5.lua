-- Diagnostic: trace ImportFromNodeList in detail

function GetVirtualScreenSize()
    return 1920, 1080
end

dofile("HeadlessWrapper.lua")

-- Monkey-patch ImportFromNodeList to trace behavior
local origImport = PassiveSpecClass.ImportFromNodeList
function PassiveSpecClass:ImportFromNodeList(classId, ascendClassId, secondaryAscendClassId, hashList, hashOverrides, masteryEffects, treeVersion)
    io.stderr:write("\n=== ImportFromNodeList called ===\n")
    io.stderr:write("classId=" .. tostring(classId) .. "\n")
    io.stderr:write("ascendClassId=" .. tostring(ascendClassId) .. "\n")
    io.stderr:write("hashList count: " .. #hashList .. "\n")
    io.stderr:write("spec.nodes count before: ")
    local c = 0
    for _ in pairs(self.nodes) do c = c + 1 end
    io.stderr:write(c .. "\n")
    
    -- Call original
    origImport(self, classId, ascendClassId, secondaryAscendClassId, hashList, hashOverrides, masteryEffects, treeVersion)
    
    -- Check result
    local allocCount = 0
    for _ in pairs(self.allocNodes) do allocCount = allocCount + 1 end
    io.stderr:write("allocNodes after import: " .. allocCount .. "\n")
    io.stderr:write("curClassId after import: " .. tostring(self.curClassId) .. "\n")
    io.stderr:write("curClassName after import: " .. tostring(self.curClassName) .. "\n")
end

local f = io.open("/data/workspace/pob-recalc-api/builds/酋长火刀阵.xml", "r")
local xml = f:read("*a")
f:close()

loadBuildFromXML(xml, "test")

if build and build.spec then
    local allocCount = 0
    for _ in pairs(build.spec.allocNodes) do allocCount = allocCount + 1 end
    io.stderr:write("\nFinal allocNodes: " .. allocCount .. "\n")
    io.stderr:write("Final curClassName: " .. tostring(build.spec.curClassName) .. "\n")
end
