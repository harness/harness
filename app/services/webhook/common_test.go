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
	"testing"
)

func TestCheckURL_LinkLocal(t *testing.T) {
	tests := []struct {
		name                string
		url                 string
		allowLoopback       bool
		allowPrivateNetwork bool
		allowLinkLocal      bool
		wantErr             bool
		desc                string
	}{
		// --- link-local: always blocked regardless of flags ---
		{
			name:    "gke_metadata_server",
			url:     "http://169.254.169.254/",
			wantErr: true,
			desc:    "GKE metadata server must be blocked (CVE: link-local SSRF)",
		},
		{
			name:    "gke_metadata_token_endpoint",
			url:     "http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token",
			wantErr: true,
			desc:    "GKE metadata token endpoint must be blocked",
		},
		{
			name:    "link_local_other",
			url:     "http://169.254.0.1/",
			wantErr: true,
			desc:    "any 169.254/16 address must be blocked",
		},
		{
			name:          "link_local_ignored_by_allow_loopback",
			url:           "http://169.254.169.254/",
			allowLoopback: true,
			wantErr:       true,
			desc:          "allowLoopback=true must not unlock link-local",
		},
		{
			name:                "link_local_ignored_by_allow_private",
			url:                 "http://169.254.169.254/",
			allowPrivateNetwork: true,
			wantErr:             true,
			desc:                "allowPrivateNetwork=true must not unlock link-local",
		},
		{
			name:                "link_local_ignored_by_both_flags",
			url:                 "http://169.254.169.254/",
			allowLoopback:       true,
			allowPrivateNetwork: true,
			wantErr:             true,
			desc:                "both allow flags must not unlock link-local",
		},
		{
			name:           "link_local_allowed_when_flag_set",
			url:            "http://169.254.169.254/",
			allowLinkLocal: true,
			wantErr:        false,
			desc:           "link-local allowed only when allowLinkLocal=true",
		},
		{
			name:           "ipv6_link_local_allowed_when_flag_set",
			url:            "http://[fe80::1]/",
			allowLinkLocal: true,
			wantErr:        false,
			desc:           "IPv6 link-local allowed only when allowLinkLocal=true",
		},
		// --- IPv6 link-local ---
		{
			name:    "ipv6_link_local",
			url:     "http://[fe80::1]/",
			wantErr: true,
			desc:    "IPv6 link-local (fe80::/10) must be blocked",
		},
		// --- loopback: blocked unless allowLoopback ---
		{
			name:    "localhost_string",
			url:     "http://localhost/",
			wantErr: true,
			desc:    "localhost string must be blocked",
		},
		{
			name:    "loopback_ip",
			url:     "http://127.0.0.1/",
			wantErr: true,
			desc:    "127.0.0.1 must be blocked by default",
		},
		{
			name:          "loopback_allowed_when_flag_set",
			url:           "http://127.0.0.1/",
			allowLoopback: true,
			wantErr:       false,
			desc:          "loopback allowed only when allowLoopback=true",
		},
		// --- private networks ---
		{
			name:    "rfc1918_10",
			url:     "http://10.0.0.1/",
			wantErr: true,
			desc:    "RFC1918 10/8 must be blocked by default",
		},
		{
			name:    "rfc1918_172",
			url:     "http://172.16.0.1/",
			wantErr: true,
			desc:    "RFC1918 172.16/12 must be blocked by default",
		},
		{
			name:    "rfc1918_192",
			url:     "http://192.168.1.1/",
			wantErr: true,
			desc:    "RFC1918 192.168/16 must be blocked by default",
		},
		{
			name:                "private_allowed_when_flag_set",
			url:                 "http://10.0.0.1/",
			allowPrivateNetwork: true,
			wantErr:             false,
			desc:                "private network allowed only when allowPrivateNetwork=true",
		},
		// --- valid external URLs ---
		{
			name:    "valid_https",
			url:     "https://example.com/webhook",
			wantErr: false,
			desc:    "valid external HTTPS URL must pass",
		},
		{
			name:    "valid_http",
			url:     "http://example.com/webhook",
			wantErr: false,
			desc:    "valid external HTTP URL must pass",
		},
		// --- scheme validation ---
		{
			name:    "invalid_scheme",
			url:     "ftp://example.com/webhook",
			wantErr: true,
			desc:    "non-http/https scheme must be blocked",
		},
		// --- link-local multicast ---
		{
			name:    "link_local_multicast_blocked",
			url:     "http://224.0.0.251/", // mDNS
			wantErr: true,
			desc:    "link-local multicast (mDNS 224.0.0.251) must be blocked",
		},
		{
			name:           "link_local_multicast_allowed_when_flag_set",
			url:            "http://224.0.0.251/",
			allowLinkLocal: true,
			wantErr:        false,
			desc:           "link-local multicast allowed when allowLinkLocal=true",
		},
		// --- catch-all: non-globally-unicast addresses ---
		{
			name:    "unspecified_blocked",
			url:     "http://0.0.0.0/",
			wantErr: true,
			desc:    "unspecified address 0.0.0.0 must be blocked",
		},
		{
			name:    "broadcast_blocked",
			url:     "http://255.255.255.255/",
			wantErr: true,
			desc:    "broadcast address 255.255.255.255 must be blocked",
		},
		{
			name:    "non_link_local_multicast_blocked",
			url:     "http://224.0.1.1/",
			wantErr: true,
			desc:    "non-link-local multicast must be blocked by the IsGlobalUnicast catch-all",
		},
		// --- DNS-based bypass (hostname resolving to blocked IP) ---
		// CheckURL only performs static/syntactic checks on literal IPs.
		// Hostnames that resolve to loopback/link-local/private at dial time
		// are NOT rejected here; the DialContext guard handles those at runtime.
		{
			name:    "nip_io_loopback_bypass_passes_url_check",
			url:     "http://localhost.nip.io/",
			wantErr: false,
			desc:    "hostname resolving to loopback passes CheckURL; DialContext blocks it at execution time",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckURL(tc.url, tc.allowLoopback, tc.allowPrivateNetwork, tc.allowLinkLocal, false)
			if (err != nil) != tc.wantErr {
				t.Errorf("%s\nCheckURL(%q, loopback=%v, private=%v, linklocal=%v) error=%v, wantErr=%v",
					tc.desc, tc.url, tc.allowLoopback, tc.allowPrivateNetwork, tc.allowLinkLocal, err, tc.wantErr)
			}
		})
	}
}
