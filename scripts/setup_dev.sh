#!/bin/bash
# Setup local development environment.
# Run this once after cloning the repo, or when you want to update cn-poe-utils.
#
# Usage:
#   bash scripts/setup_dev.sh          # clone + generate
#   bash scripts/setup_dev.sh update   # re-clone latest + regenerate

set -e

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CN_POE_DIR="$REPO_ROOT/cn-poe-utils"
CN_POE_REPO="https://github.com/cn-poe-community/cn-poe-utils.git"
CN_POE_BRANCH="main"

# --- Step 1: Clone or update cn-poe-utils ---
if [ "$1" = "update" ] || [ ! -d "$CN_POE_DIR/go" ]; then
    echo "=== Fetching cn-poe-utils (branch: $CN_POE_BRANCH) ==="
    rm -rf "$CN_POE_DIR"
    git clone --branch "$CN_POE_BRANCH" --depth 1 "$CN_POE_REPO" "$CN_POE_DIR"
    echo "cn-poe-utils: $(cd "$CN_POE_DIR" && git rev-parse --short HEAD) @ $(cd "$CN_POE_DIR" && git log -1 --format=%ci)"

    # Remove .git so VSCode doesn't treat it as a sub-repo
    rm -rf "$CN_POE_DIR/.git"

    # Fix go.mod if upstream requires newer Go version
    if grep -q "^go 1\.\(2[4-9]\|[3-9]\)" "$CN_POE_DIR/go/go.mod" 2>/dev/null; then
        echo "Fixing go.mod version..."
        sed -i 's/^go [0-9.]*/go 1.23/' "$CN_POE_DIR/go/go.mod"
    fi
else
    echo "=== cn-poe-utils already exists (use 'update' to refresh) ==="
fi

# --- Step 2: Generate translator/all.json ---
echo "=== Generating translator/all.json ==="
cd "$REPO_ROOT"
python3 scripts/gen_all_json.py
cp translate_data/all.json translator/all.json
echo "Done: translator/all.json"

# --- Step 3: Verify ---
echo "=== Verifying build + tests ==="
go build ./...
go test ./translator/... -count=1
go test ./pricer/... -count=1
echo ""
echo "=== Setup complete ==="
