-- Diagnostic: trace PassiveSpec Load and Import behavior

function GetVirtualScreenSize()
    return 1920, 1080
end

-- Patch HeadlessWrapper.lua's ConPrintf to actually print
local _origConPrintf
local function _setupTracePrint()
    _origConPrintf = ConPrintf
    ConPrintf = function(fmt, ...)
        io.stderr:write("[ConPrintf] " .. string.format(fmt, ...) .. "\n")
    end
end

dofile("HeadlessWrapper.lua")
_setupTracePrint()

-- Patch loadBuildFromXML to trace
local origLoadBuild = loadBuildFromXML

-- Instead, let's trace PassiveSpec:Load and ImportFromNodeList manually
-- by reading the build state at different points

local f = io.open("/data/workspace/pob-recalc-api/builds/酋长火刀阵.xml", "r")
local xml = f:read("*a")
f:close()

io.stderr:write("\n=== Before loadBuildFromXML ===\n")
io.stderr:write("build type: " .. type(build) .. "\n")

loadBuildFromXML(xml, "test")

io.stderr:write("\n=== After loadBuildFromXML ===\n")
io.stderr:write("build type: " .. type(build) .. "\n")

if build then
    io.stderr:write("build.spec type: " .. type(build.spec) .. "\n")
    if build.spec then
        io.stderr:write("treeVersion: " .. tostring(build.spec.treeVersion) .. "\n")
        io.stderr:write("curClassId: " .. tostring(build.spec.curClassId) .. "\n")
        io.stderr:write("curClassName: " .. tostring(build.spec.curClassName) .. "\n")
        
        local allocCount = 0
        for _ in pairs(build.spec.allocNodes) do allocCount = allocCount + 1 end
        io.stderr:write("allocNodes: " .. allocCount .. "\n")
    end
    
    -- Check treeTab
    io.stderr:write("treeTab type: " .. type(build.treeTab) .. "\n")
    if build.treeTab then
        io.stderr:write("treeTab.activeSpec: " .. tostring(build.treeTab.activeSpec) .. "\n")
        io.stderr:write("treeTab.specList count: " .. #build.treeTab.specList .. "\n")
        for i, spec in ipairs(build.treeTab.specList) do
            local ac = 0
            for _ in pairs(spec.allocNodes) do ac = ac + 1 end
            io.stderr:write("  spec[" .. i .. "]: treeVersion=" .. tostring(spec.treeVersion) .. " classId=" .. tostring(spec.curClassId) .. " className=" .. tostring(spec.curClassName) .. " allocNodes=" .. ac .. "\n")
        end
    end
    
    -- Check if buildFlag is set
    io.stderr:write("buildFlag: " .. tostring(build.buildFlag) .. "\n")
    
    -- Check promptMsg
    io.stderr:write("promptMsg: " .. tostring(mainObject.promptMsg) .. "\n")
end
