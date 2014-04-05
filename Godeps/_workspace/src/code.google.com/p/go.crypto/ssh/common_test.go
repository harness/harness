// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssh

import (
	"io"
	"net"
	"testing"
)

func TestSafeString(t *testing.T) {
	strings := map[string]string{
		"\x20\x0d\x0a":  "\x20\x0d\x0a",
		"flibble":       "flibble",
		"new\x20line":   "new\x20line",
		"123456\x07789": "123456 789",
		"\t\t\x10\r\n":  "\t\t \r\n",
	}

	for s, expected := range strings {
		actual := safeString(s)
		if expected != actual {
			t.Errorf("expected: %v, actual: %v", []byte(expected), []byte(actual))
		}
	}
}

// Make sure Read/Write are not exposed.
func TestConnHideRWMethods(t *testing.T) {
	for _, c := range []interface{}{new(ServerConn), new(ClientConn)} {
		if _, ok := c.(io.Reader); ok {
			t.Errorf("%T implements io.Reader", c)
		}
		if _, ok := c.(io.Writer); ok {
			t.Errorf("%T implements io.Writer", c)
		}
	}
}

func TestConnSupportsLocalRemoteMethods(t *testing.T) {
	type LocalAddr interface {
		LocalAddr() net.Addr
	}
	type RemoteAddr interface {
		RemoteAddr() net.Addr
	}
	for _, c := range []interface{}{new(ServerConn), new(ClientConn)} {
		if _, ok := c.(LocalAddr); !ok {
			t.Errorf("%T does not implement LocalAddr", c)
		}
		if _, ok := c.(RemoteAddr); !ok {
			t.Errorf("%T does not implement RemoteAddr", c)
		}
	}
}
