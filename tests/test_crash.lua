-- test_crash.lua: Precisely capture why buildOutput fails for a given build
-- Usage: cd /app/src && cat build.xml | luajit ../tests/test_crash.lua

local _real_stdout = io.stdout
print = function(...)
    local args = {...}
    for i = 1, select("#", ...) do
        if i > 1 then io.stderr:write("\t") end
        io.stderr:write(tostring(args[i]))
    end
    io.stderr:write("\n")
end

function GetVirtualScreenSize() return 1920, 1080 end

io.stderr:write("[test] Loading HeadlessWrapper...\n")
dofile("HeadlessWrapper.lua")

ConPrintf = function(fmt, ...)
    io.stderr:write(string.format(fmt, ...) .. "\n")
end
io.stdout = _real_stdout

for i = 1, 20 do runCallback("OnFrame") end
io.stderr:write("[test] Init complete\n")

-- Read the XML from stdin
local xmlText = io.stdin:read("*a")
io.stderr:write("[test] Read " .. #xmlText .. " bytes of XML\n")

-- Try loading with pcall
local ok, errMsg = pcall(function()
    loadBuildFromXML(xmlText, "test_crash")
end)

if not ok then
    io.stderr:write("[test] LOAD ERROR: " .. tostring(errMsg) .. "\n")
    os.exit(1)
end

io.stderr:write("[test] LOAD OK\n")

if not build or not build.calcsTab then
    io.stderr:write("[test] build or calcsTab not available!\n")
    os.exit(1)
end

io.stderr:write("[test] mainEnv = " .. tostring(build.calcsTab.mainEnv) .. "\n")
io.stderr:write("[test] mainOutput = " .. tostring(build.calcsTab.mainOutput) .. "\n")

if build.calcsTab.mainEnv then
    io.stderr:write("[test] mainEnv.player = " .. tostring(build.calcsTab.mainEnv.player) .. "\n")
    if build.calcsTab.mainEnv.player then
        io.stderr:write("[test] mainSkill = " .. tostring(build.calcsTab.mainEnv.player.mainSkill) .. "\n")
        if build.calcsTab.mainEnv.player.mainSkill then
            io.stderr:write("[test] skillFlags = " .. tostring(build.calcsTab.mainEnv.player.mainSkill.skillFlags) .. "\n")
        end
    end
    io.stderr:write("[test] SUCCESS - mainEnv is populated\n")
else
    io.stderr:write("[test] FAILURE - mainEnv is NIL!\n")
    io.stderr:write("[test] Attempting manual BuildOutput to capture the real error...\n")
    local ok2, err2 = pcall(function()
        build.calcsTab:BuildOutput()
    end)
    if not ok2 then
        io.stderr:write("[test] BuildOutput ERROR: " .. tostring(err2) .. "\n")
    else
        io.stderr:write("[test] Manual BuildOutput OK\n")
        io.stderr:write("[test] mainEnv after manual = " .. tostring(build.calcsTab.mainEnv) .. "\n")
    end
end
