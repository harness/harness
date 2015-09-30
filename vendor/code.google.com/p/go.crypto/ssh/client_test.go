package ssh

import (
	"net"
	"testing"
)

func testClientVersion(t *testing.T, config *ClientConfig, expected string) {
	clientConn, serverConn := net.Pipe()
	receivedVersion := make(chan string, 1)
	go func() {
		version, err := readVersion(serverConn)
		if err != nil {
			receivedVersion <- ""
		} else {
			receivedVersion <- string(version)
		}
		serverConn.Close()
	}()
	Client(clientConn, config)
	actual := <-receivedVersion
	if actual != expected {
		t.Fatalf("got %s; want %s", actual, expected)
	}
}

func TestCustomClientVersion(t *testing.T) {
	version := "Test-Client-Version-0.0"
	testClientVersion(t, &ClientConfig{ClientVersion: version}, version)
}

func TestDefaultClientVersion(t *testing.T) {
	testClientVersion(t, &ClientConfig{}, packageVersion)
}
