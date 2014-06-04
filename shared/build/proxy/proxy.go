package proxy

import (
	"bytes"
	"fmt"
)

// bash header
const header = "#!/bin/bash\n"

// this command string will check if the socat utility
// exists, and if it does, will proxy connections to
// the external IP address.
const command = "[ -x /usr/bin/socat ] && socat TCP-LISTEN:%s,fork TCP:%s:%s &\n"

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
	}

	return buf.String()
}

// Bytes converts the proxy configuration details
// to a bash script in byte array format.
func (p Proxy) Bytes() []byte {
	return []byte(p.String())
}
