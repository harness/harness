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
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/harness/gitness/netpolicy"
	"github.com/harness/gitness/tracing"
)

var (
	errLoopbackNotAllowed       = errors.New("loopback not allowed")
	errLinkLocalNotAllowed      = errors.New("link-local address not allowed")
	errPrivateNetworkNotAllowed = errors.New("private network not allowed")
)

// addrClassError maps the class of a blocked address to the error reported for
// the webhook execution. The webhook URL is configured by a repository admin
// and the reason is shown to them, so the class is not hidden here.
func addrClassError(class netpolicy.Class) error {
	switch class {
	case netpolicy.ClassLoopback:
		return errLoopbackNotAllowed
	case netpolicy.ClassLinkLocal:
		return errLinkLocalNotAllowed
	case netpolicy.ClassPrivate, netpolicy.ClassReserved:
		return errPrivateNetworkNotAllowed
	case netpolicy.ClassPublic:
		return nil
	default:
		return nil
	}
}

func newHTTPClient(
	allowLoopback bool,
	allowPrivateNetwork bool,
	allowLinkLocal bool,
	disableSSLVerification bool,
	traceHTTP bool,
) *http.Client {
	// Clone http.DefaultTransport (used by http.DefaultClient)
	tr := http.DefaultTransport.(*http.Transport).Clone() //nolint:errcheck

	tr.TLSClientConfig.InsecureSkipVerify = disableSSLVerification

	policy := netpolicy.Policy{
		AllowLoopback:       allowLoopback,
		AllowPrivateNetwork: allowPrivateNetwork,
		AllowLinkLocal:      allowLinkLocal,
	}

	// create basic net.Dialer (Similar to what is used by http.DefaultTransport),
	// with a Control function that rejects blocked destinations.
	// NOTE: Control runs on the resolved address before the connect syscall, so a
	// blocked destination is never contacted and DNS resolution can't bypass it.
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Control:   policy.ControlFunc(addrClassError),
	}

	tr.DialContext = dialer.DialContext

	var transport http.RoundTripper = tr
	if traceHTTP {
		transport = tracing.ExternalHTTPTransport(tr)
	}

	// httpClient is similar to http.DefaultClient, just with custom http.Transport
	return &http.Client{
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
