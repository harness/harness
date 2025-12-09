# Now run Maven tests
echo "Running Maven conformance tests..."
# Save the original directory before running OCI tests
ORIGINAL_DIR="$(pwd)"

# Create a Maven registry with timestamp to ensure uniqueness
MAVEN_REGISTRY_NAME="maven_registry_$(date +%s)"
echo "Creating registry: $MAVEN_REGISTRY_NAME with package type MAVEN"
curl --location "http://$1/api/v1/registry?space_ref=$space_lower" \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer '"$token" \
--header 'Accept: application/json' \
--data "{\"config\":{\"type\": \"VIRTUAL\"}, \"description\": \"Maven registry for conformance tests\", \"identifier\": \"$MAVEN_REGISTRY_NAME\", \"packageType\": \"MAVEN\",\"parentRef\": \"$space_lower\"}"

# Run Maven tests directly using go test
echo "Executing Maven tests..."
MAVEN_EXIT_CODE=0

# Export variables that setup_test.sh will use
export REGISTRY_SERVER_URL="$1"
export REGISTRY_TOKEN="$token"
export REGISTRY_SPACE="$space_lower"
export REGISTRY_NAME="$MAVEN_REGISTRY_NAME"
export REGISTRY_DEBUG="true"

# Run setup script to create resources and set environment variables
echo "Setting up Maven test environment..."
echo "Using Maven space: $space_lower and registry: $MAVEN_REGISTRY_NAME"
bash "./registry/tests/maven/scripts/setup_test.sh"

# Source the environment file created by setup script
source "/tmp/maven_env.sh"
echo "Maven test environment loaded from /tmp/maven_env.sh"

# Save the original directory before running Maven tests
MAVEN_ORIGINAL_DIR="$(pwd)"

# Ensure we're using the correct Go environment
export GO111MODULE=auto

# Run the tests directly in the Maven directory
echo "Running Maven tests in directory: $(pwd)"

set -o pipefail  # Ensure we get the exit code from go test, not tee
go test -v ./registry/tests/maven | tee test_output.txt
MAVEN_EXIT_CODE=$?
set +o pipefail  # Reset pipefail setting

# Report test execution status
echo "Maven test execution completed with exit code: $MAVEN_EXIT_CODE"

# Copy test output to original directory for processing
cp test_output.txt "$MAVEN_ORIGINAL_DIR/" || echo "Warning: Could not copy test output file"

# Return to the original directory after Maven tests
cd "$MAVEN_ORIGINAL_DIR"

bash "./registry/tests/scripts/generate_report.sh" "test_output.txt" "maven"

# Check for JUnit report generator script in Maven test directory
JUNIT_GENERATOR="$MAVEN_TEST_DIR/generate_junit_report.sh"
echo "Checking for JUnit report generator at: $JUNIT_GENERATOR"

if [ -f "$JUNIT_GENERATOR" ]; then
    echo "JUnit report generator found, executing..."
    chmod +x "$JUNIT_GENERATOR"
    "$JUNIT_GENERATOR"
else
    # Check in the current directory as fallback
    if [ -f "./registry/tests/maven/generate_junit_report.sh" ]; then
        echo "JUnit report generator found in current directory, executing..."
        chmod +x "./registry/tests/maven/generate_junit_report.sh"
        "./registry/tests/maven/generate_junit_report.sh"
    else
        echo "JUnit report generator not found, skipping JUnit report generation"
    fi
fi

cd "$ORIGINAL_DIR"    
echo "Maven tests completed with status: $MAVEN_EXIT_CODE"