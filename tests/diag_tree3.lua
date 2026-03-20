-- Diagnostic: check class mapping and node IDs > 65535

function GetVirtualScreenSize()
    return 1920, 1080
end

dofile("HeadlessWrapper.lua")

-- Just check tree data, no build needed
local tree = main:LoadTree("3_28")

io.stderr:write("\n=== Class mapping ===\n")
for classId, class in pairs(tree.classes) do
    io.stderr:write("classId=" .. tostring(classId) .. " name=" .. tostring(class.name) .. " startNodeId=" .. tostring(class.startNodeId) .. "\n")
    if class.classes then
        for ascId, asc in pairs(class.classes) do
            io.stderr:write("  ascId=" .. tostring(ascId) .. " name=" .. tostring(asc.name) .. "\n")
        end
    end
end

-- Check node ID range
io.stderr:write("\n=== Node ID range ===\n")
local maxId = 0
local minId = 999999
local count = 0
local over65k = 0
for id, _ in pairs(tree.nodes) do
    if id > maxId then maxId = id end
    if id < minId then minId = id end
    count = count + 1
    if id >= 65536 then over65k = over65k + 1 end
end
io.stderr:write("Total tree nodes: " .. count .. "\n")
io.stderr:write("Min ID: " .. minId .. " Max ID: " .. maxId .. "\n")
io.stderr:write("Nodes with ID >= 65536: " .. over65k .. "\n")

-- Check the specific missing IDs
local missingIds = {65770, 65762, 66704, 65683, 65706, 65760, 66714, 66768, 66706, 65689, 66771, 65698, 66777, 65680, 65696}
io.stderr:write("\n=== Missing node lookup ===\n")
for _, id in ipairs(missingIds) do
    local node = tree.nodes[id]
    if node then
        io.stderr:write("ID " .. id .. " FOUND: " .. tostring(node.dn or node.name) .. " type=" .. tostring(node.type) .. " group=" .. tostring(node.group) .. "\n")
    else
        io.stderr:write("ID " .. id .. " NOT FOUND in tree.nodes\n")
    end
end
