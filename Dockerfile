# POB Recalc API Service
#
# Standalone project — completely independent of the PathOfBuilding repo.
# POB source code is fetched from upstream at build time via git clone.
# cn-poe-utils (translation data) is also fetched from upstream at build time.
# LuaJIT commit hash is auto-extracted from POB's own Dockerfile.
#
# Build:
#   docker build -t pob-api .
#
# Build with specific POB branch/tag/commit:
#   docker build -t pob-api --build-arg POB_BRANCH=dev .
#   docker build -t pob-api --build-arg POB_BRANCH=v2.40.0 .
#
# Build with specific cn-poe-utils branch/tag:
#   docker build -t pob-api --build-arg CN_POE_UTILS_BRANCH=main .
#
# Run:
#   docker run -p 8080:8080 -e POB_POOL_SIZE=4 pob-api
#
# Configuration (runtime env vars):
#   POB_LISTEN      - Listen address (default: :8080)
#   POB_POOL_SIZE   - Number of LuaJIT worker processes (default: 2)
#

# ============================================================
# Stage 1: Clone upstream PathOfBuilding & extract LuaJIT hash
# ============================================================
FROM alpine:3.20 AS pob-source

RUN apk add --no-cache git grep

ARG POB_REPO=https://github.com/PathOfBuildingCommunity/PathOfBuilding.git
ARG POB_BRANCH=dev

WORKDIR /pob

# Shallow clone for speed; fall back to full clone if POB_BRANCH is a commit hash
RUN git clone --branch "${POB_BRANCH}" --depth 1 "${POB_REPO}" . 2>/dev/null \
    || (git clone "${POB_REPO}" . && git checkout "${POB_BRANCH}")

# Extract LuaJIT commit hash from POB's own Dockerfile (always in sync)
RUN grep -oP 'git checkout \K[0-9a-f]{40}' Dockerfile > /tmp/luajit_commit \
    && echo "Auto-detected LuaJIT commit: $(cat /tmp/luajit_commit)"

# Save POB version info for runtime
RUN echo "pob_branch=${POB_BRANCH}" > /tmp/pob_version \
    && echo "pob_commit=$(git rev-parse --short HEAD)" >> /tmp/pob_version \
    && echo "pob_date=$(git log -1 --format=%ci)" >> /tmp/pob_version \
    && cat /tmp/pob_version

# ============================================================
# Stage 2: Build LuaJIT from source (commit auto-detected)
# ============================================================
FROM alpine:3.20 AS luajit-builder

RUN apk add --no-cache build-base git
COPY --from=pob-source /tmp/luajit_commit /tmp/luajit_commit

WORKDIR /opt
RUN LUAJIT_COMMIT=$(cat /tmp/luajit_commit) \
    && echo "Building LuaJIT @ ${LUAJIT_COMMIT}" \
    && git clone https://github.com/LuaJIT/LuaJIT \
    && cd LuaJIT \
    && git checkout "${LUAJIT_COMMIT}" \
    && make -j$(nproc) \
    && make install PREFIX=/usr/local

# ============================================================
# Stage 3: Build LuaRocks + install runtime Lua dependencies
# ============================================================
FROM alpine:3.20 AS luarocks-builder

RUN apk add --no-cache build-base curl unzip readline-dev openssl tar wget

# Install Lua 5.1 (needed by luarocks)
WORKDIR /opt
RUN wget -q https://www.lua.org/ftp/lua-5.1.5.tar.gz && tar xf lua-5.1.5.tar.gz \
    && cd lua-5.1.5 && make linux && make install

# Install LuaRocks
RUN wget -q https://luarocks.org/releases/luarocks-3.7.0.tar.gz && tar xf luarocks-3.7.0.tar.gz \
    && cd luarocks-3.7.0 && ./configure && make && make install

# Copy LuaJIT (needed to compile C modules)
COPY --from=luajit-builder /usr/local/include/luajit-2.1/ /usr/local/include/luajit-2.1/
COPY --from=luajit-builder /usr/local/lib/libluajit* /usr/local/lib/
COPY --from=luajit-builder /usr/local/bin/luajit* /usr/local/bin/

# Install runtime Lua packages
RUN luarocks install luautf8 0.1.6-1

# ============================================================
# Stage 4a: Clone cn-poe-utils from upstream + generate all.json
# ============================================================
FROM python:3.12-alpine AS cn-poe-utils-source

ARG CN_POE_UTILS_REPO=https://github.com/cn-poe-community/cn-poe-utils.git
ARG CN_POE_UTILS_BRANCH=main

RUN apk add --no-cache git

WORKDIR /cn-poe-utils
RUN git clone --branch "${CN_POE_UTILS_BRANCH}" --depth 1 "${CN_POE_UTILS_REPO}" . \
    && echo "cn-poe-utils $(git rev-parse --short HEAD) @ $(git log -1 --format=%ci)"

# Generate all.json from upstream data/db/
COPY scripts/gen_all_json.py /tmp/gen_all_json.py
RUN CN_POE_UTILS_DB=/cn-poe-utils/data/db \
    CN_POE_UTILS_OUT=/cn-poe-utils/all.json \
    python3 /tmp/gen_all_json.py

# ============================================================
# Stage 4b: Build Go API binary + run tests to catch breaking changes
# ============================================================
FROM golang:1.23-alpine AS go-builder

WORKDIR /build

# Copy Go source from this repo
COPY go.mod ./
COPY main.go ./
COPY translator/ ./translator/
COPY pricer/ ./pricer/

# Copy cn-poe-utils Go code from UPSTREAM (not vendored)
COPY --from=cn-poe-utils-source /cn-poe-utils/go/ ./cn-poe-utils/go/

# Copy generated all.json into translator/
COPY --from=cn-poe-utils-source /cn-poe-utils/all.json ./translator/all.json

# Fix go.mod version if upstream bumped it beyond our Go toolchain
RUN sed -i 's/^go [0-9.]*/go 1.23/' cn-poe-utils/go/go.mod

# Run tests FIRST — if cn-poe-utils has a breaking change, build stops here
RUN echo "=== Running tests to verify cn-poe-utils compatibility ===" \
    && go test ./translator/... -v -count=1 \
    && go test ./pricer/... -v -count=1 \
    && echo "=== All tests passed ==="

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o pob-api .

# ============================================================
# Stage 5: Minimal runtime image
# ============================================================
FROM alpine:3.20

RUN apk add --no-cache readline libgcc wget zlib-dev

# Copy LuaJIT
COPY --from=luajit-builder /usr/local/bin/luajit* /usr/local/bin/
COPY --from=luajit-builder /usr/local/lib/libluajit* /usr/local/lib/
COPY --from=luajit-builder /usr/local/share/luajit-2.1/ /usr/local/share/luajit-2.1/
# Create luajit symlink dynamically (version number varies by commit)
RUN LUAJIT_BIN=$(ls /usr/local/bin/luajit-2.1.* 2>/dev/null | head -1) \
    && if [ -n "$LUAJIT_BIN" ]; then ln -sf "$LUAJIT_BIN" /usr/local/bin/luajit; fi
RUN ldconfig /usr/local/lib 2>/dev/null || true

# Copy luautf8 compiled module
COPY --from=luarocks-builder /usr/local/lib/lua/ /usr/local/lib/lua/
COPY --from=luarocks-builder /usr/local/share/lua/ /usr/local/share/lua/

# Copy Go API server binary
COPY --from=go-builder /build/pob-api /usr/local/bin/pob-api

# Copy POB source from upstream (only what's needed at runtime)
WORKDIR /app
COPY --from=pob-source /pob/src/ ./src/
COPY --from=pob-source /pob/runtime/lua/ ./runtime/lua/

# Hotfix: Patch HeadlessWrapper.lua to fix Inflate/Deflate, GetScriptPath,
# GetRuntimePath, and MakeDir stubs that break Timeless Jewel data loading.
# See docs/headless-wrapper-bugfix.md for details.
COPY builds/patch-headless.sh /tmp/patch-headless.sh
RUN cd /app/src && sh /tmp/patch-headless.sh HeadlessWrapper.lua && rm /tmp/patch-headless.sh
# Optional: manifest.xml for POB version number in logs
COPY --from=pob-source /pob/manifest.xml ./src/manifest.xml
COPY --from=pob-source /tmp/pob_version ./pob_version

# Copy API worker script (this repo's code, not POB's)
COPY worker.lua ./worker.lua

# Environment
ENV HOME=/tmp
ENV POB_SRC_DIR=/app/src
ENV POB_WORKER_SCRIPT=../worker.lua
ENV LUA_PATH="../runtime/lua/?.lua;../runtime/lua/?/init.lua;;"
ENV LUA_CPATH="/usr/local/lib/lua/5.1/?.so;;"
ENV POB_LISTEN=:8080
ENV POB_POOL_SIZE=2

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=120s --retries=3 \
    CMD wget -q -O /dev/null http://localhost:8080/health || exit 1

ENTRYPOINT ["pob-api"]
