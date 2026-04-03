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

package webhook

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// startLocalServer starts an httptest.Server bound to the given address and
// registers cleanup. Returns the bound port.
func startLocalServer(t *testing.T, addr string) int {
	t.Helper()
	lc := &net.ListenConfig{}
	ln, err := lc.Listen(context.Background(), "tcp", addr)
	if err != nil {
		t.Skipf("cannot bind %s: %v", addr, err)
	}
	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address is not *net.TCPAddr: %T", ln.Addr())
	}
	srv := &httptest.Server{
		Listener: ln,
		Config: &http.Server{
			ReadHeaderTimeout: 5 * time.Second,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		},
	}
	srv.Start()
	t.Cleanup(srv.Close)
	return tcpAddr.Port
}

func doGET(t *testing.T, client *http.Client, url string) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	return client.Do(req)
}

func TestHTTPClient_LinkLocalBlocked(t *testing.T) {
	t.Run("loopback_blocked_by_secure_client", func(t *testing.T) {
		port := startLocalServer(t, "127.0.0.1:0")

		client := newHTTPClient(false, false, false, true)
		resp, err := doGET(t, client, fmt.Sprintf("http://127.0.0.1:%d/", port))
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		if err == nil {
			t.Fatal("expected error connecting to loopback, got nil")
		}
		if !errors.Is(err, errLoopbackNotAllowed) && !strings.Contains(err.Error(), errLoopbackNotAllowed.Error()) {
			t.Errorf("expected errLoopbackNotAllowed, got: %v", err)
		}
	})

	t.Run("loopback_allowed_when_flag_set", func(t *testing.T) {
		port := startLocalServer(t, "127.0.0.1:0")

		client := newHTTPClient(true, false, false, true)
		resp, err := doGET(t, client, fmt.Sprintf("http://127.0.0.1:%d/", port))
		if err != nil {
			t.Fatalf("expected success with allowLoopback=true, got: %v", err)
		}
		resp.Body.Close()
	})

	t.Run("link_local_error_is_distinct_from_loopback_error", func(t *testing.T) {
		if errors.Is(errLinkLocalNotAllowed, errLoopbackNotAllowed) {
			t.Error("errLinkLocalNotAllowed and errLoopbackNotAllowed must be distinct errors")
		}
	})

	t.Run("redirect_not_followed", func(t *testing.T) {
		// Verify the client returns the 302 response as-is and never contacts
		// the redirect target. allowLoopback=true so httptest servers are reachable;
		// the key assertion is the 302 status, not the loopback restriction.
		targetSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			t.Error("redirect target was contacted — client followed the redirect")
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(targetSrv.Close)

		redirectSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, targetSrv.URL+"/", http.StatusFound)
		}))
		t.Cleanup(redirectSrv.Close)

		client := newHTTPClient(true, false, false, true)
		resp, err := doGET(t, client, redirectSrv.URL+"/")
		if err != nil {
			t.Fatalf("unexpected error reaching redirect server: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusFound {
			t.Errorf("expected 302 (redirect not followed), got %d", resp.StatusCode)
		}
	})

	// dns_loopback_bypass_blocked verifies that a hostname resolving to a loopback
	// address is rejected by the DialContext guard even though CheckURL passes it
	// (CheckURL only checks literal IPs, not DNS resolutions).
	// localhost.nip.io is a public wildcard DNS service that resolves to 127.0.0.1,
	// simulating the SSRF vector: attacker-controlled DNS → cluster-internal address.
	//
	// The DialContext opens a TCP connection first, then inspects the remote IP.
	// A real server must be bound on 127.0.0.1 so the TCP handshake succeeds and
	// the remoteIP check fires. Without a listening server the dial itself would
	// fail with a connection-refused error before the guard ever runs.
	t.Run("dns_loopback_bypass_blocked", func(t *testing.T) {
		addrs, lookupErr := net.DefaultResolver.LookupHost(context.Background(), "localhost.nip.io")
		if lookupErr != nil || len(addrs) == 0 {
			t.Skip("cannot resolve localhost.nip.io — DNS not available in this environment")
		}

		port := startLocalServer(t, "127.0.0.1:0")

		client := newHTTPClient(false, false, false, true)
		resp, err := doGET(t, client, fmt.Sprintf("http://localhost.nip.io:%d/", port))
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if err == nil {
			t.Fatal("expected error for DNS-based loopback bypass, got nil")
		}
		if !errors.Is(err, errLoopbackNotAllowed) && !strings.Contains(err.Error(), errLoopbackNotAllowed.Error()) {
			t.Errorf("expected errLoopbackNotAllowed for DNS bypass via localhost.nip.io, got: %v", err)
		}
	})
}
