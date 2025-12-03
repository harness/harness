# NPM Registry Conformance Tests

This directory contains comprehensive conformance tests for the NPM package registry implementation in Gitness. The tests verify that the NPM registry API behaves correctly according to NPM registry specifications.

## Test Structure

The conformance tests are organized into the following categories:

### 1. Upload Tests (`01_upload_test.go`)
- **Non-scoped package upload**: Tests uploading regular NPM packages
- **Scoped package upload**: Tests uploading scoped packages (e.g., `@scope/package`)
- Validates proper handling of package metadata, attachments, and dist-tags

### 2. Download Tests (`02_download_test.go`)
- **Package file download**: Tests downloading package tarballs by version and filename
- **Download by name**: Tests downloading package files using simplified paths
- **Scoped package download**: Tests downloading scoped package files
- **HEAD requests**: Tests HEAD requests for package files to verify metadata without downloading content

### 3. Metadata Tests (`03_metadata_test.go`)
- **Package metadata retrieval**: Tests fetching complete package metadata including all versions
- **Scoped package metadata**: Tests metadata retrieval for scoped packages
- Validates metadata structure, version information, and dist-tags

### 4. Tag Operations Tests (`04_tag_operations_test.go`)
- **List tags**: Tests retrieving all dist-tags for a package
- **Add tags**: Tests adding new dist-tags to packages
- **Delete tags**: Tests removing dist-tags from packages
- **Scoped package tags**: Tests tag operations on scoped packages

### 5. Scoped Packages Tests (`05_scoped_packages_test.go`)
- **Complete lifecycle**: Tests the full lifecycle of scoped packages (upload, download, metadata, tags)
- **Multiple versions**: Tests handling multiple versions of the same scoped package
- **HEAD requests**: Tests HEAD requests specifically for scoped packages

### 6. Error Handling Tests (`06_error_handling_test.go`)
- **404 responses**: Tests proper 404 responses for non-existent packages, files, and tags
- **Malformed requests**: Tests handling of invalid JSON payloads and malformed requests
- **Invalid operations**: Tests error responses for invalid tag operations and version formats

### 7. Search Tests (`07_search_test.go`)
- **Package search**: Tests searching for packages by name and keywords
- **Scoped package search**: Tests searching for scoped packages
- **Empty results**: Tests search behavior when no packages match
- **Pagination**: Tests search with pagination parameters (size, from)

## API Endpoints Tested

The tests cover all major NPM registry API endpoints:

### Upload Operations
- `PUT /npm/{namespace}/{registry}/{id}/` - Upload non-scoped package
- `PUT /npm/{namespace}/{registry}/@{scope}/{id}/` - Upload scoped package

### Download Operations
- `GET /npm/{namespace}/{registry}/{id}/-/{version}/{filename}` - Download package file
- `GET /npm/{namespace}/{registry}/{id}/-/{filename}` - Download package file by name
- `GET /npm/{namespace}/{registry}/@{scope}/{id}/-/{version}/@{scope}/{filename}` - Download scoped package file
- `GET /npm/{namespace}/{registry}/@{scope}/{id}/-/@{scope}/{filename}` - Download scoped package file by name

### Metadata Operations
- `GET /npm/{namespace}/{registry}/{id}/` - Get package metadata
- `GET /npm/{namespace}/{registry}/@{scope}/{id}/` - Get scoped package metadata
- `HEAD /npm/{namespace}/{registry}/{id}/-/{filename}` - Head package file
- `HEAD /npm/{namespace}/{registry}/@{scope}/{id}/-/@{scope}/{filename}` - Head scoped package file

### Tag Operations
- `GET /npm/{namespace}/{registry}/-/package/{id}/dist-tags/` - List tags
- `PUT /npm/{namespace}/{registry}/-/package/{id}/dist-tags/{tag}` - Add tag
- `DELETE /npm/{namespace}/{registry}/-/package/{id}/dist-tags/{tag}` - Delete tag
- `GET /npm/{namespace}/{registry}/-/package/@{scope}/{id}/dist-tags/` - List scoped tags
- `PUT /npm/{namespace}/{registry}/-/package/@{scope}/{id}/dist-tags/{tag}` - Add scoped tag
- `DELETE /npm/{namespace}/{registry}/-/package/@{scope}/{id}/dist-tags/{tag}` - Delete scoped tag

### Search Operations
- `GET /npm/{namespace}/{registry}/-/v1/search/` - Search packages

## Configuration

The tests use environment variables for configuration:

- `REGISTRY_ROOT_URL` - Base URL of the Gitness server (default: `http://localhost:3000`)
- `REGISTRY_USERNAME` - Username for authentication (default: `admin@gitness.io`)
- `REGISTRY_PASSWORD` - Authentication token (required)
- `REGISTRY_NAMESPACE` - Namespace for test registry (required)
- `REGISTRY_NAME` - Name of test registry (required)
- `DEBUG` - Enable debug logging (default: `true`)

## Test Data Generation

The tests use helper functions to generate realistic NPM package data:

- **Package metadata**: Generates complete NPM package.json-style metadata
- **Tarball attachments**: Creates base64-encoded mock tarball content
- **Version information**: Generates unique versions and timestamps
- **Scoped packages**: Handles both scoped and non-scoped package naming

## Running the Tests

```bash
# Run all NPM conformance tests
go test -v ./registry/tests/npm/

# Run specific test categories
go test -v ./registry/tests/npm/ -ginkgo.focus="Upload"
go test -v ./registry/tests/npm/ -ginkgo.focus="Download"
go test -v ./registry/tests/npm/ -ginkgo.focus="Tag Operations"

# Run with debug output
DEBUG=true go test -v ./registry/tests/npm/
```

## Test Isolation

Each test generates unique package names and versions to ensure test isolation:

- Package names include timestamps and test IDs
- Versions are generated with unique identifiers
- Tests clean up after themselves where possible
- No dependencies between individual test cases

## Compliance

These tests verify compliance with:

- NPM registry API specifications
- Standard NPM client behavior expectations
- Scoped package handling requirements
- Error response standards
- Search API functionality

The tests ensure that the Gitness NPM registry implementation can serve as a drop-in replacement for standard NPM registries while maintaining full compatibility with NPM clients and tooling.
