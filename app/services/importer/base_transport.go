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

package importer

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/harness/gitness/app/api/usererror"
)

var baseTransport http.RoundTripper

func init() {
	tr := http.DefaultTransport.(*http.Transport).Clone() //nolint:errcheck

	// the client verifies the server's certificate chain and host name
	tr.TLSClientConfig.InsecureSkipVerify = false

	// Overwrite DialContext method to block connections to localhost and private networks.
	tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		// create basic net.Dialer (Similar to what is used by http.DefaultTransport)
		dialer := &net.Dialer{Timeout: 30 * time.Second}

		// dial connection using
		con, err := dialer.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}

		tcpAddr, ok := con.RemoteAddr().(*net.TCPAddr)
		if !ok { // not expected to happen, but to be sure
			_ = con.Close()
			return nil, fmt.Errorf("address resolved to a non-TCP address (original: '%s', resolved: '%s')",
				addr, con.RemoteAddr())
		}

		if tcpAddr.IP.IsLoopback() {
			_ = con.Close()
			return nil, usererror.BadRequestf("Loopback address is not allowed.")
		}

		if tcpAddr.IP.IsPrivate() {
			_ = con.Close()
			return nil, usererror.BadRequestf("Private network address is not allowed.")
		}

		return con, nil
	}

	baseTransport = tr
}
