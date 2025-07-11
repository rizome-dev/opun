#!/bin/bash

# E2E Test Runner for Opun
# This script runs all end-to-end tests

set -e

echo "🧪 Running Opun E2E Tests..."
echo "================================"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
PASSED=0
FAILED=0

# Function to run a test
run_test() {
    local test_name=$1
    local test_path=$2
    
    echo -e "\n${YELLOW}Running: ${test_name}${NC}"
    
    if go test -v ${test_path} -count=1 2>&1 | tee test_output.tmp; then
        echo -e "${GREEN}✓ ${test_name} passed${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ ${test_name} failed${NC}"
        ((FAILED++))
    fi
    
    rm -f test_output.tmp
}

# Check prerequisites
echo "Checking prerequisites..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    exit 1
fi

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo -e "${RED}Error: Must run from project root directory${NC}"
    exit 1
fi

# Build the project first
echo "Building opun..."
if ! make build; then
    echo -e "${RED}Error: Build failed${NC}"
    exit 1
fi

# Run unit tests first (quick smoke test)
echo -e "\n${YELLOW}Running unit tests first...${NC}"
if go test -short ./... > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Unit tests passed${NC}"
else
    echo -e "${RED}✗ Unit tests failed - skipping E2E tests${NC}"
    exit 1
fi

# Run E2E tests
echo -e "\n${YELLOW}Starting E2E tests...${NC}"

# PTY Automation Tests
run_test "PTY Automation Tests" "./test/e2e -run TestPTYAutomation"
run_test "Claude Provider Tests" "./test/e2e -run TestClaudeProvider"
run_test "Gemini Provider Tests" "./test/e2e -run TestGeminiProvider"

# Workflow Execution Tests
run_test "Workflow Execution Tests" "./test/e2e -run TestWorkflowExecution"
run_test "Workflow Parser Tests" "./test/e2e -run TestWorkflowParser"

# CLI Command Tests
run_test "CLI Commands Tests" "./test/e2e -run TestCLICommands"
run_test "Workflow Run Tests" "./test/e2e -run TestWorkflowRun"

# Prompt Garden Tests
run_test "Prompt Garden Tests" "./test/e2e -run TestPromptGarden"

# Integration test (if exists)
if [ -f "test/e2e/integration_test.go" ]; then
    run_test "Integration Tests" "./test/e2e -run TestIntegration"
fi

# Summary
echo -e "\n================================"
echo "Test Summary:"
echo -e "${GREEN}Passed: ${PASSED}${NC}"
echo -e "${RED}Failed: ${FAILED}${NC}"
echo "================================"

# Exit with appropriate code
if [ $FAILED -eq 0 ]; then
    echo -e "\n${GREEN}✅ All E2E tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}❌ Some E2E tests failed${NC}"
    exit 1
fi