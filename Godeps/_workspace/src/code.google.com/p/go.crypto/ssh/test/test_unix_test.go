// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin freebsd linux netbsd openbsd plan9

package test

// functional test harness for unix.

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"testing"
	"text/template"

	"code.google.com/p/go.crypto/ssh"
)

const sshd_config = `
Protocol 2
HostKey {{.Dir}}/ssh_host_rsa_key
HostKey {{.Dir}}/ssh_host_dsa_key
HostKey {{.Dir}}/ssh_host_ecdsa_key
Pidfile {{.Dir}}/sshd.pid
#UsePrivilegeSeparation no
KeyRegenerationInterval 3600
ServerKeyBits 768
SyslogFacility AUTH
LogLevel DEBUG2
LoginGraceTime 120
PermitRootLogin no
StrictModes no
RSAAuthentication yes
PubkeyAuthentication yes
AuthorizedKeysFile	{{.Dir}}/authorized_keys
IgnoreRhosts yes
RhostsRSAAuthentication no
HostbasedAuthentication no
`

var (
	configTmpl   template.Template
	privateKey   ssh.Signer
	hostKeyRSA   ssh.Signer
	hostKeyECDSA ssh.Signer
	hostKeyDSA   ssh.Signer
)

func init() {
	template.Must(configTmpl.Parse(sshd_config))

	for n, k := range map[string]*ssh.Signer{
		"ssh_host_ecdsa_key": &hostKeyECDSA,
		"ssh_host_rsa_key":   &hostKeyRSA,
		"ssh_host_dsa_key":   &hostKeyDSA,
	} {
		var err error
		*k, err = ssh.ParsePrivateKey([]byte(keys[n]))
		if err != nil {
			panic(fmt.Sprintf("ParsePrivateKey(%q): %v", n, err))
		}
	}

	var err error
	privateKey, err = ssh.ParsePrivateKey([]byte(testClientPrivateKey))
	if err != nil {
		panic(fmt.Sprintf("ParsePrivateKey: %v", err))
	}
}

type server struct {
	t          *testing.T
	cleanup    func() // executed during Shutdown
	configfile string
	cmd        *exec.Cmd
	output     bytes.Buffer // holds stderr from sshd process

	// Client half of the network connection.
	clientConn net.Conn
}

func username() string {
	var username string
	if user, err := user.Current(); err == nil {
		username = user.Username
	} else {
		// user.Current() currently requires cgo. If an error is
		// returned attempt to get the username from the environment.
		log.Printf("user.Current: %v; falling back on $USER", err)
		username = os.Getenv("USER")
	}
	if username == "" {
		panic("Unable to get username")
	}
	return username
}

type storedHostKey struct {
	// keys map from an algorithm string to binary key data.
	keys map[string][]byte
}

func (k *storedHostKey) Add(key ssh.PublicKey) {
	if k.keys == nil {
		k.keys = map[string][]byte{}
	}
	k.keys[key.PublicKeyAlgo()] = ssh.MarshalPublicKey(key)
}

func (k *storedHostKey) Check(addr string, remote net.Addr, algo string, key []byte) error {
	if k.keys == nil || bytes.Compare(key, k.keys[algo]) != 0 {
		return fmt.Errorf("host key mismatch. Got %q, want %q", key, k.keys[algo])
	}
	return nil
}

func clientConfig() *ssh.ClientConfig {
	keyChecker := storedHostKey{}
	keyChecker.Add(hostKeyECDSA.PublicKey())
	keyChecker.Add(hostKeyRSA.PublicKey())
	keyChecker.Add(hostKeyDSA.PublicKey())

	kc := new(keychain)
	kc.keys = append(kc.keys, privateKey)
	config := &ssh.ClientConfig{
		User: username(),
		Auth: []ssh.ClientAuth{
			ssh.ClientAuthKeyring(kc),
		},
		HostKeyChecker: &keyChecker,
	}
	return config
}

// unixConnection creates two halves of a connected net.UnixConn.  It
// is used for connecting the Go SSH client with sshd without opening
// ports.
func unixConnection() (*net.UnixConn, *net.UnixConn, error) {
	dir, err := ioutil.TempDir("", "unixConnection")
	if err != nil {
		return nil, nil, err
	}
	defer os.Remove(dir)

	addr := filepath.Join(dir, "ssh")
	listener, err := net.Listen("unix", addr)
	if err != nil {
		return nil, nil, err
	}
	defer listener.Close()
	c1, err := net.Dial("unix", addr)
	if err != nil {
		return nil, nil, err
	}

	c2, err := listener.Accept()
	if err != nil {
		c1.Close()
		return nil, nil, err
	}

	return c1.(*net.UnixConn), c2.(*net.UnixConn), nil
}

func (s *server) TryDial(config *ssh.ClientConfig) (*ssh.ClientConn, error) {
	sshd, err := exec.LookPath("sshd")
	if err != nil {
		s.t.Skipf("skipping test: %v", err)
	}

	c1, c2, err := unixConnection()
	if err != nil {
		s.t.Fatalf("unixConnection: %v", err)
	}

	s.cmd = exec.Command(sshd, "-f", s.configfile, "-i", "-e")
	f, err := c2.File()
	if err != nil {
		s.t.Fatalf("UnixConn.File: %v", err)
	}
	defer f.Close()
	s.cmd.Stdin = f
	s.cmd.Stdout = f
	s.cmd.Stderr = &s.output
	if err := s.cmd.Start(); err != nil {
		s.t.Fail()
		s.Shutdown()
		s.t.Fatalf("s.cmd.Start: %v", err)
	}
	s.clientConn = c1
	return ssh.Client(c1, config)
}

func (s *server) Dial(config *ssh.ClientConfig) *ssh.ClientConn {
	conn, err := s.TryDial(config)
	if err != nil {
		s.t.Fail()
		s.Shutdown()
		s.t.Fatalf("ssh.Client: %v", err)
	}
	return conn
}

func (s *server) Shutdown() {
	if s.cmd != nil && s.cmd.Process != nil {
		// Don't check for errors; if it fails it's most
		// likely "os: process already finished", and we don't
		// care about that. Use os.Interrupt, so child
		// processes are killed too.
		s.cmd.Process.Signal(os.Interrupt)
		s.cmd.Wait()
	}
	if s.t.Failed() {
		// log any output from sshd process
		s.t.Logf("sshd: %s", s.output.String())
	}
	s.cleanup()
}

// newServer returns a new mock ssh server.
func newServer(t *testing.T) *server {
	dir, err := ioutil.TempDir("", "sshtest")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, "sshd_config"))
	if err != nil {
		t.Fatal(err)
	}
	err = configTmpl.Execute(f, map[string]string{
		"Dir": dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	for k, v := range keys {
		f, err := os.OpenFile(filepath.Join(dir, k), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(v)); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}

	return &server{
		t:          t,
		configfile: f.Name(),
		cleanup: func() {
			if err := os.RemoveAll(dir); err != nil {
				t.Error(err)
			}
		},
	}
}

// keychain implements the ClientKeyring interface.
type keychain struct {
	keys []ssh.Signer
}

func (k *keychain) Key(i int) (ssh.PublicKey, error) {
	if i < 0 || i >= len(k.keys) {
		return nil, nil
	}
	return k.keys[i].PublicKey(), nil
}

func (k *keychain) Sign(i int, rand io.Reader, data []byte) (sig []byte, err error) {
	return k.keys[i].Sign(rand, data)
}

func (k *keychain) loadPEM(file string) error {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	key, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		return err
	}
	k.keys = append(k.keys, key)
	return nil
}
