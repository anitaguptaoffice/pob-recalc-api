-- Diagnostic: manually call TreeTab:Load with the parsed Tree XML node

function GetVirtualScreenSize()
    return 1920, 1080
end

dofile("HeadlessWrapper.lua")

local f = io.open("/data/workspace/pob-recalc-api/builds/酋长火刀阵.xml", "r")
local xmlText = f:read("*a")
f:close()

-- First load normally
loadBuildFromXML(xmlText, "test")

io.stderr:write("\n=== After initial load, specList=" .. #build.treeTab.specList .. " ===\n")

-- Parse XML again and find the Tree node
local dbXML = common.xml.ParseXML(xmlText)
local treeNode = nil
for _, node in ipairs(dbXML[1]) do
    if type(node) == "table" and node.elem == "Tree" then
        treeNode = node
        break
    end
end

if not treeNode then
    io.stderr:write("No Tree node found in XML!\n")
    return
end

io.stderr:write("Tree node found, children: " .. #treeNode .. "\n")

-- Manually call treeTab:Load and check
io.stderr:write("\n=== Manually calling treeTab:Load ===\n")
local err = build.treeTab:Load(treeNode, "test.xml")
io.stderr:write("Load returned: " .. tostring(err) .. "\n")
io.stderr:write("specList after manual load: " .. #build.treeTab.specList .. "\n")

for i, spec in ipairs(build.treeTab.specList) do
    local ac = 0
    for _ in pairs(spec.allocNodes) do ac = ac + 1 end
    io.stderr:write("  spec[" .. i .. "]: treeVersion=" .. tostring(spec.treeVersion) .. " classId=" .. tostring(spec.curClassId) .. " className=" .. tostring(spec.curClassName) .. " allocNodes=" .. ac .. "\n")
end

-- Check build.spec after manual load
if build.spec then
    local ac = 0
    for _ in pairs(build.spec.allocNodes) do ac = ac + 1 end
    io.stderr:write("\nbuild.spec: className=" .. tostring(build.spec.curClassName) .. " allocNodes=" .. ac .. "\n")
end
