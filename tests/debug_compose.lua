-- Debug XML compose error
dofile("HeadlessWrapper.lua")

local f = io.open("/data/workspace/pob-recalc-api/builds/酋长火刀阵.xml", "r")
local xmlText = f:read("*a")
f:close()

loadBuildFromXML(xmlText, "test")
for i = 1, 10 do
    if not build.buildFlag then break end
    runCallback("OnFrame")
end

local newBuildNode = { elem = "Build" }
build:Save(newBuildNode)

-- Inspect all children
print("attrib type: " .. type(newBuildNode.attrib))
print("children count: " .. #newBuildNode)
for i, child in ipairs(newBuildNode) do
    if type(child) == "table" then
        if not child.elem then
            print("  [" .. i .. "] MISSING elem! keys:")
            for k, v in pairs(child) do
                print("    " .. tostring(k) .. " = " .. tostring(v))
            end
        end
    elseif type(child) == "string" then
        print("  [" .. i .. "] STRING: " .. child:sub(1,50))
    else
        print("  [" .. i .. "] type=" .. type(child))
    end
end
