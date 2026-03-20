-- Diagnostic: check node ID matching

function GetVirtualScreenSize()
    return 1920, 1080
end

dofile("HeadlessWrapper.lua")

local f = io.open("/data/workspace/pob-recalc-api/builds/酋长火刀阵.xml", "r")
local xml = f:read("*a")
f:close()

loadBuildFromXML(xml, "test")

if build and build.spec then
    -- XML 中的节点 ID 列表
    local xmlNodes = "18901,31683,44908,32245,24970,26712,16775,49178,62319,23027,63976,53042,42659,36859,65770,6289,63282,46578,47175,32249,65762,13714,55190,66704,34171,65683,65706,43902,44191,1731,65760,66714,66768,14993,66706,50862,34400,14930,46910,26196,19144,49080,33287,65689,66771,62429,4139,65698,9971,17527,26866,29993,10851,5430,26725,50422,2913,6204,55485,61308,48929,33508,9511,55676,26523,30380,3644,60472,5233,29353,10532,34513,5643,66777,9995,33718,48423,44941,65034,23881,2491,48813,6712,59928,22480,64406,6967,42792,10221,36881,9402,46519,33631,65680,6230,25989,24704,36949,41415,48267,53118,28475,16167,48287,6446,50570,63723,35503,21650,65696,30733,22893,44169,61666,44202,861,44903,15405,25168,33740,31628,9920,28330,35288,64128,27166,48480,24472"

    local found = 0
    local missing = 0
    local missingIds = {}
    local foundIds = {}
    for id in xmlNodes:gmatch("%d+") do
        local nodeId = tonumber(id)
        if build.spec.nodes[nodeId] then
            found = found + 1
            table.insert(foundIds, nodeId)
        else
            missing = missing + 1
            table.insert(missingIds, nodeId)
        end
    end
    io.stderr:write("Nodes found in spec.nodes: " .. found .. "\n")
    io.stderr:write("Nodes missing from spec.nodes: " .. missing .. "\n")
    if missing > 0 then
        io.stderr:write("Missing IDs (first 20): ")
        for i = 1, math.min(20, #missingIds) do
            io.stderr:write(missingIds[i] .. " ")
        end
        io.stderr:write("\n")
    end
    
    -- Check if tree.nodes has these
    local treeFound = 0
    local treeMissing = 0
    local treeMissingIds = {}
    for id in xmlNodes:gmatch("%d+") do
        local nodeId = tonumber(id)
        if build.spec.tree.nodes[nodeId] then
            treeFound = treeFound + 1
        else
            treeMissing = treeMissing + 1
            table.insert(treeMissingIds, nodeId)
        end
    end
    io.stderr:write("Nodes found in tree.nodes: " .. treeFound .. "\n")
    io.stderr:write("Nodes missing from tree.nodes: " .. treeMissing .. "\n")
    if treeMissing > 0 then
        io.stderr:write("Tree missing IDs (first 20): ")
        for i = 1, math.min(20, #treeMissingIds) do
            io.stderr:write(treeMissingIds[i] .. " ")
        end
        io.stderr:write("\n")
    end

    -- Check allocNodes detail
    io.stderr:write("\nallocNodes:\n")
    for id, node in pairs(build.spec.allocNodes) do
        io.stderr:write("  " .. id .. ": " .. tostring(node.name) .. " type=" .. tostring(node.type) .. "\n")
    end
end
