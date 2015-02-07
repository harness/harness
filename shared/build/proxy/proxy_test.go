package proxy

import (
	"testing"
)

func TestProxy(t *testing.T) {
	// test creating a proxy with a few different
	// addresses, and our ability to create the
	// proxy shell script.
	p := Proxy{}
	p.Set("8080", "172.1.4.5")
	b := p.Bytes()

	expected := header + `[ -x /usr/bin/socat ] && socat TCP-LISTEN:8080,fork TCP:172.1.4.5:8080 &
[ -x /tmp/socat ] && /tmp/socat TCP-LISTEN:8080,fork TCP:172.1.4.5:8080 &
`
	if string(b) != expected {
		t.Errorf("AInvalid proxy \n%s", string(b))
	}

	// test creating a proxy script when there
	// are no proxy addresses added to the map
	p = Proxy{}
	b = p.Bytes()
	if string(b) != header {
		t.Errorf("Invalid empty proxy file. Expected\n%s", header)
	}
}
