#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

REPO_URL="https://github.com/hgs3/confetti"
TESTS_DIR="tests/conformance"
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
TEMP_DIR=$(mktemp -d)

echo -e "${BLUE}📦 Downloading Confetti conformance tests...${NC}"

# Clean up on exit
cleanup() {
    echo -e "${YELLOW}🧹 Cleaning up...${NC}"
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

# Clone only the tests directory
echo -e "${BLUE}⬇️  Fetching latest tests from $REPO_URL${NC}"
git clone --depth 1 --filter=blob:none --sparse "$REPO_URL" "$TEMP_DIR" 2>/dev/null

cd "$TEMP_DIR"
git sparse-checkout set tests/conformance

# Create tests directory if it doesn't exist
mkdir -p "$SCRIPT_DIR/$TESTS_DIR"

# Copy tests
echo -e "${BLUE}📋 Copying test files...${NC}"
cp -r tests/conformance/* "$SCRIPT_DIR/$TESTS_DIR/"

# Count test files
CONF_COUNT=$(find "$SCRIPT_DIR/$TESTS_DIR" -name "*.conf" | wc -l)
PASS_COUNT=$(find "$SCRIPT_DIR/$TESTS_DIR" -name "*.pass" | wc -l)
FAIL_COUNT=$(find "$SCRIPT_DIR/$TESTS_DIR" -name "*.fail" | wc -l)

echo ""
echo -e "${GREEN}✅ Done!${NC}"
echo -e "${GREEN}📊 Downloaded:${NC}"
echo -e "   • ${CONF_COUNT} test cases (.conf files)"
echo -e "   • ${PASS_COUNT} expected pass results"
echo -e "   • ${FAIL_COUNT} expected fail results"
echo ""
echo -e "${BLUE}🚀 Run tests with:${NC}"
echo -e "   go run tests/cmd/conformance.go -dir $TESTS_DIR -v"