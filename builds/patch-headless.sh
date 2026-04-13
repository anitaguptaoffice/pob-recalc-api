#!/bin/sh
#
# patch-headless.sh — Patches HeadlessWrapper.lua at Docker build time
#
# The upstream HeadlessWrapper.lua has several stub functions that break
# Timeless Jewel data loading and path resolution in headless/API mode:
#   - Inflate/Deflate return "" → zlib data can't be decompressed
#   - GetScriptPath/GetRuntimePath return "" → file paths resolve as absolute
#   - MakeDir is a no-op → cache directories can't be created
#
# This script patches those functions in-place using sed.
# See docs/headless-wrapper-bugfix.md for full details.
#

set -e

TARGET="${1:-HeadlessWrapper.lua}"

if [ ! -f "$TARGET" ]; then
    echo "[patch-headless] ERROR: $TARGET not found" >&2
    exit 1
fi

echo "[patch-headless] Patching $TARGET ..." >&2

# --- Patch 1: Replace Inflate/Deflate stubs with FFI zlib implementation ---
# We replace the entire Deflate+Inflate block with a do...end block using FFI.
# Use a temp file approach for multi-line replacement.

cat > /tmp/new_inflate_deflate.lua << 'ENDOFPATCH'
do
	local ffi = require("ffi")
	ffi.cdef[[
		unsigned long compressBound(unsigned long sourceLen);
		int compress(uint8_t *dest, unsigned long *destLen,
		             const uint8_t *source, unsigned long sourceLen);
		int uncompress(uint8_t *dest, unsigned long *destLen,
		               const uint8_t *source, unsigned long sourceLen);
	]]
	local zlib = ffi.load("z")

	function Deflate(data)
		if not data or #data == 0 then return "" end
		local sourceLen = #data
		local destLen = ffi.new("unsigned long[1]", zlib.compressBound(sourceLen))
		local dest = ffi.new("uint8_t[?]", destLen[0])
		local ret = zlib.compress(dest, destLen, data, sourceLen)
		if ret ~= 0 then return nil end
		return ffi.string(dest, destLen[0])
	end

	function Inflate(data)
		if not data or #data == 0 then return nil end
		local sourceLen = #data
		for mult = 4, 256, 4 do
			local destLen = ffi.new("unsigned long[1]", sourceLen * mult)
			local dest = ffi.new("uint8_t[?]", destLen[0])
			local ret = zlib.uncompress(dest, destLen, data, sourceLen)
			if ret == 0 then return ffi.string(dest, destLen[0])
			elseif ret ~= -5 then return nil end
		end
		return nil
	end
end
ENDOFPATCH

# Use awk for multi-line Deflate/Inflate replacement
awk '
/^function Deflate\(data\)/ { skip=1; next }
/^function Inflate\(data\)/ { skip=1; next }
skip && /^end$/ {
    skip=0
    # Only print replacement after the Inflate end
    if (!printed_inflate) {
        while ((getline line < "/tmp/new_inflate_deflate.lua") > 0) print line
        close("/tmp/new_inflate_deflate.lua")
        printed_inflate=1
    }
    next
}
skip { next }
{ print }
' "$TARGET" > "${TARGET}.tmp" && mv "${TARGET}.tmp" "$TARGET"

echo "[patch-headless] Patched Inflate/Deflate with FFI zlib" >&2

# --- Patch 2: GetScriptPath returns "." instead of "" ---
sed -i 's/^function GetScriptPath()$/function GetScriptPath()/' "$TARGET"
sed -i '/^function GetScriptPath()$/{n;s/return ""/return "."/}' "$TARGET"
echo "[patch-headless] Patched GetScriptPath" >&2

# --- Patch 3: GetRuntimePath returns "." instead of "" ---
sed -i '/^function GetRuntimePath()$/{n;s/return ""/return "."/}' "$TARGET"
echo "[patch-headless] Patched GetRuntimePath" >&2

# --- Patch 4: MakeDir actually creates directories ---
sed -i 's/^function MakeDir(path) end$/function MakeDir(path)\n\tos.execute("mkdir -p " .. path)\nend/' "$TARGET"
echo "[patch-headless] Patched MakeDir" >&2

# --- Patch 5: Merge split .zip.part* files into single .zip ---
# New POB versions split large TimelessJewelData files (e.g. GloriousVanity.zip)
# into .zip.part0, .zip.part1, ... for git-friendly storage.
# The code uses NewFileSearch() (which is a stub in headless mode) to iterate parts.
# Instead of implementing NewFileSearch, we simply concatenate the parts back into
# a single .zip at patch time. The old io.open() path then works as-is.

JEWEL_DIR="$(dirname "$TARGET")/Data/TimelessJewelData"
if [ -d "$JEWEL_DIR" ]; then
    for base in "$JEWEL_DIR"/*.zip.part0; do
        [ -f "$base" ] || continue
        zipname="${base%.part0}"
        echo "[patch-headless] Merging split parts → $(basename "$zipname")" >&2
        cat "$zipname".part* > "$zipname"
        rm -f "$zipname".part*
    done
fi
echo "[patch-headless] Patched TimelessJewelData split files" >&2

# --- Verify patches ---
ERRORS=0

if grep -q 'TODO: Might need this' "$TARGET"; then
    echo "[patch-headless] ERROR: Deflate stub still present!" >&2
    ERRORS=$((ERRORS+1))
fi

if grep -q 'TODO: And this' "$TARGET"; then
    echo "[patch-headless] ERROR: Inflate stub still present!" >&2
    ERRORS=$((ERRORS+1))
fi

# Check GetScriptPath returns "."
if grep -A1 'function GetScriptPath' "$TARGET" | grep -q 'return ""'; then
    echo "[patch-headless] ERROR: GetScriptPath still returns empty string!" >&2
    ERRORS=$((ERRORS+1))
fi

if [ $ERRORS -gt 0 ]; then
    echo "[patch-headless] FAILED: $ERRORS patches did not apply correctly" >&2
    exit 1
fi

echo "[patch-headless] All patches applied successfully to $TARGET" >&2
