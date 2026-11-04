package gitspace

import (
	"context"
	"strings"
	"testing"

	"github.com/harness/gitness/netpolicy"
	"github.com/harness/gitness/types"
)

func TestSanitizeCreateInput_URLValidation(t *testing.T) {
	// 1. Create a mock config and set the fields directly using dot notation.
	mockConfig := &types.Config{}
	mockConfig.Gitspace.AllowLoopback = false
	mockConfig.Gitspace.AllowPrivateNetwork = false
	mockConfig.Gitspace.AllowLinkLocal = false
	mockConfig.Gitspace.AllowedSchemes = []string{"https", "http", "ssh", "git"}

	// 2. Instantiate the Controller exactly as ProvideController would
	ctrl := &Controller{
		networkPolicy: netpolicy.Policy{
			AllowLoopback:       mockConfig.Gitspace.AllowLoopback,
			AllowPrivateNetwork: mockConfig.Gitspace.AllowPrivateNetwork,
			AllowLinkLocal:      mockConfig.Gitspace.AllowLinkLocal,
		},
		allowedSchemes: mockConfig.Gitspace.AllowedSchemes,
	}

	tests := []struct {
		name          string
		url           string
		expectError   bool
		errorContains string
	}{
		// VALID CASES (Using real, publicly resolvable domains so netpolicy allows them)
		{
			name:        "valid https URL",
			url:         "https://github.com/harness/gitness.git",
			expectError: false,
		},
		{
			name:        "valid http URL (public domain)",
			url:         "http://example.com/repo.git",
			expectError: false,
		},
		{
			name:        "valid ssh URL (public domain)",
			url:         "ssh://git@github.com/harness/harness.git",
			expectError: false,
		},
		{
			name:        "valid git protocol URL (public domain)",
			url:         "git://github.com/harness/harness.git",
			expectError: false,
		},

		// INVALID SCHEMES (LFI / RCE Prevention)
		{
			name:          "file scheme blocked (LFI)",
			url:           "file:///etc/passwd",
			expectError:   true,
			errorContains: "scheme",
		},
		{
			name:          "ext scheme blocked (RCE)",
			url:           "ext::sh,-c,echo%20pwned::/tmp/test",
			expectError:   true,
			errorContains: "scheme",
		},
		{
			name:          "ftp scheme blocked",
			url:           "ftp://example.com/repo.git",
			expectError:   true,
			errorContains: "scheme",
		},

		// INVALID HOSTS (SSRF Prevention)
		{
			name:          "AWS metadata IP blocked",
			url:           "http://169.254.169.254/latest/meta-data/",
			expectError:   true,
			errorContains: "not allowed",
		},
		{
			name:          "localhost loopback blocked",
			url:           "http://127.0.0.1:3000/repo.git",
			expectError:   true,
			errorContains: "not allowed",
		},
		{
			name:          "private network IP blocked",
			url:           "http://10.0.0.5/internal-repo.git",
			expectError:   true,
			errorContains: "not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &CreateInput{
				CodeRepoURL:        tt.url,
				SpaceRef:           "1",
				ResourceIdentifier: "valid-identifier-1",
			}

			err := ctrl.sanitizeCreateInput(context.Background(), input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for URL %s, got nil", tt.url)
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for valid URL %s: %v", tt.url, err)
				}
			}
		})
	}
}
