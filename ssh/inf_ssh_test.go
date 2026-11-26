// Copyright 2026 Harness, Inc.
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

package ssh

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/services/publickey"
	gitnesserrors "github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"

	"github.com/gliderlabs/ssh"
	"github.com/stretchr/testify/require"
	gossh "golang.org/x/crypto/ssh"
)

// infAddr is a net.Addr with a fixed string.
type infAddr string

func (a infAddr) Network() string { return "tcp" }
func (a infAddr) String() string  { return string(a) }

// infCtx is a minimal ssh.Context.
type infCtx struct {
	context.Context
	sync.Mutex

	user      string
	sessionID string
	values    map[any]any
}

func infNewCtx(user, sessionID string) *infCtx {
	return &infCtx{Context: context.Background(), user: user, sessionID: sessionID, values: map[any]any{}}
}

func (c *infCtx) User() string                  { return c.user }
func (c *infCtx) SessionID() string             { return c.sessionID }
func (c *infCtx) ClientVersion() string         { return "SSH-2.0-client" }
func (c *infCtx) ServerVersion() string         { return "SSH-2.0-server" }
func (c *infCtx) RemoteAddr() net.Addr          { return infAddr("203.0.113.9:4000") }
func (c *infCtx) LocalAddr() net.Addr           { return infAddr("127.0.0.1:22") }
func (c *infCtx) Permissions() *ssh.Permissions { return &ssh.Permissions{} }
func (c *infCtx) SetValue(key, value any) {
	c.Lock()
	defer c.Unlock()
	c.values[key] = value
}

func (c *infCtx) Value(key any) any {
	c.Lock()
	defer c.Unlock()
	if v, ok := c.values[key]; ok {
		return v
	}
	return c.Context.Value(key)
}

// infSession is a minimal ssh.Session writing to buffers.
type infSession struct {
	ssh.Session

	ctx     *infCtx
	rawCmd  string
	cmd     []string
	environ []string
	stdout  bytes.Buffer
	stderr  bytes.Buffer
}

func (s *infSession) Context() ssh.Context        { return s.ctx }
func (s *infSession) User() string                { return s.ctx.user }
func (s *infSession) RemoteAddr() net.Addr        { return s.ctx.RemoteAddr() }
func (s *infSession) RawCommand() string          { return s.rawCmd }
func (s *infSession) Command() []string           { return s.cmd }
func (s *infSession) Environ() []string           { return s.environ }
func (s *infSession) Write(p []byte) (int, error) { return s.stdout.Write(p) }
func (s *infSession) Stderr() io.ReadWriter       { return &s.stderr }

func TestInfGetRepoRefFromCommand(t *testing.T) {
	tests := map[string]string{
		"'space/repo.git'":      "space/repo",
		"space/repo.git":        "space/repo",
		"'space/sub/repo'":      "space/sub/repo",
		"repo":                  "repo",
		"'weird.git.git'":       "weird.git",
		"'space/repo.git/more'": "space/repo.git/more",
	}
	for in, want := range tests {
		require.Equal(t, want, getRepoRefFromCommand(in), in)
	}
}

func TestInfGetRequestID(t *testing.T) {
	require.Equal(t, "short", getRequestID("short"))
	long := strings.Repeat("a", 30)
	require.Equal(t, strings.Repeat("a", 20), getRequestID(long))
	require.Equal(t, strings.Repeat("b", 20), getRequestID(strings.Repeat("b", 20)))
}

func TestInfLoggerWithRequestID(t *testing.T) {
	logger := getLoggerWithRequestID(strings.Repeat("z", 40))
	require.NotNil(t, logger)

	ctx := infNewCtx("u", "sid")
	require.NotNil(t, getLogger(ctx))
	ctx.SetValue(loggerKey, logger)
	got := getLogger(ctx)
	require.Equal(t, logger, got)
}

func TestInfServerSanitize(t *testing.T) {
	s := &Server{}
	require.Error(t, s.sanitize(), "a repository controller is required")

	s = &Server{RepoCtrl: &repo.Controller{}}
	require.NoError(t, s.sanitize())
	require.Equal(t, 22, s.Port)
	require.Equal(t, defaultCiphers, s.Ciphers)
	require.Equal(t, defaultKeyExchanges, s.KeyExchanges)
	require.Equal(t, defaultMACs, s.MACs)
	require.NotZero(t, s.KeepAliveInterval)

	s = &Server{
		RepoCtrl: &repo.Controller{}, Port: 2222, Ciphers: []string{"c"}, KeyExchanges: []string{"k"},
		MACs: []string{"m"}, KeepAliveInterval: time.Second,
	}
	require.NoError(t, s.sanitize())
	require.Equal(t, 2222, s.Port)
	require.Equal(t, []string{"c"}, s.Ciphers)
	require.Equal(t, []string{"k"}, s.KeyExchanges)
	require.Equal(t, []string{"m"}, s.MACs)
	require.Equal(t, time.Second, s.KeepAliveInterval)

	require.ErrorContains(t, (&Server{}).ListenAndServe(), "failed to sanitize server defaults")
}

func TestInfCreateKeyIfNotExists(t *testing.T) {
	dir := t.TempDir()

	t.Run("generates a key pair once", func(t *testing.T) {
		path := filepath.Join(dir, "nested", "dir", "host_key")
		require.NoError(t, CreateKeyIfNotExists(path))

		privateData, err := os.ReadFile(path)
		require.NoError(t, err)
		signer, err := gossh.ParsePrivateKey(privateData)
		require.NoError(t, err)

		publicData, err := os.ReadFile(path + ".pub")
		require.NoError(t, err)
		pub, _, _, _, err := gossh.ParseAuthorizedKey(publicData)
		require.NoError(t, err)
		require.Equal(t, signer.PublicKey().Marshal(), pub.Marshal())

		info, err := os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o600), info.Mode().Perm())

		// a second call leaves the existing key untouched.
		require.NoError(t, CreateKeyIfNotExists(path))
		again, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, privateData, again)
	})
	t.Run("existing path is a no-op", func(t *testing.T) {
		existing := filepath.Join(dir, "existing")
		require.NoError(t, os.WriteFile(existing, []byte("not a key"), 0o600))
		require.NoError(t, CreateKeyIfNotExists(existing))
		data, err := os.ReadFile(existing)
		require.NoError(t, err)
		require.Equal(t, "not a key", string(data))
	})
	t.Run("stat errors other than not-exist are returned", func(t *testing.T) {
		file := filepath.Join(dir, "plain-file")
		require.NoError(t, os.WriteFile(file, nil, 0o600))
		err := CreateKeyIfNotExists(filepath.Join(file, "child"))
		require.ErrorContains(t, err, "failed to check for for existence of key")
	})
	t.Run("directory creation failures are returned", func(t *testing.T) {
		file := filepath.Join(dir, "blocker")
		require.NoError(t, os.WriteFile(file, nil, 0o600))
		// the parent is a file, but the stat of the nested path reports ENOTDIR, which is a stat error.
		require.Error(t, CreateKeyIfNotExists(filepath.Join(file, "sub", "key")))
	})
}

func TestInfGenerateKeyPair_UnwritablePath(t *testing.T) {
	err := GenerateKeyPair(filepath.Join(t.TempDir(), "missing-dir", "key"))
	require.Error(t, err)
}

func TestInfSetupHostKeys(t *testing.T) {
	t.Run("missing configured host key", func(t *testing.T) {
		s := &Server{internal: &ssh.Server{}, HostKeys: []string{filepath.Join(t.TempDir(), "nope")}}
		require.ErrorContains(t, s.setupHostKeys(), "failed to read provided host key")
	})
	t.Run("unusable default key path", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(file, nil, 0o600))
		s := &Server{internal: &ssh.Server{}, ServerKeyPath: filepath.Join(file, "key")}
		require.ErrorContains(t, s.setupHostKeys(), "failed to setup default key")
	})
	t.Run("configured host keys are loaded and bad ones are only logged", func(t *testing.T) {
		keyPath := filepath.Join(t.TempDir(), "bad-key")
		require.NoError(t, os.WriteFile(keyPath, []byte("not a pem key"), 0o600))
		s := &Server{internal: &ssh.Server{}, HostKeys: []string{keyPath}}
		require.NoError(t, s.setupHostKeys())
	})
}

func TestInfChainMiddleware(t *testing.T) {
	var order []string
	mw := func(name string) Middleware {
		return func(next ssh.Handler) ssh.Handler {
			return func(s ssh.Session) {
				order = append(order, name+":before")
				next(s)
				order = append(order, name+":after")
			}
		}
	}
	handler := ChainMiddleware(func(ssh.Session) { order = append(order, "handler") }, mw("a"), mw("b"))
	handler(nil)
	require.Equal(t, []string{"a:before", "b:before", "handler", "b:after", "a:after"}, order)

	var pkOrder []string
	pk := func(name string, allow bool) PublicKeyMiddleware {
		return func(next ssh.PublicKeyHandler) ssh.PublicKeyHandler {
			return func(ctx ssh.Context, key ssh.PublicKey) bool {
				pkOrder = append(pkOrder, name)
				return allow && next(ctx, key)
			}
		}
	}
	pkHandler := ChainPublicKeyMiddleware(func(ssh.Context, ssh.PublicKey) bool {
		pkOrder = append(pkOrder, "handler")
		return true
	}, pk("a", true), pk("b", true))
	require.True(t, pkHandler(nil, nil))
	require.Equal(t, []string{"a", "b", "handler"}, pkOrder)

	pkOrder = nil
	require.False(t, ChainPublicKeyMiddleware(func(ssh.Context, ssh.PublicKey) bool { return true },
		pk("deny", false))(nil, nil))
}

func TestInfPanicRecoverMiddleware(t *testing.T) {
	sess := &infSession{ctx: infNewCtx("alice", "sid")}
	PanicRecoverMiddleware(func(ssh.Session) { panic("kaboom") })(sess)
	require.Contains(t, sess.stdout.String(), "Internal server error")

	sess = &infSession{ctx: infNewCtx("alice", "sid")}
	PanicRecoverMiddleware(func(s ssh.Session) { _, _ = s.Write([]byte("ok")) })(sess)
	require.Equal(t, "ok", sess.stdout.String())
}

func TestInfAccessLogAndRequestIDMiddleware(t *testing.T) {
	sess := &infSession{ctx: infNewCtx("alice", strings.Repeat("s", 40)), cmd: []string{"git-upload-pack", "repo"}}
	called := false
	ChainMiddleware(func(s ssh.Session) {
		called = true
		require.NotNil(t, s.Context().Value(loggerKey), "request logger is stored on the context")
	}, HLogRequestIDHandler, HLogAccessLogHandler)(sess)
	require.True(t, called)
	reqID, ok := request.RequestIDFrom(sess.ctx)
	require.True(t, ok)
	require.Equal(t, strings.Repeat("s", 20), reqID)
}

type infKey struct {
	keyType string
}

func (k infKey) Type() string                          { return k.keyType }
func (k infKey) Marshal() []byte                       { return []byte(k.keyType) }
func (k infKey) Verify([]byte, *gossh.Signature) error { return nil }

type infVerifier struct {
	publickey.SSHAuthService

	principal *types.PrincipalInfo
	err       error
	gotUser   string
}

func (v *infVerifier) ValidateKey(_ context.Context, username string, _ ssh.PublicKey) (*types.PrincipalInfo, error) {
	v.gotUser = username
	return v.principal, v.err
}

func infEd25519(t *testing.T) gossh.PublicKey {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	key, err := gossh.NewPublicKey(pub)
	require.NoError(t, err)
	return key
}

func TestInfPublicKeyHandler(t *testing.T) {
	key := infEd25519(t)
	principal := &types.PrincipalInfo{ID: 5, UID: "alice"}

	t.Run("accepts a verified key and stores the principal", func(t *testing.T) {
		v := &infVerifier{principal: principal}
		s := &Server{Verifier: v, DefaultUser: "git"}
		ctx := infNewCtx("git", "sid")
		require.True(t, s.publicKeyHandler(ctx, key))
		require.Equal(t, "git", v.gotUser)
		require.Equal(t, principal, ctx.Value(principalKey))
	})
	t.Run("any user name is accepted without a default user", func(t *testing.T) {
		s := &Server{Verifier: &infVerifier{principal: principal}}
		require.True(t, s.publicKeyHandler(infNewCtx("whoever", "sid"), key))
	})
	t.Run("wrong user name", func(t *testing.T) {
		v := &infVerifier{principal: principal}
		s := &Server{Verifier: v, DefaultUser: "git"}
		require.False(t, s.publicKeyHandler(infNewCtx("root", "sid"), key))
		require.Empty(t, v.gotUser, "the verifier must not be consulted")
	})
	t.Run("disallowed key type", func(t *testing.T) {
		v := &infVerifier{principal: principal}
		s := &Server{Verifier: v}
		require.False(t, s.publicKeyHandler(infNewCtx("git", "sid"), infKey{keyType: gossh.KeyAlgoDSA}))
		require.Empty(t, v.gotUser)
	})
	t.Run("unknown key", func(t *testing.T) {
		s := &Server{Verifier: &infVerifier{err: gitnesserrors.NotFound("no such key")}}
		require.False(t, s.publicKeyHandler(infNewCtx("git", "sid"), key))
	})
	t.Run("verifier failure", func(t *testing.T) {
		s := &Server{Verifier: &infVerifier{err: errors.New("db down")}}
		require.False(t, s.publicKeyHandler(infNewCtx("git", "sid"), key))
	})
}

func TestInfPublicKeyHandler_Certificates(t *testing.T) {
	principal := &types.PrincipalInfo{ID: 5, UID: "alice"}
	_, caPriv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	caSigner, err := gossh.NewSignerFromKey(caPriv)
	require.NoError(t, err)

	newCert := func(certType uint32, principals ...string) *gossh.Certificate {
		cert := &gossh.Certificate{
			Key:             infEd25519(t),
			CertType:        certType,
			ValidPrincipals: principals,
			ValidBefore:     gossh.CertTimeInfinity,
		}
		require.NoError(t, cert.SignCert(rand.Reader, caSigner))
		return cert
	}

	t.Run("rejected without trusted CAs", func(t *testing.T) {
		s := &Server{Verifier: &infVerifier{principal: principal}}
		require.False(t, s.publicKeyHandler(infNewCtx("git", "sid"), newCert(gossh.UserCert, "alice")))
	})
	t.Run("host certificates are rejected", func(t *testing.T) {
		s := &Server{Verifier: &infVerifier{principal: principal}, TrustedUserCAKeys: []string{"ca"}}
		require.False(t, s.publicKeyHandler(infNewCtx("git", "sid"), newCert(gossh.HostCert, "alice")))
	})
	t.Run("certificate not valid for the principal", func(t *testing.T) {
		s := &Server{Verifier: &infVerifier{principal: principal}, TrustedUserCAKeys: []string{"ca"}}
		require.False(t, s.publicKeyHandler(infNewCtx("git", "sid"), newCert(gossh.UserCert, "someone-else")))
	})
}

func TestInfSendKeepAliveMsg(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sess := &infKeepAliveSession{infSession: infSession{ctx: infNewCtx("u", "sid")}}
	done := make(chan struct{})
	go func() {
		sendKeepAliveMsg(ctx, sess, 5*time.Millisecond)
		close(done)
	}()
	require.Eventually(t, func() bool { return sess.requests() >= 2 }, 5*time.Second, time.Millisecond)
	cancel()
	<-done
	require.Equal(t, KeepAliveMsg, sess.lastName())
}

type infKeepAliveSession struct {
	infSession

	mx    sync.Mutex
	names []string
}

func (s *infKeepAliveSession) SendRequest(name string, _ bool, _ []byte) (bool, error) {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.names = append(s.names, name)
	return true, errors.New("client gone")
}

func (s *infKeepAliveSession) requests() int {
	s.mx.Lock()
	defer s.mx.Unlock()
	return len(s.names)
}

func (s *infKeepAliveSession) lastName() string {
	s.mx.Lock()
	defer s.mx.Unlock()
	return s.names[len(s.names)-1]
}

func TestInfLogPublicKeyMiddleware(t *testing.T) {
	ctx := infNewCtx("alice", strings.Repeat("k", 40))
	var seen bool
	handler := LogPublicKeyMiddleware(func(ssh.Context, ssh.PublicKey) bool {
		seen = true
		return true
	})
	require.True(t, handler(ctx, infKey{keyType: "ssh-ed25519"}))
	require.True(t, seen)
	reqID, ok := request.RequestIDFrom(ctx)
	require.True(t, ok)
	require.Equal(t, strings.Repeat("k", 20), reqID)

	deny := LogPublicKeyMiddleware(func(ssh.Context, ssh.PublicKey) bool { return false })
	require.False(t, deny(ctx, infKey{keyType: "ssh-ed25519"}))
}

func TestInfSessionHandler_EarlyExits(t *testing.T) {
	principal := &types.PrincipalInfo{ID: 5, UID: "alice"}
	tests := []struct {
		name      string
		rawCmd    string
		principal bool
		want      string
	}{
		{name: "no principal", rawCmd: "git-upload-pack 'space/repo.git'", want: "principal not found"},
		{name: "missing argument", rawCmd: "git-upload-pack", principal: true, want: "must have an argument"},
		{name: "empty command", rawCmd: "", principal: true, want: "must have an argument"},
		{name: "unsupported command", rawCmd: "rm -rf /", principal: true, want: "command not supported"},
		{
			name: "lfs transfer is not supported", rawCmd: "git-lfs-transfer 'space/repo.git' download",
			principal: true, want: "git-lfs-transfer is not supported.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sess := &infSession{ctx: infNewCtx("alice", "sid"), rawCmd: tt.rawCmd}
			if tt.principal {
				sess.ctx.SetValue(principalKey, principal)
			}
			(&Server{}).sessionHandler(sess)
			require.Contains(t, sess.stderr.String(), tt.want)
			require.Empty(t, sess.stdout.String())
		})
	}
}
