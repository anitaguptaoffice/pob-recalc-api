-- Diagnostic: test XML parsing of tree section

function GetVirtualScreenSize()
    return 1920, 1080
end

dofile("HeadlessWrapper.lua")

local f = io.open("/data/workspace/pob-recalc-api/builds/酋长火刀阵.xml", "r")
local xml = f:read("*a")
f:close()

-- Parse XML directly
local dbXML, errMsg = common.xml.ParseXML(xml)
if errMsg then
    io.stderr:write("Parse error: " .. errMsg .. "\n")
    return
end

-- Find Tree node
for _, node in ipairs(dbXML[1]) do
    if type(node) == "table" then
        io.stderr:write("Top-level elem: " .. tostring(node.elem) .. "\n")
        if node.elem == "Tree" then
            io.stderr:write("  Tree attrib.activeSpec: " .. tostring(node.attrib.activeSpec) .. "\n")
            io.stderr:write("  Tree children count: " .. #node .. "\n")
            for i, child in ipairs(node) do
                if type(child) == "table" then
                    io.stderr:write("  Tree child[" .. i .. "]: elem=" .. tostring(child.elem) .. "\n")
                    if child.elem == "Spec" then
                        io.stderr:write("    treeVersion=" .. tostring(child.attrib.treeVersion) .. "\n")
                        io.stderr:write("    classId=" .. tostring(child.attrib.classId) .. "\n")
                        io.stderr:write("    nodes length=" .. tostring(child.attrib.nodes and #child.attrib.nodes or "nil") .. "\n")
                    end
                else
                    io.stderr:write("  Tree child[" .. i .. "]: type=" .. type(child) .. " = " .. tostring(child):sub(1,50) .. "\n")
                end
            end
        end
    end
end
