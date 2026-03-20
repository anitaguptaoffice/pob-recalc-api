dofile("HeadlessWrapper.lua")

print("=== Inflate/Deflate basic test ===")
local orig = "Hello World! This is a test of zlib compression in LuaJIT FFI. Repeated: AAAAAAAAAA"
local compressed = Deflate(orig)
print("Original length: " .. #orig)
print("Compressed length: " .. #compressed)
local decompressed = Inflate(compressed)
print("Decompressed length: " .. #decompressed)
print("Match: " .. tostring(orig == decompressed))

print("\n=== Timeless Jewel zip test ===")
local f = io.open("./Data/TimelessJewelData/ElegantHubris.zip", "rb")
if f then
    local data = f:read("*a")
    f:close()
    print("Zip file size: " .. #data)
    local inflated = Inflate(data)
    if inflated then
        print("Inflated size: " .. #inflated)
        print("SUCCESS: ElegantHubris.zip decompressed!")
    else
        print("ERROR: Inflate returned nil!")
    end
else
    print("ERROR: cannot open zip file at ./Data/TimelessJewelData/ElegantHubris.zip")
end
