-- test_crash2.lua: Determine why first BuildOutput fails but second succeeds
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

dofile("HeadlessWrapper.lua")
ConPrintf = function(fmt, ...)
    io.stderr:write(string.format(fmt, ...) .. "\n")
end
io.stdout = _real_stdout
for i = 1, 20 do runCallback("OnFrame") end
io.stderr:write("[test] Init complete\n")

local xmlText = io.stdin:read("*a")
io.stderr:write("[test] Read " .. #xmlText .. " bytes\n")

-- Load with pcall and capture any error inside BuildOutput
local loadOk, loadErr = pcall(function()
    loadBuildFromXML(xmlText, "test_crash")
end)
io.stderr:write("[test] loadBuildFromXML pcall result: ok=" .. tostring(loadOk) .. " err=" .. tostring(loadErr) .. "\n")

-- Check state after load
io.stderr:write("[test] mainEnv after load = " .. tostring(build.calcsTab.mainEnv) .. "\n")
io.stderr:write("[test] buildFlag = " .. tostring(build.buildFlag) .. "\n")

-- Check if buildFlag is set (meaning it needs another OnFrame to compute)
if build.buildFlag then
    io.stderr:write("[test] buildFlag is TRUE — running OnFrame...\n")
    runCallback("OnFrame")
    io.stderr:write("[test] mainEnv after OnFrame = " .. tostring(build.calcsTab.mainEnv) .. "\n")
    io.stderr:write("[test] buildFlag after OnFrame = " .. tostring(build.buildFlag) .. "\n")
end

-- If still nil, try direct BuildOutput with pcall to get the actual error
if not build.calcsTab.mainEnv then
    io.stderr:write("[test] Still nil, calling BuildOutput directly with pcall...\n")
    local ok2, err2 = pcall(function()
        build.calcsTab:BuildOutput()
    end)
    io.stderr:write("[test] BuildOutput pcall: ok=" .. tostring(ok2) .. " err=" .. tostring(err2) .. "\n")
    io.stderr:write("[test] mainEnv after direct BuildOutput = " .. tostring(build.calcsTab.mainEnv) .. "\n")
end

-- Now check if mainEnv is populated and test SaveDB
if build.calcsTab.mainEnv then
    io.stderr:write("[test] mainEnv is now populated!\n")
    local saveOk, saveErr = pcall(function()
        local xml = build:SaveDB("test")
        io.stderr:write("[test] SaveDB produced " .. #xml .. " bytes\n")
    end)
    if not saveOk then
        io.stderr:write("[test] SaveDB ERROR: " .. tostring(saveErr) .. "\n")
    end
else
    io.stderr:write("[test] mainEnv is STILL nil after all attempts\n")
end
