// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aigateway

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedactRules(t *testing.T) {
	r := &Redactor{}

	tests := []struct {
		name, in, leak, want string
	}{
		{"email", "user jane.doe@example.com failed", "jane.doe@example.com", "[REDACTED:email]"},
		{"ipv4", "from 192.168.1.20 port 22", "192.168.1.20", "[REDACTED:ipv4]"},
		{"bearer", "curl -H 'X: Bearer abcdefgh12345678'", "abcdefgh12345678", "[REDACTED:bearer]"},
		{"auth header", "Authorization: Basic dXNlcjpwYXNz", "dXNlcjpwYXNz", "[REDACTED:auth_header]"},
		{"cookie", "Cookie: session=abc123; theme=dark", "session=abc123", "[REDACTED:cookie]"},
		{"credential", `db_password = "s3cr3t-value"`, "s3cr3t-value", "[REDACTED:credential]"},
		{"api key", "OPENROUTER_API_KEY=sk-or-v1-0123456789abcdef", "sk-or-v1-0123456789abcdef", "[REDACTED:"},
		{"url credentials", "postgres://app:pa55word@db:5432/x", "pa55word", "[REDACTED:url_credentials]"},
		{"jwt", "token eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			"dozjgNryP4J3jVmNHl0w5N", "[REDACTED:jwt]"},
		{"private key", "-----BEGIN RSA PRIVATE KEY-----\nMIIEow\n-----END RSA PRIVATE KEY-----", "MIIEow",
			"[REDACTED:private_key]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.Redact(tt.in)
			if strings.Contains(got, tt.leak) {
				t.Errorf("leaked %q in %q", tt.leak, got)
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("expected %q in %q", tt.want, got)
			}
		})
	}
}

func TestRedactKeepsUsefulContext(t *testing.T) {
	r := &Redactor{}
	in := "panic: runtime error: index out of range [3] with length 3\n\tat app/handler.go:42"
	if got := r.Redact(in); got != in {
		t.Errorf("stack traces without sensitive data must stay intact, got %q", got)
	}
}

func TestRedactKeepsJSONValid(t *testing.T) {
	r := &Redactor{}
	in := `{"dsn":"https://u:p4ss@host/db","note":"mail me at a@b.io","auth":"Bearer abcdefgh12345678"}`
	out := r.Redact(in)

	var v map[string]string
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("redacted JSON must stay valid: %v\n%s", err, out)
	}
	if strings.Contains(out, "p4ss") || strings.Contains(out, "a@b.io") || strings.Contains(out, "abcdefgh12345678") {
		t.Errorf("leak in %s", out)
	}
}

func TestRedactWithGitleaks(t *testing.T) {
	r, err := NewRedactor()
	if err != nil {
		t.Fatalf("failed to create redactor: %v", err)
	}
	// a GitHub personal access token is detected by gitleaks' default rules
	token := "ghp_" + strings.Repeat("a1B2c3D4e5", 3) + "XYZ123"
	got := r.Redact("export GH=" + token)
	if strings.Contains(got, token) {
		t.Errorf("gitleaks secret leaked: %q", got)
	}
}
