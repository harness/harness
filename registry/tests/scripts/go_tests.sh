# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # 
# # # # # # # # # # # # # # Go Conformance Tests # # # # # # # # # # # # # # # # # # # # # # # 
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # 
# Run Go tests directly using go test
echo "Executing Go tests..."
GO_EXIT_CODE=0

# Export variables that setup_test.sh will use
export REGISTRY_SERVER_URL="$1"
export REGISTRY_TOKEN="$token"
export REGISTRY_SPACE="$space"
export REGISTRY_NAME=""
export REGISTRY_DEBUG="true"
export GITNESS_REGISTRY_STORAGE_TYPE="filesystem"
export GITNESS_REGISTRY_FILESYSTEM_ROOT_DIRECTORY="/tmp/go"

# Run setup script to create resources and set environment variables
echo "Setting up Go test environment..."
echo "Using Go space: $space"
bash "./registry/tests/gopkg/scripts/setup_test.sh"

# Source the environment file created by setup script
source "/tmp/go_env.sh"
echo "Go test environment loaded from /tmp/go_env.sh"

# Save the original directory before running Go tests
GO_ORIGINAL_DIR="$(pwd)"

# Ensure we're using the correct Go environment
export GO111MODULE=auto

# Run the tests directly in the Go directory
echo "Running Go tests in directory: $(pwd)"

set -o pipefail  # Ensure we get the exit code from go test, not tee
go test -v ./registry/tests/gopkg | tee test_output.txt
GO_EXIT_CODE=$?
set +o pipefail  # Reset pipefail setting

# Report test execution status
echo "Go test execution completed with exit code: $GO_EXIT_CODE"

# Copy test output to original directory for processing
cp test_output.txt "$GO_ORIGINAL_DIR/" || echo "Warning: Could not copy test output file"

# Return to the original directory after Go tests
cd "$GO_ORIGINAL_DIR"

echo "Go tests completed with status: $GO_EXIT_CODE"

bash "./registry/tests/scripts/generate_report.sh" "test_output.txt" "go"

if [ $GO_EXIT_CODE -ne 0 ]; then
    echo "Go tests failed: GO_EXIT_CODE=$GO_EXIT_CODE"
    TEST_EXIT_CODE=1
    exit $GO_EXIT_CODE
fi

