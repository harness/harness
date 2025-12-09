#!/bin/bash

# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # 
# # # # # # # # # # # # # # NPM Conformance Tests # # # # # # # # # # # # # # # # # # # # # # # 
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # 

# Now run NPM tests
echo "Running NPM conformance tests..."
# Save the original directory before running NPM tests
ORIGINAL_DIR="$(pwd)"

# Create an NPM registry with timestamp to ensure uniqueness
NPM_REGISTRY_NAME="npm_registry_$(date +%s)"
echo "Creating registry: $NPM_REGISTRY_NAME with package type NPM"
curl --location "http://$1/api/v1/registry?space_ref=$space_lower" \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer '"$token" \
--header 'Accept: application/json' \
--data "{\"config\":{\"type\": \"VIRTUAL\"}, \"description\": \"NPM registry for conformance tests\", \"identifier\": \"$NPM_REGISTRY_NAME\", \"packageType\": \"NPM\",\"parentRef\": \"$space_lower\"}"

# Run NPM tests directly using go test
echo "Executing NPM tests..."
NPM_EXIT_CODE=0

# Export variables that setup_test.sh will use
export REGISTRY_SERVER_URL="$1"
export REGISTRY_TOKEN="$token"
export REGISTRY_SPACE="$space_lower"
export REGISTRY_NAME="$NPM_REGISTRY_NAME"
export REGISTRY_DEBUG="true"

# Run setup script to create resources and set environment variables
echo "Setting up NPM test environment..."
echo "Using NPM space: $space_lower and registry: $NPM_REGISTRY_NAME"
bash "./registry/tests/npm/scripts/setup_test.sh"

# Source the environment file created by setup script
source "/tmp/npm_env.sh"
echo "NPM test environment loaded from /tmp/npm_env.sh"

# Save the original directory before running NPM tests
NPM_ORIGINAL_DIR="$(pwd)"

# Ensure we're using the correct Go environment
export GO111MODULE=auto

# Run the tests directly in the NPM directory
echo "Running NPM tests in directory: $(pwd)"

set -o pipefail  # Ensure we get the exit code from go test, not tee
go test -v ./registry/tests/npm | tee test_output.txt
NPM_EXIT_CODE=$?
set +o pipefail  # Reset pipefail setting

# Report test execution status
echo "NPM test execution completed with exit code: $NPM_EXIT_CODE"

# Copy test output to original directory for processing
cp test_output.txt "$NPM_ORIGINAL_DIR/" || echo "Warning: Could not copy test output file"

# Return to the original directory after NPM tests
cd "$NPM_ORIGINAL_DIR"

echo "NPM tests completed with status: $NPM_EXIT_CODE"

bash "./registry/tests/scripts/generate_report.sh" "test_output.txt" "npm"

if [ $NPM_EXIT_CODE -ne 0 ]; then
    echo "NPM tests failed: NPM_EXIT_CODE=$NPM_EXIT_CODE"
    TEST_EXIT_CODE=1
    exit $NPM_EXIT_CODE
fi

cd "$ORIGINAL_DIR"
echo "NPM tests completed with status: $NPM_EXIT_CODE"
