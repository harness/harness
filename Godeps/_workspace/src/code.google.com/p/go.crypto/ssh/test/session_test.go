// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !windows

package test

// Session functional tests.

import (
	"bytes"
	"code.google.com/p/go.crypto/ssh"
	"io"
	"strings"
	"testing"
)

func TestRunCommandSuccess(t *testing.T) {
	server := newServer(t)
	defer server.Shutdown()
	conn := server.Dial(clientConfig())
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		t.Fatalf("session failed: %v", err)
	}
	defer session.Close()
	err = session.Run("true")
	if err != nil {
		t.Fatalf("session failed: %v", err)
	}
}

func TestHostKeyCheck(t *testing.T) {
	server := newServer(t)
	defer server.Shutdown()

	conf := clientConfig()
	k := conf.HostKeyChecker.(*storedHostKey)

	// change the keys.
	k.keys[ssh.KeyAlgoRSA][25]++
	k.keys[ssh.KeyAlgoDSA][25]++
	k.keys[ssh.KeyAlgoECDSA256][25]++

	conn, err := server.TryDial(conf)
	if err == nil {
		conn.Close()
		t.Fatalf("dial should have failed.")
	} else if !strings.Contains(err.Error(), "host key mismatch") {
		t.Fatalf("'host key mismatch' not found in %v", err)
	}
}

func TestRunCommandFailed(t *testing.T) {
	server := newServer(t)
	defer server.Shutdown()
	conn := server.Dial(clientConfig())
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		t.Fatalf("session failed: %v", err)
	}
	defer session.Close()
	err = session.Run(`bash -c "kill -9 $$"`)
	if err == nil {
		t.Fatalf("session succeeded: %v", err)
	}
}

func TestRunCommandWeClosed(t *testing.T) {
	server := newServer(t)
	defer server.Shutdown()
	conn := server.Dial(clientConfig())
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		t.Fatalf("session failed: %v", err)
	}
	err = session.Shell()
	if err != nil {
		t.Fatalf("shell failed: %v", err)
	}
	err = session.Close()
	if err != nil {
		t.Fatalf("shell failed: %v", err)
	}
}

func TestFuncLargeRead(t *testing.T) {
	server := newServer(t)
	defer server.Shutdown()
	conn := server.Dial(clientConfig())
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		t.Fatalf("unable to create new session: %s", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		t.Fatalf("unable to acquire stdout pipe: %s", err)
	}

	err = session.Start("dd if=/dev/urandom bs=2048 count=1")
	if err != nil {
		t.Fatalf("unable to execute remote command: %s", err)
	}

	buf := new(bytes.Buffer)
	n, err := io.Copy(buf, stdout)
	if err != nil {
		t.Fatalf("error reading from remote stdout: %s", err)
	}

	if n != 2048 {
		t.Fatalf("Expected %d bytes but read only %d from remote command", 2048, n)
	}
}

func TestInvalidTerminalMode(t *testing.T) {
	server := newServer(t)
	defer server.Shutdown()
	conn := server.Dial(clientConfig())
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		t.Fatalf("session failed: %v", err)
	}
	defer session.Close()

	if err = session.RequestPty("vt100", 80, 40, ssh.TerminalModes{255: 1984}); err == nil {
		t.Fatalf("req-pty failed: successful request with invalid mode")
	}
}

func TestValidTerminalMode(t *testing.T) {
	server := newServer(t)
	defer server.Shutdown()
	conn := server.Dial(clientConfig())
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		t.Fatalf("session failed: %v", err)
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		t.Fatalf("unable to acquire stdout pipe: %s", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		t.Fatalf("unable to acquire stdin pipe: %s", err)
	}

	tm := ssh.TerminalModes{ssh.ECHO: 0}
	if err = session.RequestPty("xterm", 80, 40, tm); err != nil {
		t.Fatalf("req-pty failed: %s", err)
	}

	err = session.Shell()
	if err != nil {
		t.Fatalf("session failed: %s", err)
	}

	stdin.Write([]byte("stty -a && exit\n"))

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, stdout); err != nil {
		t.Fatalf("reading failed: %s", err)
	}

	if sttyOutput := buf.String(); !strings.Contains(sttyOutput, "-echo ") {
		t.Fatalf("terminal mode failure: expected -echo in stty output, got %s", sttyOutput)
	}
}
