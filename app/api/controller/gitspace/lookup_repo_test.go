// Copyright 2026 Harness, Inc.
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

package gitspace

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"github.com/harness/gitness/netpolicy"
)

type fakeResolver map[string][]netip.Addr

func (r fakeResolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	addrs, ok := r[host]
	if !ok {
		return nil, errors.New("no such host")
	}
	return addrs, nil
}

func TestCheckLookupURLAllowed(t *testing.T) {
	resolver := fakeResolver{
		"github.com":       {netip.MustParseAddr("140.82.121.4")},
		"metadata.attack":  {netip.MustParseAddr("169.254.169.254")},
		"internal.example": {netip.MustParseAddr("10.0.0.5")},
		"rebind.example":   {netip.MustParseAddr("140.82.121.4"), netip.MustParseAddr("127.0.0.1")},
	}

	tests := []struct {
		name    string
		policy  netpolicy.Policy
		url     string
		allowed bool
	}{
		{name: "public host", url: "https://github.com/harness/harness.git", allowed: true},
		{name: "loopback literal", url: "http://127.0.0.1:6379/", allowed: false},
		{name: "localhost name", url: "http://localhost:3000/repo.git", allowed: false},
		{name: "cloud metadata literal", url: "http://169.254.169.254/latest/meta-data/", allowed: false},
		{name: "name resolving to metadata", url: "http://metadata.attack/repo.git", allowed: false},
		{name: "name resolving to private network", url: "https://internal.example/repo.git", allowed: false},
		{name: "any internal address in the answer", url: "https://rebind.example/repo.git", allowed: false},
		{name: "ipv4-mapped ipv6 loopback", url: "http://[::ffff:127.0.0.1]/repo.git", allowed: false},
		{name: "unresolvable host", url: "https://does-not-exist.example/repo.git", allowed: false},
		{name: "file scheme", url: "file:///var/lib/gitness/repos/1.git", allowed: false},
		{name: "ssh scheme", url: "ssh://git@internal.example/repo.git", allowed: false},
		{
			name:    "private network allowed by config",
			policy:  netpolicy.Policy{AllowPrivateNetwork: true},
			url:     "https://internal.example/repo.git",
			allowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{networkPolicy: tt.policy, resolver: resolver}
			err := c.checkLookupURLAllowed(context.Background(), tt.url)
			if tt.allowed && err != nil {
				t.Fatalf("expected %q to be allowed, got %v", tt.url, err)
			}
			if !tt.allowed && !errors.Is(err, ErrURLNotAllowed) {
				t.Fatalf("expected %q to be rejected with ErrURLNotAllowed, got %v", tt.url, err)
			}
		})
	}
}

func TestSanitizeLookupRepoInputRequiresHTTP(t *testing.T) {
	c := &Controller{}
	for _, raw := range []string{"file:///etc/passwd", "ext::sh -c id", "git://example.com/repo.git", "example.com/repo"} {
		if err := c.sanitizeLookupRepoInput(&LookupRepoInput{URL: raw}); err == nil {
			t.Errorf("expected %q to be rejected", raw)
		}
	}
	if err := c.sanitizeLookupRepoInput(&LookupRepoInput{URL: "https://github.com/harness/harness"}); err != nil {
		t.Errorf("expected an https URL to pass, got %v", err)
	}
}
