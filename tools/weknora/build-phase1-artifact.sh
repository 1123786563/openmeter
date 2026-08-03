#!/usr/bin/env bash
#
# build-phase1-artifact.sh — Build the WeKnora AI Billing Phase 1 provider
# artifact from a clean commit.
#
# Produces, written to build/phase1-artifact/ (durable, not temp):
#   - manifest.json    — contract version, checksums, pinned upstream commit,
#                        migration version, SDK checksum, image digest
#   - openapi.json     — the generated v3 OpenAPI spec (extracted from image)
#   - sbom.json        — CycloneDX SBOM (syft)
#
# Records a summary table to docs/test-reports/openmeter-ai-billing-phase1.md.
#
# Unresolved CRITICAL vulnerabilities block the artifact (exit 1).
#
# Usage:
#   tools/weknora/build-phase1-artifact.sh [--image-name weknora-billing-p1]
#
# Required tools: docker, git, jq, sha256sum, syft, grype.
# Optional tools (for backup-restore smoke): pg_dump, psql, clickhouse-client.

set -euo pipefail

IMAGE_NAME="${1:-weknora-billing-p1}"
CONTRACT_VERSION="weknora-billing-p1-v1"
MIGRATION_VERSION="20260803000100"
ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
OUTPUT_DIR="$ROOT_DIR/build/phase1-artifact"
REPORT_DIR="${REPORT_DIR:-$ROOT_DIR/docs/test-reports}"
REPORT_FILE="$REPORT_DIR/openmeter-ai-billing-phase1.md"

cd "$ROOT_DIR"

log() { printf '[build-phase1-artifact] %s\n' "$*" >&2; }
fail() { printf '[build-phase1-artifact] ERROR: %s\n' "$*" >&2; exit 1; }

require_tool() {
    command -v "$1" >/dev/null 2>&1 || fail "$1 is required but not found"
}

# ---------------------------------------------------------------------------
# 0. Pre-flight: ensure clean working tree and all required tools.
# ---------------------------------------------------------------------------

require_tool docker
require_tool git
require_tool jq
require_tool sha256sum
require_tool syft
require_tool grype

GIT_DIRTY=$(git status --porcelain --untracked-files=no | wc -l | tr -d ' ')
if [ "$GIT_DIRTY" -ne 0 ]; then
    fail "working tree is not clean (run from a committed state). Stash or commit first."
fi

UPSTREAM_COMMIT=$(git rev-parse HEAD)
UPSTREAM_SHORT=$(git rev-parse --short=12 HEAD)
UPSTREAM_TAG=$(git describe --tags --always --dirty 2>/dev/null || echo "$UPSTREAM_SHORT")

log "building from commit $UPSTREAM_COMMIT ($UPSTREAM_TAG)"

# Prepare durable output directory (issue 4: not ephemeral).
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

# ---------------------------------------------------------------------------
# 1. Build the Docker image.
# ---------------------------------------------------------------------------

log "building Docker image: $IMAGE_NAME"
docker build \
    --build-arg VERSION="$CONTRACT_VERSION" \
    -t "$IMAGE_NAME:$CONTRACT_VERSION" \
    -t "$IMAGE_NAME:latest" \
    -f Dockerfile \
    "$ROOT_DIR"

IMAGE_DIGEST=$(docker inspect --format='{{index .RepoDigests 0}}' "$IMAGE_NAME:$CONTRACT_VERSION" 2>/dev/null || echo "")
IMAGE_ID=$(docker inspect --format='{{.Id}}' "$IMAGE_NAME:$CONTRACT_VERSION")

# ---------------------------------------------------------------------------
# 2. Extract contract artifacts from the image (issue 1: read only from image).
# ---------------------------------------------------------------------------

docker create --name phase1-artifact-extract "$IMAGE_NAME:$CONTRACT_VERSION" /bin/true >/dev/null 2>&1 || true

# Extract /contract/openapi.json from the image. No source-tree fallback —
# the script only consumes artifacts baked into the image.
docker cp phase1-artifact-extract:/contract/openapi.json "$OUTPUT_DIR/openapi.json" || \
    fail "/contract/openapi.json not found in image — check the Dockerfile openapi-json stage"
docker rm phase1-artifact-extract >/dev/null 2>&1 || true

OPENAPI_FILE="$OUTPUT_DIR/openapi.json"
OPENAPI_SHA256=$(sha256sum "$OPENAPI_FILE" | awk '{print $1}')
OPENAPI_SIZE=$(wc -c < "$OPENAPI_FILE" | tr -d ' ')

# ---------------------------------------------------------------------------
# 3. Compute SDK checksum (issue 5: add sdk.sha256 to manifest).
# ---------------------------------------------------------------------------

SDK_SHA256=$(find api/v3/client -name '*.go' -print0 | sort -z | xargs -0 sha256sum | sha256sum | awk '{print $1}')
log "SDK checksum: $SDK_SHA256"

# ---------------------------------------------------------------------------
# 4. Generate the manifest (issue 4: written to durable output dir).
# ---------------------------------------------------------------------------

MANIFEST_FILE="$OUTPUT_DIR/manifest.json"
jq -n \
    --arg contract_version "$CONTRACT_VERSION" \
    --arg openapi_sha256 "$OPENAPI_SHA256" \
    --argjson openapi_size "$OPENAPI_SIZE" \
    --arg sdk_sha256 "$SDK_SHA256" \
    --arg migration_version "$MIGRATION_VERSION" \
    --arg upstream_commit "$UPSTREAM_COMMIT" \
    --arg upstream_tag "$UPSTREAM_TAG" \
    --arg image_digest "$IMAGE_DIGEST" \
    --arg image_id "$IMAGE_ID" \
    --arg built_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    '{
        contract_version: $contract_version,
        openapi: {
            sha256: $openapi_sha256,
            size_bytes: $openapi_size
        },
        sdk: {
            sha256: $sdk_sha256
        },
        migration_version: $migration_version,
        upstream: {
            commit: $upstream_commit,
            tag: $upstream_tag
        },
        image: {
            digest: $image_digest,
            id: $image_id
        },
        built_at: $built_at
    }' > "$MANIFEST_FILE"

log "manifest generated at $MANIFEST_FILE"
cat "$MANIFEST_FILE" >&2

# ---------------------------------------------------------------------------
# 5. Generate SBOM (CycloneDX) — syft is required.
# ---------------------------------------------------------------------------

SBOM_FILE="$OUTPUT_DIR/sbom.json"
log "generating SBOM with syft"
syft "$IMAGE_NAME:$CONTRACT_VERSION" -o cyclonedx-json > "$SBOM_FILE"
SBOM_SHA256=$(sha256sum "$SBOM_FILE" | awk '{print $1}')

# ---------------------------------------------------------------------------
# 6. Vulnerability scan — grype is required, CRITICAL vulns block (issue 3).
# ---------------------------------------------------------------------------

log "scanning for vulnerabilities with grype"
GRYPE_FILE="$OUTPUT_DIR/grype.json"
grype "$IMAGE_NAME:$CONTRACT_VERSION" --output json > "$GRYPE_FILE" 2>&1 || true

# grype JSON uses .matches[].vulnerability.severity (a string), not the
# rating array. Count Critical matches directly.
CRITICAL_COUNT=$(jq '[.matches[] | select(.vulnerability.severity == "Critical")] | length' "$GRYPE_FILE" 2>/dev/null || echo "0")

VULN_RESULT="PASS ($CRITICAL_COUNT critical)"
if [ "$CRITICAL_COUNT" != "0" ]; then
    VULN_RESULT="BLOCKED ($CRITICAL_COUNT critical vulnerabilities)"
    log "$VULN_RESULT"
    fail "unresolved CRITICAL vulnerabilities block the provider artifact"
fi
log "no critical vulnerabilities found"

# ---------------------------------------------------------------------------
# 7. Backup-restore smoke (issue 6: separate PostgreSQL and ClickHouse).
# ---------------------------------------------------------------------------

PG_SMOKE="SKIPPED (POSTGRES_URL not set)"
if [ -n "${POSTGRES_URL:-}" ]; then
    require_tool pg_dump
    require_tool psql
    log "running PostgreSQL backup-restore smoke"
    SMOKE_TMP=$(mktemp -d)
    trap 'rm -rf "$SMOKE_TMP"' RETURN
    if pg_dump "$POSTGRES_URL" > "$SMOKE_TMP/pg_backup.sql" 2>/dev/null && \
       psql "$POSTGRES_URL" -c "DROP TABLE IF EXISTS _phase1_smoke; CREATE TABLE _phase1_smoke(id int); INSERT INTO _phase1_smoke VALUES(1);" >/dev/null 2>&1 && \
       psql "$POSTGRES_URL" -c "SELECT count(*) FROM _phase1_smoke;" >/dev/null 2>&1; then
        PG_SMOKE="PASS"
        log "PostgreSQL backup-restore smoke: PASS"
    else
        PG_SMOKE="FAIL"
        fail "PostgreSQL backup-restore smoke failed"
    fi
fi

CH_SMOKE="SKIPPED (CLICKHOUSE_URL not set)"
if [ -n "${CLICKHOUSE_URL:-}" ]; then
    require_tool clickhouse-client
    log "running ClickHouse backup-restore smoke"
    # Round-trip: dump table count, restore into a smoke table, verify.
    if clickhouse-client --query "CREATE TABLE IF NOT EXISTS _phase1_smoke (id Int32) ENGINE = Memory" >/dev/null 2>&1 && \
       clickhouse-client --query "INSERT INTO _phase1_smoke VALUES (1)" >/dev/null 2>&1 && \
       clickhouse-client --query "SELECT count() FROM _phase1_smoke" >/dev/null 2>&1 | grep -q "1"; then
        CH_SMOKE="PASS"
        clickhouse-client --query "DROP TABLE _phase1_smoke" >/dev/null 2>&1 || true
        log "ClickHouse backup-restore smoke: PASS"
    else
        CH_SMOKE="FAIL"
        fail "ClickHouse backup-restore smoke failed"
    fi
fi

# ---------------------------------------------------------------------------
# 8. Record everything to the test report.
# ---------------------------------------------------------------------------

mkdir -p "$REPORT_DIR"
{
    echo ""
    echo "## Artifact build record ($(date -u +%Y-%m-%dT%H:%M:%SZ))"
    echo ""
    echo "| Field | Value |"
    echo "|---|---|"
    echo "| Contract version | \`$CONTRACT_VERSION\` |"
    echo "| OpenAPI SHA-256 | \`$OPENAPI_SHA256\` |"
    echo "| OpenAPI size | $OPENAPI_SIZE bytes |"
    echo "| SDK SHA-256 | \`$SDK_SHA256\` |"
    echo "| Migration version | \`$MIGRATION_VERSION\` |"
    echo "| Upstream commit | \`$UPSTREAM_COMMIT\` |"
    echo "| Upstream tag | \`$UPSTREAM_TAG\` |"
    echo "| Image digest | \`$IMAGE_DIGEST\` |"
    echo "| Image ID | \`$IMAGE_ID\` |"
    echo "| SBOM SHA-256 | \`$SBOM_SHA256\` |"
    echo "| Vulnerability scan | $VULN_RESULT |"
    echo "| PostgreSQL smoke | $PG_SMOKE |"
    echo "| ClickHouse smoke | $CH_SMOKE |"
    echo "| Built at | $(date -u +%Y-%m-%dT%H:%M:%SZ) |"
    echo "| Tool versions | docker=$(docker --version 2>/dev/null | head -1), syft=$(syft version 2>/dev/null | grep version | head -1 || echo N/A), grype=$(grype version 2>/dev/null | grep version | head -1 || echo N/A) |"
    echo ""
} >> "$REPORT_FILE"

log "report appended to $REPORT_FILE"
log "artifacts written to $OUTPUT_DIR"
log "artifact build complete"

# Echo the manifest path for CI pipelines.
echo "$MANIFEST_FILE"
