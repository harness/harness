# Security Vulnerability Fixes

## CVE-2024-25621 - containerd Vulnerability

### Issue
CVE-2024-25621 affects containerd versions 0.1.0 through 1.7.28, 2.0.0-beta.0 through 2.0.6, 2.1.0-beta.0 through 2.1.4 and 2.2.0-beta.0. The vulnerability requires containerd >= v1.7.29 to be fixed.

### Solution Applied
A replace directive has been added to `go.mod` to force containerd to use version 1.7.29 or later:

```go
// Force containerd to a secure version to fix CVE-2024-25621 (requires >= v1.7.29)
replace github.com/containerd/containerd => github.com/containerd/containerd v1.7.29
```

### Next Steps

1. **Update dependencies:**
   ```bash
   go mod tidy
   go mod download
   ```

2. **Verify the fix:**
   ```bash
   go list -m github.com/containerd/containerd
   ```
   This should show version 1.7.29 or later.

3. **Run vulnerability scan:**
   ```bash
   make sec
   # or
   govulncheck ./...
   ```

4. **Test the build:**
   ```bash
   make build
   ```

### Additional Recommendations

1. **Update other dependencies:** Consider running `go get -u ./...` to update all dependencies to their latest secure versions.

2. **Monitor for new vulnerabilities:** Set up automated dependency scanning (e.g., Dependabot, Snyk) to catch future vulnerabilities early.

3. **Review transitive dependencies:** The containerd dependency comes through `github.com/docker/docker` or `github.com/drone-runners/drone-runner-docker`. Consider updating these packages to their latest versions as well.

### Related Dependencies
- `github.com/docker/docker` - May need updating to pull in newer containerd
- `github.com/drone-runners/drone-runner-docker` - May need updating to pull in newer containerd

