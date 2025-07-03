# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # 
# # # # # # # # # # # # # # Cargo Conformance Tests # # # # # # # # # # # # # # # # # # # # # # # 
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # 
# Run Cargo tests directly using go test
echo "Executing Cargo tests..."
CARGO_EXIT_CODE=0

# Export variables that setup_test.sh will use
export REGISTRY_SERVER_URL="$1"
export REGISTRY_TOKEN="$token"
export REGISTRY_SPACE="$space_lower"
export REGISTRY_NAME=""
export REGISTRY_DEBUG="true"

# Run setup script to create resources and set environment variables
echo "Setting up Cargo test environment..."
echo "Using Cargo space: $space_lower"
bash "./registry/tests/cargo/scripts/setup_test.sh"

# Source the environment file created by setup script
source "/tmp/cargo_env.sh"
echo "Cargo test environment loaded from /tmp/cargo_env.sh"

# Save the original directory before running Cargo tests
CARGO_ORIGINAL_DIR="$(pwd)"

# Ensure we're using the correct Go environment
export GO111MODULE=auto

# Run the tests directly in the Cargo directory
echo "Running Cargo tests in directory: $(pwd)"

set -o pipefail  # Ensure we get the exit code from go test, not tee
go test -v ./registry/tests/cargo | tee test_output.txt
CARGO_EXIT_CODE=$?
set +o pipefail  # Reset pipefail setting

# Report test execution status
echo "Cargo test execution completed with exit code: $CARGO_EXIT_CODE"

# Copy test output to original directory for processing
cp test_output.txt "$CARGO_ORIGINAL_DIR/" || echo "Warning: Could not copy test output file"

# Return to the original directory after Cargo tests
cd "$CARGO_ORIGINAL_DIR"

# Display test output
if [ -f "test_output.txt" ]; then
    cat "test_output.txt"
else
    echo "Warning: test_output.txt not found"
fi

echo "Cargo tests completed with status: $CARGO_EXIT_CODE"

bash "./registry/tests/scripts/generate_report.sh" "test_output.txt" "cargo"

if [ $CARGO_EXIT_CODE -ne 0 ]; then
    echo "Cargo tests failed: CARGO_EXIT_CODE=$CARGO_EXIT_CODE"
    TEST_EXIT_CODE=1
    exit $CARGO_EXIT_CODE
fi

