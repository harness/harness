# Load Testing Suite

This directory contains load and integration tests for Gitness using [Vegeta](https://github.com/tsenart/vegeta) HTTP load testing tool and [Testify](https://github.com/stretchr/testify) assertions.

## Test Suite Overview

### Available Tests

1. **TestUsageMetricsLoad** - Load tests the usage metrics endpoint
   - Tests: `/api/v1/spaces/{space}/metric` endpoint
   - Rate: 50 requests/second
   - Duration: 30 seconds
   - Assertions: 95% success rate, P95 < 500ms

2. **TestRepositoryImportAndFileAccess** - Integration test for repository operations
   - Authenticates to Gitness
   - Imports `github.com/google/uuid` repository via `/api/v1/repos/import`
   - Waits for import completion by polling the repository status
   - Fetches README.md from the imported repository
   - Validates file content
   - Cleans up (deletes repository via `/api/v1/repos/{space}/{repo}`)

3. **TestFileAccessLoad** - Load tests file access endpoint
   - Sets up: Imports a test repository from GitHub
   - Tests: File content endpoint with high concurrency
   - Rate: 100 requests/second
   - Duration: 30 seconds
   - Assertions: 99% success rate, P95 < 200ms, P99 < 500ms
   - Cleans up: Deletes the test repository

## Prerequisites

- Running Gitness instance (local or remote)
- Admin credentials for authentication
- Network access to GitHub for repository import

## Environment Variables

Set these environment variables before running the tests:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GITNESS_E2E_TEST_ENABLED` | **Yes** | - | Must be `"true"` to run tests |
| `GITNESS_BASE_URL` | No | `http://localhost:3000` | Gitness instance URL |
| `GITNESS_AUTH_TOKEN` | No | - | Pre-authenticated token (if not set, will login) |
| `GITNESS_PRINCIPAL_ADMIN_EMAIL` | No* | - | Admin email for login |
| `GITNESS_PRINCIPAL_ADMIN_PASSWORD` | No* | - | Admin password for login |
| `SPACE_REF` | No | `default` | Space identifier to use |

\* Required if `GITNESS_AUTH_TOKEN` is not provided

## Running the Tests

### Run All Tests

```bash
# Set required environment variables
export GITNESS_E2E_TEST_ENABLED=true
export GITNESS_BASE_URL=http://localhost:3000
export GITNESS_PRINCIPAL_ADMIN_EMAIL=admin@example.com
export GITNESS_PRINCIPAL_ADMIN_PASSWORD=adminpassword

# Run all tests
cd tests/load
go test -v
```

### Run Specific Test

```bash
# Run only the repository import test
go test -v -run TestRepositoryImportAndFileAccess

# Run only the file access load test
go test -v -run TestFileAccessLoad

# Run only the usage metrics load test
go test -v -run TestUsageMetricsLoad
```

### Using Pre-authenticated Token

If you already have an authentication token:

```bash
export GITNESS_E2E_TEST_ENABLED=true
export GITNESS_AUTH_TOKEN="your-token-here"
go test -v
```

### Run with Verbose Output

```bash
go test -v -count=1
```

The `-count=1` flag disables test caching, ensuring fresh results.

## Test Output

Each test provides detailed output including:

### Load Test Metrics

```
=== RUN   TestFileAccessLoad
=== File Access Load Test Results ===
Requests:      3000
Success Rate:  99.97%
Mean Latency:  45.2ms
P50 Latency:   38ms
P95 Latency:   125ms
P99 Latency:   287ms
Max Latency:   450ms
Throughput:    99.85 req/s
--- PASS: TestFileAccessLoad (35.42s)
```

### Integration Test Output

```
=== RUN   TestRepositoryImportAndFileAccess
    usage_metrics_test.go:168: Successfully authenticated
    usage_metrics_test.go:177: Repository imported: uuid-test-1234567890 (ID: 42)
    usage_metrics_test.go:180: Waiting for repository import to complete...
    usage_metrics_test.go:184: Repository import completed successfully
    usage_metrics_test.go:192: Successfully fetched README.md (2456 bytes)
    usage_metrics_test.go:199: Repository cleaned up successfully
--- PASS: TestRepositoryImportAndFileAccess (87.23s)
```

## Test Customization

### Adjust Load Test Parameters

Edit the test file to modify load test parameters:

```go
// In TestFileAccessLoad
rate := vegeta.Rate{Freq: 100, Per: time.Second} // Change request rate
duration := 30 * time.Second                      // Change test duration

// Adjust assertions
assert.GreaterOrEqual(t, metrics.Success, 0.99, "Success rate should be at least 99%")
assert.LessOrEqual(t, metrics.Latencies.P95, 200*time.Millisecond, "P95 under 200ms")
```

### Change Test Repository

To test with a different repository, modify the import request in the `importRepository` function:

```go
importReq := repoctl.ImportInput{
    ParentRef:   spaceRef,
    Identifier:  identifier,
    Description: "Test repository",
    Provider: importer.Provider{
        Type: importer.ProviderTypeGitHub, // or ProviderTypeGitLab, ProviderTypeBitbucket, etc.
    },
    ProviderRepo: "your-org/your-repo", // Change this
}
```

Available provider types:
- `importer.ProviderTypeGitHub` - GitHub repositories
- `importer.ProviderTypeGitLab` - GitLab repositories
- `importer.ProviderTypeBitbucket` - Bitbucket repositories
- `importer.ProviderTypeGitea` - Gitea repositories
- `importer.ProviderTypeGogs` - Gogs repositories
- `importer.ProviderTypeAzure` - Azure DevOps repositories

### Adjust Timeouts

```go
// Repository import timeout
err = waitForRepositoryReady(ctx, baseURL, authToken, spaceRef, repoIdentifier,
    5*time.Minute) // Increase if importing large repositories
```

## Troubleshooting

### Tests are Skipped

**Symptom**: Tests show as `SKIP` instead of running

**Solution**: Ensure `GITNESS_E2E_TEST_ENABLED=true` is set:
```bash
export GITNESS_E2E_TEST_ENABLED=true
```

### Authentication Failures

**Symptom**: `Failed to authenticate` errors

**Solutions**:
1. Verify credentials are correct
2. Check Gitness instance is running and accessible
3. Verify the base URL is correct
4. Try using a pre-authenticated token instead

### Repository Import Timeout

**Symptom**: `repository import timed out` error

**Solutions**:
1. Check GitHub is accessible from Gitness instance
2. Increase timeout in `waitForRepositoryReady`
3. Verify the repository exists and is public
4. Check Gitness logs for import errors

### Load Test Failures

**Symptom**: Success rate below threshold or high latency

**Possible Causes**:
1. Gitness instance is under-resourced
2. Network latency issues
3. Database performance bottlenecks
4. Load test rate is too aggressive

**Solutions**:
1. Reduce request rate: `rate := vegeta.Rate{Freq: 50, Per: time.Second}`
2. Scale up Gitness instance resources
3. Run tests from same network/region as Gitness
4. Relax assertion thresholds temporarily to understand baseline

### Import Request JSON Unmarshal Error

**Symptom**: `import failed with status 400: {"message":"Invalid Request Body: json: cannot unmarshal string into Go struct field ImportInput.provider of type importer.Provider."}`

**Cause**: The `Provider` field must be a struct, not a string

**Solution**: Ensure you're using the correct structure:
```go
// ✓ Correct - Provider is a struct
Provider: importer.Provider{
    Type: importer.ProviderTypeGitHub,
}

// ✗ Wrong - Provider as string (deprecated)
Provider: "github"
```

## Performance Baselines

Expected performance for a properly configured Gitness instance:

| Metric | File Access | Usage Metrics |
|--------|-------------|---------------|
| Success Rate | ≥99% | ≥95% |
| Mean Latency | <50ms | <100ms |
| P95 Latency | <200ms | <500ms |
| P99 Latency | <500ms | <1s |
| Throughput | ~100 req/s | ~50 req/s |

These baselines assume:
- Local or low-latency network
- Adequate system resources (2+ CPU cores, 4GB+ RAM)
- No other significant load on the system

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Load Tests

on:
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM
  workflow_dispatch:

jobs:
  load-test:
    runs-on: ubuntu-latest

    services:
      gitness:
        image: harness/gitness:latest
        ports:
          - 3000:3000
        env:
          GITNESS_PRINCIPAL_ADMIN_EMAIL: admin@test.com
          GITNESS_PRINCIPAL_ADMIN_PASSWORD: testpassword

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Wait for Gitness
        run: |
          timeout 60 bash -c 'until curl -f http://localhost:3000/api/v1/health; do sleep 2; done'

      - name: Run Load Tests
        env:
          GITNESS_E2E_TEST_ENABLED: true
          GITNESS_BASE_URL: http://localhost:3000
          GITNESS_PRINCIPAL_ADMIN_EMAIL: admin@test.com
          GITNESS_PRINCIPAL_ADMIN_PASSWORD: testpassword
        run: |
          cd tests/load
          go test -v -timeout 15m
```

## Implementation Details

### Type System

The tests use actual Gitness types from the project instead of custom mock types:

- **`repoctl.ImportInput`** - Repository import request structure
  - Uses `importer.Provider` struct with typed provider constants
  - Includes `ParentRef` field to specify the target space

- **`repoctl.RepositoryOutput`** - Repository response structure
  - Contains all repository metadata and state information
  - Includes `Importing` boolean field to track import status

- **`importer.Provider`** - Provider configuration
  - Type-safe provider constants (GitHub, GitLab, Bitbucket, etc.)
  - Supports authentication credentials if needed

### API Endpoints Used

The tests interact with these Gitness API endpoints:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/login` | POST | Authenticate and obtain access token |
| `/api/v1/repos/import` | POST | Import repository from external provider |
| `/api/v1/repos/{space}/{repo}` | GET | Get repository details and status |
| `/api/v1/repos/{space}/{repo}` | DELETE | Delete repository |
| `/api/v1/repos/{space}/{repo}/+/content/{path}?git_ref={ref}` | GET | Fetch file content from repository |
| `/api/v1/spaces/{space}/metric` | GET | Retrieve usage metrics for a space |

### Authentication

Tests support two authentication methods:

1. **Token-based**: Use `GITNESS_AUTH_TOKEN` environment variable
2. **Credential-based**: Use `GITNESS_PRINCIPAL_ADMIN_EMAIL` and `GITNESS_PRINCIPAL_ADMIN_PASSWORD`

The `authenticate` function handles credential-based login and token extraction.

## Contributing

When adding new load tests:

1. Follow the existing test patterns
2. Use testify assertions for clarity
3. **Use project types** - Import types from `app/api/controller` and `app/services` packages instead of creating custom structs
4. Always clean up resources (repositories, etc.)
5. Add appropriate environment variable documentation
6. Include expected performance baselines
7. Use meaningful test and metric names
8. Add test description in this README

## Dependencies

- **vegeta** - HTTP load testing tool
- **testify** - Testing toolkit with assertions

Both are already included in `go.mod`.

## References

- [Vegeta Documentation](https://github.com/tsenart/vegeta)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Gitness API Documentation](https://github.com/harness/gitness)