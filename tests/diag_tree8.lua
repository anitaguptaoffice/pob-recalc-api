-- Diagnostic: Use instrumented HeadlessWrapper

function GetVirtualScreenSize()
    return 1920, 1080
end

dofile("HeadlessWrapper.lua")

-- The classes are loaded via the PoB module system, not globals
-- Let's instead directly test loadBuildFromXML with full tracing

-- Wrap the common build loading path
local f = io.open("/data/workspace/pob-recalc-api/builds/酋长火刀阵.xml", "r")
local xmlText = f:read("*a")
f:close()

-- Parse XML to check structure
local dbXML, errMsg = common.xml.ParseXML(xmlText)
io.stderr:write("XML parse error: " .. tostring(errMsg) .. "\n")

-- Now call loadBuildFromXML  
loadBuildFromXML(xmlText, "test")

-- Check xmlSectionList  
io.stderr:write("\n=== Build internals ===\n")
io.stderr:write("xmlSectionList: " .. type(build.xmlSectionList) .. "\n")
if build.xmlSectionList then
    io.stderr:write("xmlSectionList count: " .. #build.xmlSectionList .. "\n")
    for i, node in ipairs(build.xmlSectionList) do
        if type(node) == "table" then
            io.stderr:write("  [" .. i .. "] elem=" .. tostring(node.elem) .. "\n")
        end
    end
end

-- Check savers
io.stderr:write("\nsavers: " .. type(build.savers) .. "\n")
if build.savers then
    for k, v in pairs(build.savers) do
        io.stderr:write("  saver: " .. k .. " = " .. tostring(v) .. "\n")
    end
else
    io.stderr:write("savers is nil! (CloseBuild was called?)\n")
end

-- Check treeTab
io.stderr:write("\ntreeTab: " .. type(build.treeTab) .. "\n")
if build.treeTab then
    io.stderr:write("specList: " .. #build.treeTab.specList .. "\n")
    io.stderr:write("activeSpec: " .. tostring(build.treeTab.activeSpec) .. "\n")
end

-- Check spec
io.stderr:write("\nspec: " .. type(build.spec) .. "\n")
if build.spec then
    local ac = 0
    for _ in pairs(build.spec.allocNodes) do ac = ac + 1 end
    io.stderr:write("allocNodes: " .. ac .. "\n")
    io.stderr:write("curClassName: " .. tostring(build.spec.curClassName) .. "\n")
end
