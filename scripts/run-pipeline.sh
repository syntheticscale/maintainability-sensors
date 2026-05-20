#!/usr/bin/env bash
# ==============================================================================
# Maintainability Sensors: Verification & Report Generation Pipeline 📡
# ==============================================================================
# This script automates downloading, caching, and scanning of our 5 canonical
# reference repositories. It generates beautiful, standalone Markdown reports
# for each repo inside the 'dist/reports/' directory.
#
# Can be run locally (caching to '.cache/') or inside GitHub Actions CI.
# ==============================================================================

set -euo pipefail

# Ensure user local bin folders and local venv are in PATH (for isolated pip tools like pylint)
export PATH="./.venv/bin:$HOME/.local/bin:$HOME/.hermes/home/.local/bin:/usr/local/bin:$PATH"

# Configuration
CACHE_DIR=".cache"
OUTPUT_DIR="dist/reports"
BINARY_PATH="bin/maintainability-sensors"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Ensure directories exist
mkdir -p "$CACHE_DIR"
mkdir -p "$OUTPUT_DIR"

# Step 1: Ensure static binary is built
log_info "Ensuring maintainability-sensors CLI is compiled..."
if [ ! -f "$BINARY_PATH" ]; then
    log_warn "CLI binary not found. Building now..."
    /usr/local/go/bin/go build -o "$BINARY_PATH" main.go
fi
log_success "CLI binary verified: $($BINARY_PATH --help | head -n 1)"

# Helper function to clone or update a repository
sync_repo() {
    local name=$1
    local url=$2
    local target_dir="$CACHE_DIR/$name"

    if [ -d "$target_dir" ]; then
        log_info "Cache hit for '$name'. Pulling latest changes..."
        git -C "$target_dir" pull --ff-only || log_warn "Failed to pull latest for $name, using cached version."
    else
        log_info "Cache miss for '$name'. Cloning depth=1..."
        git clone --depth 1 "$url" "$target_dir"
    fi
}

# Step 2: Download/Sync canonical repositories
log_info "Syncing canonical validation repositories..."
sync_repo "go-chi" "https://github.com/go-chi/chi.git"
sync_repo "requests" "https://github.com/psf/requests.git"
sync_repo "go-std-net" "https://github.com/golang/go.git"
sync_repo "fastapi" "https://github.com/tiangolo/fastapi.git"
sync_repo "nestjs" "https://github.com/nestjs/nest.git"
log_success "All validation repositories successfully synchronized."

# Step 3: Run pipeline analysis
analyze_repo() {
    local name=$1
    local scan_path=$2
    local report_name=$3
    local repo_root="$CACHE_DIR/$name"
    local output_file="$OUTPUT_DIR/$report_name"

    log_info "--------------------------------------------------------"
    log_info "Processing: $name"
    log_info "--------------------------------------------------------"

    # Bootstrap the repository context to Level 1+ analysis boundaries
    log_info "Bootstrapping maintainability configurations inside '$name'..."
    $BINARY_PATH bootstrap "$repo_root"

    # For Python (requests, fastapi): ensure pylint is available via a local venv
    if [[ "$name" == "requests" || "$name" == "fastapi" ]]; then
        if ! command -v pylint &>/dev/null; then
            log_warn "pylint not detected. Installing into a local virtual environment to enable Level 1+ orchestrated analysis..."
            if [ ! -d ".venv" ]; then
                python3 -m venv .venv
            fi
            .venv/bin/pip install --quiet pylint || log_warn "Failed to install pylint. Run will fall back to Level 0."
        fi
    fi

    # For TypeScript/JavaScript (nestjs): we run npm install to ensure local eslint is available
    if [[ "$name" == "nestjs" ]]; then
        log_info "Ensuring local NestJS ESLint dependencies are installed..."
        if [ ! -d "$repo_root/node_modules" ]; then
            (cd "$repo_root" && npm install --quiet --legacy-peer-deps) || log_warn "Failed to install NestJS npm dependencies. Run will fall back to Level 0."
        else
            log_info "NestJS node_modules already cached."
        fi
    fi

    # Run the sensors and output the report in three standard formats (MD, JSON, HTML)
    log_info "Running maintainability-sensors on '$scan_path'..."
    set +e # Don't exit on linter warning exits
    $BINARY_PATH run \
        --markdown-out="${output_file}.md" \
        --json-out="${output_file}.json" \
        --html-out="${output_file}.html" \
        "$CACHE_DIR/$name/$scan_path"
    set -e

    log_success "Generated beautiful scorecards (MD, JSON, HTML) for: $name"
}

# Execute analysis on our 5 Case Studies
analyze_repo "go-chi" "tree.go" "go-chi-tree-report"
analyze_repo "requests" "src/requests/adapters.py" "requests-adapters-report"
analyze_repo "go-std-net" "src/net/http/server.go" "go-std-http-server-report"
analyze_repo "fastapi" "fastapi/dependencies/utils.py" "fastapi-dependencies-report"
analyze_repo "nestjs" "packages/core/injector/injector.ts" "nestjs-injector-report"

log_info "========================================================"
log_success "All pipeline analysis completed successfully!"
log_info "Reports generated in: $OUTPUT_DIR/"
ls -la "$OUTPUT_DIR"
log_info "========================================================"
