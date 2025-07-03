echo "Running OCI conformance tests..."
export OCI_ROOT_URL="http://$1"
export OCI_NAMESPACE="$space_lower/$conformance/testrepo"
export OCI_DEBUG="true"

export OCI_TEST_PUSH=1
export OCI_TEST_PULL=1
export OCI_TEST_CONTENT_DISCOVERY=1
export OCI_TEST_CONTENT_MANAGEMENT=1
export OCI_CROSSMOUNT_NAMESPACE="$space_lower/$crossmount/testrepo"
export OCI_AUTOMATIC_CROSSMOUNT="false"

export OCI_USERNAME="admin"
export OCI_PASSWORD="$pat"

# Save the original directory before running OCI tests
ORIGINAL_DIR="$(pwd)"

# Create a temporary directory to run tests outside of the module
TEMP_DIR=$(mktemp -d)

# Clone the repository directly into the temp dir to avoid module conflicts
echo "Cloning fresh copy of distribution-spec to avoid module conflicts..."
rm -rf "$TEMP_DIR/distribution-spec" 2>/dev/null || true
git clone https://github.com/opencontainers/distribution-spec.git "$TEMP_DIR/distribution-spec"

# Run the tests from the clean directory
cd "$TEMP_DIR/distribution-spec/conformance"

# Force Go to use its own modules, not the parent module
GO111MODULE=on \
GOWORK=off \
go test .

# Store OCI exit code for return
OCI_EXIT_CODE=$?

# Return to the original directory after OCI tests
echo "Returning to original directory: $ORIGINAL_DIR"
cd "$ORIGINAL_DIR"

echo "OCI tests completed with status: $OCI_EXIT_CODE"

if [ $OCI_EXIT_CODE -ne 0 ]; then
    echo "OCI tests failed: OCI_EXIT_CODE=$OCI_EXIT_CODE"
    TEST_EXIT_CODE=1
    exit $TEST_EXIT_CODE
fi

