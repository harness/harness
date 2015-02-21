package proxy

import (
	"bytes"
	"fmt"
)

// bash header plus an embedded perl script that can be used
// as an alternative to socat to proxy tcp traffic.
const header = `#!/bin/bash
set +e
`

// TODO(bradrydzewski) probably going to remove this
//echo H4sICGKv1VQAA3NvY2F0LnBsAH1SXUvDQBB8Tn7FipUmkpr6gWBKgyIiBdGixVeJ6RZP00u4S6wi8be7t3exFsWEhNzO7M7MXba34kar+FHIuEJV+I1GmNwkyV2Zv2A9Wq+xwJzWfk/IqqlhDM+lkEEf+tHp2e3lfTj6Rj5hGc/Op4Oryd3s4joJ9nbDaFGqF6Air/gVU0M2nyua1Dug76pUZmrvkDSW79ATpUZTWIsPUomrkQF3NLt7WGaVY2tUr6g6OqNJMrm+mHFT4HtXZZ4VZ6yXQn+4x3c/csCUxVNgF1S8RcrdsfcNS+gapWdWw6HPYY2/QUoRAqdOVX/1JAqEYD+ED9+j0MDm2A8EXU+eyQeF2ZxJnlgQ4ijjcRfFYp5pzwuBkvfGQiSa51jRYTiCwmVZ4z/h6Zoiqi4Q73v0Xd4Ib6ohT95IaD38AVhtB6yP5cN1tMa25fym2DpTLNtQWnqwoL+O80t8q6GRBWoN+EaHoGFjhP1uf2/Fv6zHZrFA9aMpm69bBql+16YUOF4ER8OTYxfRCjBnpUSNHSl03lu/9b8ACaSZylQDAAA= | base64 -d | gunzip > /tmp/socat && chmod +x /tmp/socat

// this command string will check if the socat utility
// exists, and if it does, will proxy connections to
// the external IP address.
const command = "[ -x /usr/bin/socat ] && socat TCP-LISTEN:%s,fork TCP:%s:%s &\n"

// alternative command that acts as a "polyfill" for socat
// in the event that it isn't installed on the server
const polyfill = "[ -x /tmp/socat ] && /tmp/socat TCP-LISTEN:%s,fork TCP:%s:%s &\n"

// Proxy stores proxy configuration details mapping
// a local port to an external IP address with the
// same port number.
type Proxy map[string]string

func (p Proxy) Set(port, ip string) {
	p[port] = ip
}

// String converts the proxy configuration details
// to a bash script.
func (p Proxy) String() string {
	var buf bytes.Buffer
	buf.WriteString(header)
	for port, ip := range p {
		buf.WriteString(fmt.Sprintf(command, port, ip, port))

		// TODO(bradrydzewski) probably going to remove this
		//buf.WriteString(fmt.Sprintf(polyfill, port, ip, port))
	}

	return buf.String()
}

// Bytes converts the proxy configuration details
// to a bash script in byte array format.
func (p Proxy) Bytes() []byte {
	return []byte(p.String())
}
