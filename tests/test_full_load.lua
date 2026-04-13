-- Full tree loading diagnostic after Inflate/Deflate fix
dofile("HeadlessWrapper.lua")

-- Load the test build
local f = io.open("../tests/testdata/fixture.xml", "r")
local xml = f:read("*a")
f:close()
print("XML length: " .. #xml)

loadBuildFromXML(xml, "test")
print("=== Build loaded ===")

if build and build.spec then
    print("treeVersion: " .. tostring(build.spec.treeVersion))
    print("curClassId: " .. tostring(build.spec.curClassId))
    print("curClassName: " .. tostring(build.spec.curClassName))
    print("curAscendClassId: " .. tostring(build.spec.curAscendClassId))
    print("curAscendClassName: " .. tostring(build.spec.curAscendClassName))
    local count = 0
    for _ in pairs(build.spec.allocNodes) do count = count + 1 end
    print("allocNodes count: " .. count)
    
    -- Check subgraph (cluster jewels)
    local subGraphCount = 0
    if build.spec.subGraphs then
        for _ in pairs(build.spec.subGraphs) do subGraphCount = subGraphCount + 1 end
    end
    print("subGraph count: " .. subGraphCount)
else
    print("build.spec is nil!")
end

if build and build.calcsTab and build.calcsTab.mainOutput then
    local out = build.calcsTab.mainOutput
    print("\n=== Calculation Results ===")
    print("CombinedDPS: " .. tostring(out.CombinedDPS))
    print("TotalDPS: " .. tostring(out.TotalDPS))
    print("Life: " .. tostring(out.Life))
    print("Str: " .. tostring(out.Str))
    print("Dex: " .. tostring(out.Dex))
    print("Int: " .. tostring(out.Int))
else
    print("calcsTab.mainOutput not available yet - running extra frames...")
    for i = 1, 10 do
        runCallback("OnFrame")
    end
    if build and build.calcsTab and build.calcsTab.mainOutput then
        local out = build.calcsTab.mainOutput
        print("\n=== Calculation Results (after extra frames) ===")
        print("CombinedDPS: " .. tostring(out.CombinedDPS))
        print("TotalDPS: " .. tostring(out.TotalDPS))
        print("Life: " .. tostring(out.Life))
        print("Str: " .. tostring(out.Str))
    else
        print("Still no mainOutput after extra frames!")
    end
end
