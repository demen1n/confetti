#!/bin/bash
set -euo pipefail

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

REPO_URL="https://github.com/hgs3/confetti"
# Pinned upstream revision of the conformance suite, so test results only change
# when this line does. Override with CONFETTI_TESTS_REF=<commit> to try another one.
DEFAULT_REF="03e114d30b78f9cd49abf170122ca624bbd164bd"
REF="${CONFETTI_TESTS_REF:-$DEFAULT_REF}"
TESTS_DIR="tests/conformance"
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
TARGET="$SCRIPT_DIR/$TESTS_DIR"
STAGING="$TARGET.new"
TEMP_DIR=$(mktemp -d)

echo -e "${BLUE}📦 Downloading Confetti conformance tests...${NC}"

# Clean up on exit
cleanup() {
    echo -e "${YELLOW}🧹 Cleaning up...${NC}"
    rm -rf "$TEMP_DIR" "$STAGING"
}
trap cleanup EXIT

# Fetch only the tests directory at the pinned revision
echo -e "${BLUE}⬇️  Fetching $REPO_URL at $REF${NC}"
git -C "$TEMP_DIR" init -q
git -C "$TEMP_DIR" remote add origin "$REPO_URL"
git -C "$TEMP_DIR" sparse-checkout set tests/conformance
git -C "$TEMP_DIR" fetch -q --depth 1 --filter=blob:none origin "$REF"
git -C "$TEMP_DIR" checkout -q FETCH_HEAD
REVISION=$(git -C "$TEMP_DIR" rev-parse HEAD)

# Stage the new set next to the old one and swap it in only once it is complete,
# so stale fixtures and markers never survive and a failed download keeps the old set.
echo -e "${BLUE}📋 Copying test files...${NC}"
rm -rf "$STAGING"
mkdir -p "$STAGING"
cp -R "$TEMP_DIR/tests/conformance/." "$STAGING/"
echo "$REVISION" > "$STAGING/REVISION"

CONF_COUNT=$(find "$STAGING" -name "*.conf" | wc -l | tr -d ' ')
if [ "$CONF_COUNT" -eq 0 ]; then
    echo "error: no .conf files found at $REF" >&2
    exit 1
fi
PASS_COUNT=$(find "$STAGING" -name "*.pass" | wc -l | tr -d ' ')
FAIL_COUNT=$(find "$STAGING" -name "*.fail" | wc -l | tr -d ' ')

rm -rf "$TARGET"
mv "$STAGING" "$TARGET"

echo ""
echo -e "${GREEN}✅ Done!${NC}"
echo -e "${GREEN}📊 Downloaded revision ${REVISION}:${NC}"
echo -e "   • ${CONF_COUNT} test cases (.conf files)"
echo -e "   • ${PASS_COUNT} expected pass results"
echo -e "   • ${FAIL_COUNT} expected fail results"
echo ""
echo -e "${BLUE}🚀 Run tests with:${NC}"
echo -e "   go run tests/cmd/conformance.go -dir $TESTS_DIR -v"
