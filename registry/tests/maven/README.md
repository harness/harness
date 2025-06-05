# Gitness Maven Registry Conformance Tests

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)](https://harness.io)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

This package contains conformance tests for the Maven registry functionality in Gitness, ensuring compliance with Maven repository specifications and validating the correct behavior of the Gitness Maven registry implementation.

## Test Suites

### 1. OCI (Docker) Registry Conformance Tests

These tests validate compliance with the [OCI Distribution Specification](https://github.com/opencontainers/distribution-spec).

**Functionality Tested:**
- Content management (push/pull)
- Content discovery
- Cross-repository mounting
- Blob operations
- Tag operations

**Test Status:** All enabled and functioning

### 2. Maven Registry Conformance Tests

These tests validate compliance with Maven repository specifications.

## Test Categories

The test suite is organized into several categories, each focusing on specific aspects of Maven registry functionality:

| Category | Status | Details |
|----------|--------|--------|
| **Basic Download** | ✅ Enabled | Artifact download, basic functionality |
| **Basic Upload** | ✅ Enabled | Simple artifact upload operations |
| **Content Discovery** | ✅ Enabled | Basic artifact discovery and listing |
| **Error Handling** | ✅ Enabled | Validation of error responses and edge cases |

#### Enabled Test Details

**Basic Download Tests**
- `should download an artifact`: Verifies ability to download previously uploaded JAR files
  - Tests GET operations with proper content type handling
  - Verifies content integrity is maintained

**Basic Upload Tests**
- `should upload an artifact`: Validates JAR files can be uploaded to the repository
  - Tests PUT operations with application/java-archive content type
  - Verifies 201 Created status code on successful upload

**Content Discovery Tests**
- `should find artifacts by version`: Tests artifact lookup by specific version
  - Ensures properly versioned artifacts can be located
  - Validates correct content is returned
- `should find POM files`: Tests POM file discovery and retrieval
  - Ensures Maven build descriptors can be located
  - Verifies correct content is returned
- `should handle non-existent artifacts`: Verifies 404 responses for missing artifacts
  - Tests proper error handling for non-existent files

**Error Handling Tests**
- `should reject invalid artifact path`: Verifies proper handling of malformed paths
  - Tests 500 error response with appropriate error message
- `should reject invalid version format`: Tests validation of Maven version format
  - Ensures non-compliant version strings are rejected with 404
- `should reject invalid groupId`: Tests validation of groupId format
  - Verifies rejection of path traversal attempts with 404
- `should reject unauthorized access`: Tests authentication requirements
  - Ensures 401 Unauthorized response with invalid credentials
- `should reject access to non-existent space`: Tests space/namespace validation
  - Verifies 500 error response when accessing non-existent spaces
- `should handle invalid POM XML`: Tests POM content validation
  - Verifies rejecting malformed POM files
- `should handle mismatched content type`: Tests content type validation
  - Ensures content type headers match actual content

**Current Status:** 12 tests passing, 0 tests skipped, 100% success rate

## Running Tests

The Maven conformance tests are integrated into the main Gitness Makefile and can be run in two different modes:

### 1. Standard Test Mode

This mode starts a new Gitness server instance, runs the tests, and then shuts down the server:

```bash
# Run both OCI and Maven conformance tests
make ar-conformance-test
```

### 2. Hot Test Mode

This mode runs tests against an already running Gitness server, which is useful for development and debugging:

```bash
# Run tests against an already running Gitness server
make ar-hot-conformance-test
```

### 3. Running Individual Test Categories

You can run specific test categories using the Go test command with filters:

```bash
# Run only download tests
cd registry/tests/maven
go test -v -run "TestMavenConformance/Download"

# Run only upload tests
go test -v -run "TestMavenConformance/Upload"
```

## Configuration

The test environment is automatically set up by the test scripts, with no manual configuration required in most cases.

### Automatic Setup

The setup process includes:

- Authentication with the Gitness server using admin credentials
- Creating unique test spaces with timestamp-based names
- Creating Maven registries with proper configuration
- Setting all required environment variables

### Environment Variables

The following environment variables can be used to customize the test behavior:

| Variable | Description | Default |
|----------|-------------|---------|
| `REGISTRY_ROOT_URL` | Base URL of the Gitness server | `http://localhost:3000` |
| `REGISTRY_USERNAME` | Username for authentication | `admin@gitness.io` |
| `REGISTRY_PASSWORD` | Password or token for authentication | Generated automatically |
| `REGISTRY_NAMESPACE` | Space/namespace for testing | Generated automatically |
| `REGISTRY_NAME` | Registry name for testing | Generated automatically |
| `DEBUG` | Enable verbose logging | `true` |

## Architecture

### Test Framework

The Maven conformance tests are built using the following components:

1. **Ginkgo/Gomega**: BDD-style test framework for structured and readable tests
2. **HTTP Client**: Custom HTTP client for Maven registry API interactions
3. **Setup Scripts**: Bash scripts for environment preparation and cleanup

### Directory Structure

```
registry/tests/maven/
├── 00_conformance_suite_test.go  # Main test suite definition
├── 01_download_test.go          # Download functionality tests
├── 02_upload_test.go            # Upload functionality tests
├── 03_content_discovery_test.go # Content discovery tests
├── 05_error_test.go             # Error handling tests
├── config.go                    # Test configuration and helpers
├── client.go                    # HTTP client implementation
├── reporter.go                  # Test reporting functionality
└── scripts/                     # Helper scripts
    └── setup_test.sh            # Environment setup script
```

### Test Isolation

Each test uses unique artifact names and versions to prevent conflicts between test runs. This is achieved through:

1. Timestamp-based unique identifiers
2. Context-specific naming prefixes
3. BeforeAll setup with Ordered test containers

## Extending the Test Suite

### Adding New Tests

To add new tests to the Maven conformance suite:

1. Create a new test file following the naming convention: `XX_category_test.go`
2. Import the necessary packages:
   ```go
   import (
       "github.com/onsi/ginkgo/v2"
       "github.com/onsi/gomega"
   )
   ```
3. Define your test function using Ginkgo's BDD style:
   ```go
   func testNewCategory() {
       ginkgo.Describe("New Category", func() {
           ginkgo.Context("Feature X", ginkgo.Ordered, func() {
               // Define test variables
               var artifactName string
               
               // Setup with BeforeAll
               ginkgo.BeforeAll(func() {
                   // Use unique artifact names
                   artifactName = GetUniqueArtifactName("category", timestamp)
                   // Setup code...
               })
               
               // Define test cases
               ginkgo.It("should do something", func() {
                   // Test code...
                   gomega.Expect(result).To(gomega.Equal(expected))
               })
           })
       })
   }
   ```
4. Add your test function to the main test suite in `00_conformance_suite_test.go`

### Best Practices

1. **Unique Artifacts**: Always use `GetUniqueArtifactName()` and `GetUniqueVersion()` to prevent test conflicts
2. **Test Isolation**: Use `ginkgo.Ordered` contexts with `BeforeAll` for setup
3. **Error Handling**: Test both success and error cases
4. **Clean Code**: Follow Go best practices and maintain consistent style
5. **Documentation**: Add clear comments explaining test purpose and expectations
## Test Reports

### JSON Reports

Test results are saved in `maven_conformance_report.json` with the following information:

```json
{
  "timestamp": "2025-05-13T14:15:16Z",
  "summary": {
    "passed": 12,
    "failed": 0,
    "pending": 0,
    "skipped": 0,
    "total": 12
  },
  "tests": [
    {
      "name": "Download/should download an artifact",
      "status": "passed",
      "duration": 0.123
    },
    // Additional test results...
  ]
}
```

### JUnit XML Reports

The test suite can also generate JUnit XML reports compatible with CI systems:

```bash
# Generate JUnit XML report
cd registry/tests/maven
./generate_junit_report.sh
```

## Troubleshooting

### Common Issues

1. **Connection Errors**
   - Ensure Gitness server is running (for hot tests)
   - Verify server URL is correct in environment variables

2. **Authentication Failures**
   - Check admin credentials in `.local.env`
   - Ensure token generation is working properly

3. **Registry Not Found**
   - Verify registry creation is successful
   - Check namespace/space name formatting

### Debugging

Enable verbose logging by setting `DEBUG=true` in the environment:

```bash
DEBUG=true make ar-hot-conformance-test
```

## License

Copyright 2023 Harness, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
