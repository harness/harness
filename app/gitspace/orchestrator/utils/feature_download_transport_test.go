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

package utils

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/harness/gitness/netpolicy"
)

func TestDownloadTarballNetworkPolicy(t *testing.T) {
	var hits int
	// httptest listens on 127.0.0.1, so it stands in for an internal host.
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = w.Write([]byte("feature"))
	}))
	defer internal.Close()

	tests := []struct {
		name     string
		policy   netpolicy.Policy
		url      string
		allowed  bool
		wantHits int
	}{
		{name: "loopback blocked by default", url: internal.URL + "/feature.tgz"},
		{
			name:     "loopback allowed when enabled",
			policy:   netpolicy.Policy{AllowLoopback: true},
			url:      internal.URL + "/feature.tgz",
			allowed:  true,
			wantHits: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hits = 0
			dir := t.TempDir()
			file := filepath.Join(dir, "feature.tgz")

			err := downloadTarball(context.Background(), NewFeatureDownloadTransport(tt.policy), tt.url, dir, file)

			if tt.allowed {
				if err != nil {
					t.Fatalf("download failed: %v", err)
				}
				if got, _ := os.ReadFile(file); string(got) != "feature" {
					t.Fatalf("unexpected file content %q", got)
				}
			} else if !errors.Is(err, netpolicy.ErrAddressNotAllowed) {
				t.Fatalf("expected %v, got %v", netpolicy.ErrAddressNotAllowed, err)
			}
			if hits != tt.wantHits {
				t.Fatalf("internal server was hit %d times, want %d", hits, tt.wantHits)
			}
		})
	}
}
