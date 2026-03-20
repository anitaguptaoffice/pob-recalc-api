-- Diagnostic: Trace TreeTab:Load behavior

function GetVirtualScreenSize()
    return 1920, 1080
end

dofile("HeadlessWrapper.lua")

-- Monkey-patch TreeTabClass:Load to trace
local origTreeTabLoad = TreeTabClass.Load
TreeTabClass.Load = function(self, xml, dbFileName)
    io.stderr:write("\n=== TreeTabClass:Load called ===\n")
    io.stderr:write("xml.elem: " .. tostring(xml.elem) .. "\n")
    io.stderr:write("xml children count: " .. #xml .. "\n")
    for i, child in ipairs(xml) do
        if type(child) == "table" then
            io.stderr:write("  child[" .. i .. "]: elem=" .. tostring(child.elem) .. "\n")
        end
    end
    
    local result = origTreeTabLoad(self, xml, dbFileName)
    
    io.stderr:write("After TreeTabLoad: specList count=" .. #self.specList .. "\n")
    io.stderr:write("After TreeTabLoad: activeSpec=" .. tostring(self.activeSpec) .. "\n")
    for i, spec in ipairs(self.specList) do
        local ac = 0
        for _ in pairs(spec.allocNodes) do ac = ac + 1 end
        io.stderr:write("  spec[" .. i .. "]: classId=" .. tostring(spec.curClassId) .. " className=" .. tostring(spec.curClassName) .. " allocNodes=" .. ac .. "\n")
    end
    return result
end

-- Also patch PassiveSpecClass:Load
local origSpecLoad = PassiveSpecClass.Load
PassiveSpecClass.Load = function(self, xml, dbFileName)
    io.stderr:write("\n=== PassiveSpecClass:Load called ===\n")
    io.stderr:write("treeVersion: " .. tostring(self.treeVersion) .. "\n")
    io.stderr:write("nodes attrib: " .. tostring(xml.attrib.nodes and #xml.attrib.nodes or "nil") .. "\n")
    io.stderr:write("classId attrib: " .. tostring(xml.attrib.classId) .. "\n")
    
    local result = origSpecLoad(self, xml, dbFileName)
    
    local ac = 0
    for _ in pairs(self.allocNodes) do ac = ac + 1 end
    io.stderr:write("After SpecLoad: allocNodes=" .. ac .. "\n")
    io.stderr:write("After SpecLoad: curClassId=" .. tostring(self.curClassId) .. "\n")
    io.stderr:write("After SpecLoad: curClassName=" .. tostring(self.curClassName) .. "\n")
    io.stderr:write("After SpecLoad: returned " .. tostring(result) .. "\n")
    return result
end

local f = io.open("/data/workspace/pob-recalc-api/builds/酋长火刀阵.xml", "r")
local xml = f:read("*a")
f:close()

loadBuildFromXML(xml, "test")

io.stderr:write("\n=== Final state ===\n")
if build and build.spec then
    local ac = 0
    for _ in pairs(build.spec.allocNodes) do ac = ac + 1 end
    io.stderr:write("allocNodes: " .. ac .. "\n")
    io.stderr:write("className: " .. tostring(build.spec.curClassName) .. "\n")
end
if build and build.treeTab then
    io.stderr:write("specList: " .. #build.treeTab.specList .. "\n")
end
