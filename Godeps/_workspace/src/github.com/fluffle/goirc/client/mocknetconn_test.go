package client

import (
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	mockReadCloser = iota
	mockInCloser
	mockOutCloser
)

type mockNetConn struct {
	*testing.T

	In, Out chan string
	in, out chan []byte
	closers [3]chan bool
	rc      chan bool

	closed bool
	rt, wt time.Time
}

func MockNetConn(t *testing.T) *mockNetConn {
	// Our mock connection is a testing object
	m := &mockNetConn{T: t}

	// buffer input
	m.In = make(chan string, 20)
	m.in = make(chan []byte)
	m.closers[mockInCloser] = make(chan bool, 1)
	go func() {
		for {
			select {
			case <-m.closers[mockInCloser]:
				return
			case s := <-m.In:
				m.in <- []byte(s)
			}
		}
	}()

	// buffer output
	m.Out = make(chan string)
	m.out = make(chan []byte, 20)
	m.closers[mockOutCloser] = make(chan bool, 1)
	go func() {
		for {
			select {
			case <-m.closers[mockOutCloser]:
				return
			case b := <-m.out:
				m.Out <- string(b)
			}
		}
	}()

	// Set up channel to force EOF to Read() on close.
	m.closers[mockReadCloser] = make(chan bool, 1)

	return m
}

// Test helpers
func (m *mockNetConn) Send(s string) {
	m.In <- s + "\r\n"
}

func (m *mockNetConn) Expect(e string) {
	select {
	case <-time.After(1e6):
		m.Errorf("Mock connection did not receive expected output.\n\t"+
			"Expected: '%s', got nothing.", e)
	case s := <-m.Out:
		s = strings.Trim(s, "\r\n")
		if e != s {
			m.Errorf("Mock connection received unexpected value.\n\t"+
				"Expected: '%s'\n\tGot: '%s'", e, s)
		}
	}
}

func (m *mockNetConn) ExpectNothing() {
	select {
	case <-time.After(1e6):
	case s := <-m.Out:
		s = strings.Trim(s, "\r\n")
		m.Errorf("Mock connection received unexpected output.\n\t"+
			"Expected nothing, got: '%s'", s)
	}
}

// Implement net.Conn interface
func (m *mockNetConn) Read(b []byte) (int, error) {
	if m.closed {
		return 0, os.ErrInvalid
	}
	l := 0
	select {
	case s := <-m.in:
		l = len(s)
		copy(b, s)
	case <-m.closers[mockReadCloser]:
		return 0, io.EOF
	}
	return l, nil
}

func (m *mockNetConn) Write(s []byte) (int, error) {
	if m.closed {
		return 0, os.ErrInvalid
	}
	b := make([]byte, len(s))
	copy(b, s)
	m.out <- b
	return len(s), nil
}

func (m *mockNetConn) Close() error {
	if m.closed {
		return os.ErrInvalid
	}
	// Shut down *ALL* the goroutines!
	// This will trigger an EOF event in Read() too
	for _, c := range m.closers {
		c <- true
	}
	m.closed = true
	return nil
}

func (m *mockNetConn) LocalAddr() net.Addr {
	return &net.IPAddr{net.IPv4(127, 0, 0, 1)}
}

func (m *mockNetConn) RemoteAddr() net.Addr {
	return &net.IPAddr{net.IPv4(127, 0, 0, 1)}
}

func (m *mockNetConn) SetDeadline(t time.Time) error {
	m.rt = t
	m.wt = t
	return nil
}

func (m *mockNetConn) SetReadDeadline(t time.Time) error {
	m.rt = t
	return nil
}

func (m *mockNetConn) SetWriteDeadline(t time.Time) error {
	m.wt = t
	return nil
}
