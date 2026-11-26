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
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/controller/repo"
	gitnesserrors "github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"

	"github.com/gliderlabs/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gossh "golang.org/x/crypto/ssh"
)

const gtxTimeout = 10 * time.Second

// gtxVerifier is a fake of publickey.SSHAuthService.
type gtxVerifier struct {
	mu        sync.Mutex
	principal *types.PrincipalInfo
	err       error
	calls     []string
}

func (v *gtxVerifier) ValidateKey(_ context.Context, username string, _ ssh.PublicKey) (*types.PrincipalInfo, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.calls = append(v.calls, username)

	return v.principal, v.err
}

func (v *gtxVerifier) callCount() int {
	v.mu.Lock()
	defer v.mu.Unlock()

	return len(v.calls)
}

func gtxNewSigner(t *testing.T) gossh.Signer {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	signer, err := gossh.NewSignerFromKey(priv)
	require.NoError(t, err)

	return signer
}

// gtxWriteHostKey writes a new ed25519 private key (PKCS#8 PEM) and returns the file path and the public key.
func gtxWriteHostKey(t *testing.T) (string, gossh.PublicKey) {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	der, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "host_key")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600))

	signer, err := gossh.NewSignerFromKey(priv)
	require.NoError(t, err)

	return path, signer.PublicKey()
}

func gtxFreePort(t *testing.T) int {
	t.Helper()

	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = l.Close() }()

	addr, ok := l.Addr().(*net.TCPAddr)
	require.True(t, ok)

	return addr.Port
}

func gtxDial(
	t *testing.T, port int, user string, hostKey gossh.PublicKey, auth ...gossh.AuthMethod,
) (*gossh.Client, error) {
	t.Helper()

	return gossh.Dial("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), &gossh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: gossh.FixedHostKey(hostKey),
		Timeout:         gtxTimeout,
	})
}

// gtxDialEventually retries to connect while the server is starting.
func gtxDialEventually(
	t *testing.T, port int, user string, hostKey gossh.PublicKey, auth ...gossh.AuthMethod,
) *gossh.Client {
	t.Helper()

	var client *gossh.Client
	require.Eventually(t, func() bool {
		c, err := gtxDial(t, port, user, hostKey, auth...)
		if err != nil {
			return false
		}
		client = c

		return true
	}, gtxTimeout, 20*time.Millisecond)
	t.Cleanup(func() { _ = client.Close() })

	return client
}

func gtxRun(t *testing.T, client *gossh.Client, command string) (string, error) {
	t.Helper()

	session, err := client.NewSession()
	require.NoError(t, err)
	defer func() { _ = session.Close() }()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	err = session.Run(command)

	return stdout.String() + stderr.String(), err
}

// gtxServe serves the session and public key handlers of the server on a free port, configured like ListenAndServe
// does it, and returns the port and the host key. The listener is created up front, so there are no data races on
// the server's internals as there would be when running ListenAndServe in a goroutine.
func gtxServe(t *testing.T, s *Server) (int, gossh.PublicKey) {
	t.Helper()

	hostKeyPath, hostKey := gtxWriteHostKey(t)

	srv := &ssh.Server{
		Handler: ChainMiddleware(
			s.sessionHandler,
			PanicRecoverMiddleware,
			HLogRequestIDHandler,
			HLogAccessLogHandler,
		),
		PublicKeyHandler: ChainPublicKeyMiddleware(s.publicKeyHandler, LogPublicKeyMiddleware),
		PtyCallback:      func(ssh.Context, ssh.Pty) bool { return false },
	}
	require.NoError(t, srv.SetOption(ssh.HostKeyFile(hostKeyPath)))

	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() { _ = srv.Serve(l) }()
	t.Cleanup(func() { _ = srv.Close() })

	addr, ok := l.Addr().(*net.TCPAddr)
	require.True(t, ok)

	return addr.Port, hostKey
}

func TestGtxServer_Sanitize(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		s := &Server{RepoCtrl: &repo.Controller{}}
		require.NoError(t, s.sanitize())

		assert.Equal(t, 22, s.Port)
		assert.Equal(t, defaultCiphers, s.Ciphers)
		assert.Equal(t, defaultKeyExchanges, s.KeyExchanges)
		assert.Equal(t, defaultMACs, s.MACs)
		assert.Equal(t, time.Duration(5000), s.KeepAliveInterval)
	})

	t.Run("configured values are kept", func(t *testing.T) {
		s := &Server{
			Port: 2222, Ciphers: []string{"c"}, KeyExchanges: []string{"k"}, MACs: []string{"m"},
			KeepAliveInterval: time.Second, RepoCtrl: &repo.Controller{},
		}
		require.NoError(t, s.sanitize())

		assert.Equal(t, 2222, s.Port)
		assert.Equal(t, []string{"c"}, s.Ciphers)
		assert.Equal(t, []string{"k"}, s.KeyExchanges)
		assert.Equal(t, []string{"m"}, s.MACs)
		assert.Equal(t, time.Second, s.KeepAliveInterval)
	})

	t.Run("repository controller is required", func(t *testing.T) {
		err := (&Server{}).sanitize()
		require.Error(t, err)
		assert.True(t, gitnesserrors.IsInvalidArgument(err), "unexpected error: %v", err)
	})

	t.Run("listen and serve fails for an invalid server", func(t *testing.T) {
		err := (&Server{}).ListenAndServe()
		require.ErrorContains(t, err, "failed to sanitize server defaults")
	})
}

func TestGtxGetRepoRefFromCommand(t *testing.T) {
	tests := map[string]string{
		"'space/repo.git'":         "space/repo",
		"space/repo.git":           "space/repo",
		"'space/repo'":             "space/repo",
		"space/sub/repo.git":       "space/sub/repo",
		"'repo.git'":               "repo",
		"":                         "",
		"'only.git.git'":           "only.git",
		"/space/repo.git":          "/space/repo",
		"'space/repo.github.git'":  "space/repo.github",
		"'space/repo.git.backup'":  "space/repo.git.backup",
		"''":                       "",
		"'unbalanced/repo.git":     "unbalanced/repo",
		"unbalanced/repo.git'":     "unbalanced/repo",
		"'space/with space.git'":   "space/with space",
		"'space/with-dash_us.git'": "space/with-dash_us",
	}

	for in, want := range tests {
		assert.Equal(t, want, getRepoRefFromCommand(in), in)
	}
}

func TestGtxGetRequestID(t *testing.T) {
	assert.Equal(t, "", getRequestID(""))
	assert.Equal(t, "short", getRequestID("short"))
	assert.Equal(t, strings.Repeat("a", 20), getRequestID(strings.Repeat("a", 20)))
	assert.Equal(t, strings.Repeat("a", 20), getRequestID(strings.Repeat("a", 40)))
}

func TestGtxKeyGeneration(t *testing.T) {
	t.Run("existing key is kept", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "key")
		require.NoError(t, os.WriteFile(path, []byte("existing"), 0o600))

		require.NoError(t, CreateKeyIfNotExists(path))

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "existing", string(content))
	})

	t.Run("missing key is generated including the directory", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nested", "dir", "key")

		require.NoError(t, CreateKeyIfNotExists(path))

		signer, err := gossh.ParsePrivateKey(gtxReadFile(t, path))
		require.NoError(t, err)

		pub, _, _, _, err := gossh.ParseAuthorizedKey(gtxReadFile(t, path+".pub"))
		require.NoError(t, err)
		assert.Equal(t, signer.PublicKey().Marshal(), pub.Marshal())
		assert.Equal(t, "ssh-rsa", pub.Type())

		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	})

	t.Run("the path can't be checked", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "blocker")
		require.NoError(t, os.WriteFile(blocker, nil, 0o600))

		err := CreateKeyIfNotExists(filepath.Join(blocker, "key"))
		require.Error(t, err)
	})

	t.Run("the directory can't be created", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "blocker")
		require.NoError(t, os.WriteFile(blocker, nil, 0o600))

		err := CreateKeyIfNotExists(filepath.Join(blocker, "sub", "key"))
		require.Error(t, err)
	})

	t.Run("key can't be written", func(t *testing.T) {
		err := GenerateKeyPair(filepath.Join(t.TempDir(), "missing-dir", "key"))
		require.Error(t, err)
	})
}

func gtxReadFile(t *testing.T, path string) []byte {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err)

	return content
}

func TestGtxServer_SetupHostKeys(t *testing.T) {
	t.Run("provided host keys must exist", func(t *testing.T) {
		s := &Server{internal: &ssh.Server{}, HostKeys: []string{filepath.Join(t.TempDir(), "missing")}}
		err := s.setupHostKeys()
		require.ErrorContains(t, err, "failed to read provided host key")
	})

	t.Run("provided host keys are used", func(t *testing.T) {
		path, _ := gtxWriteHostKey(t)
		s := &Server{internal: &ssh.Server{}, HostKeys: []string{path}}
		require.NoError(t, s.setupHostKeys())
		assert.Len(t, s.internal.HostSigners, 1)
	})

	t.Run("an unparsable host key is only logged", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad")
		require.NoError(t, os.WriteFile(path, []byte("not a key"), 0o600))

		s := &Server{internal: &ssh.Server{}, HostKeys: []string{path}}
		require.NoError(t, s.setupHostKeys())
		assert.Empty(t, s.internal.HostSigners)
	})

	t.Run("the default key is created if no key is provided", func(t *testing.T) {
		path, _ := gtxWriteHostKey(t)
		s := &Server{internal: &ssh.Server{}, ServerKeyPath: path}
		require.NoError(t, s.setupHostKeys())
		assert.Len(t, s.internal.HostSigners, 1)
	})

	t.Run("the default key can't be created", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "blocker")
		require.NoError(t, os.WriteFile(blocker, nil, 0o600))

		s := &Server{internal: &ssh.Server{}, ServerKeyPath: filepath.Join(blocker, "key")}
		require.ErrorContains(t, s.setupHostKeys(), "failed to setup default key")
	})

	t.Run("listen and serve fails for missing host keys", func(t *testing.T) {
		s := &Server{
			RepoCtrl: &repo.Controller{}, Host: "127.0.0.1", Port: gtxFreePort(t),
			HostKeys: []string{filepath.Join(t.TempDir(), "missing")},
		}
		require.ErrorContains(t, s.ListenAndServe(), "failed to setup host keys")
	})
}

func TestGtxServer_EndToEnd(t *testing.T) {
	signer := gtxNewSigner(t)

	verifier := &gtxVerifier{principal: &types.PrincipalInfo{ID: 7, UID: "gtx-user", Email: "gtx@example.com"}}
	port, hostKey := gtxServe(t, &Server{
		DefaultUser:       "git",
		KeepAliveInterval: time.Hour,
		Verifier:          verifier,
		RepoCtrl:          &repo.Controller{},
	})

	client := gtxDialEventually(t, port, "git", hostKey, gossh.PublicKeys(signer))
	assert.Equal(t, 1, verifier.callCount())

	t.Run("git-lfs-transfer isn't supported", func(t *testing.T) {
		out, err := gtxRun(t, client, "git-lfs-transfer 'space/repo.git' download")
		require.NoError(t, err)
		assert.Contains(t, out, "git-lfs-transfer is not supported.")
	})

	t.Run("command needs an argument", func(t *testing.T) {
		out, err := gtxRun(t, client, "git-upload-pack")
		require.NoError(t, err)
		assert.Contains(t, out, `command "git-upload-pack" must have an argument`)
	})

	t.Run("empty command", func(t *testing.T) {
		out, err := gtxRun(t, client, "")
		require.NoError(t, err)
		assert.Contains(t, out, "must have an argument")
	})

	t.Run("unsupported command", func(t *testing.T) {
		out, err := gtxRun(t, client, "rm -rf /")
		require.NoError(t, err)
		assert.Contains(t, out, `command not supported: "rm -rf /"`)
	})

	t.Run("invalid lfs command", func(t *testing.T) {
		out, err := gtxRun(t, client, "git-lfs-gtx 'space/repo.git'")
		require.NoError(t, err)
		assert.Contains(t, out, "command not supported")
	})

	t.Run("pty requests are rejected", func(t *testing.T) {
		session, err := client.NewSession()
		require.NoError(t, err)
		defer func() { _ = session.Close() }()

		require.Error(t, session.RequestPty("xterm", 24, 80, gossh.TerminalModes{}))
	})

	t.Run("wrong user name is rejected", func(t *testing.T) {
		before := verifier.callCount()

		_, err := gtxDial(t, port, "someone-else", hostKey, gossh.PublicKeys(signer))
		require.Error(t, err)
		assert.Equal(t, before, verifier.callCount(), "the verifier isn't asked for an invalid user name")
	})

	t.Run("unknown keys are rejected", func(t *testing.T) {
		verifier.mu.Lock()
		verifier.principal, verifier.err = nil, gitnesserrors.NotFound("unknown key")
		verifier.mu.Unlock()

		_, err := gtxDial(t, port, "git", hostKey, gossh.PublicKeys(signer))
		require.Error(t, err)
	})

	t.Run("verification failures are rejected", func(t *testing.T) {
		verifier.mu.Lock()
		verifier.principal, verifier.err = nil, errors.New("gtx verification failure")
		verifier.mu.Unlock()

		_, err := gtxDial(t, port, "git", hostKey, gossh.PublicKeys(signer))
		require.Error(t, err)
	})
}

func TestGtxServer_NoDefaultUserRestriction(t *testing.T) {
	signer := gtxNewSigner(t)

	verifier := &gtxVerifier{principal: &types.PrincipalInfo{ID: 1, UID: "anyone"}}
	port, hostKey := gtxServe(t, &Server{Verifier: verifier, RepoCtrl: &repo.Controller{}})

	client := gtxDialEventually(t, port, "any-user-name", hostKey, gossh.PublicKeys(signer))

	out, err := gtxRun(t, client, "git-lfs-transfer 'a/b.git' upload")
	require.NoError(t, err)
	assert.Contains(t, out, "not supported")
	assert.Equal(t, 1, verifier.callCount())
}

func TestGtxServer_SessionWithoutPrincipal(t *testing.T) {
	hostKeyPath, hostKey := gtxWriteHostKey(t)
	port := gtxFreePort(t)

	// a server that doesn't authenticate through the public key handler has no principal in the session.
	srv := &ssh.Server{
		Addr:            net.JoinHostPort("127.0.0.1", strconv.Itoa(port)),
		Handler:         (&Server{RepoCtrl: &repo.Controller{}}).sessionHandler,
		PasswordHandler: func(ssh.Context, string) bool { return true },
	}
	require.NoError(t, srv.SetOption(ssh.HostKeyFile(hostKeyPath)))

	go func() { _ = srv.ListenAndServe() }()
	t.Cleanup(func() { _ = srv.Close() })

	client := gtxDialEventually(t, port, "user", hostKey, gossh.Password("secret"))

	out, err := gtxRun(t, client, "git-upload-pack 'a/b.git'")
	require.NoError(t, err)
	assert.Contains(t, out, "principal not found or empty")
}

func TestGtxServer_ListenAndServeFailsIfTheAddressIsTaken(t *testing.T) {
	hostKeyPath, _ := gtxWriteHostKey(t)

	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = l.Close() }()

	addr, ok := l.Addr().(*net.TCPAddr)
	require.True(t, ok)

	s := &Server{
		Host: "127.0.0.1", Port: addr.Port, HostKeys: []string{hostKeyPath},
		Verifier: &gtxVerifier{}, RepoCtrl: &repo.Controller{},
	}

	require.ErrorContains(t, s.ListenAndServe(), "ssh service not running")
	require.NotNil(t, s.internal)
	assert.Equal(t, net.JoinHostPort("127.0.0.1", strconv.Itoa(addr.Port)), s.internal.Addr)
}

func TestGtxServer_Shutdown(t *testing.T) {
	s := &Server{internal: &ssh.Server{Addr: "127.0.0.1:0"}}

	ctx, cancel := context.WithTimeout(context.Background(), gtxTimeout)
	defer cancel()

	require.NoError(t, s.Shutdown(ctx))
}

func TestGtxChainMiddleware(t *testing.T) {
	var calls []string
	record := func(name string) Middleware {
		return func(next ssh.Handler) ssh.Handler {
			return func(s ssh.Session) {
				calls = append(calls, name+" before")
				next(s)
				calls = append(calls, name+" after")
			}
		}
	}

	handler := ChainMiddleware(func(ssh.Session) { calls = append(calls, "handler") }, record("a"), record("b"))
	handler(nil)

	assert.Equal(t, []string{"a before", "b before", "handler", "b after", "a after"}, calls)

	calls = nil
	ChainMiddleware(func(ssh.Session) { calls = append(calls, "handler") })(nil)
	assert.Equal(t, []string{"handler"}, calls)
}

func TestGtxChainPublicKeyMiddleware(t *testing.T) {
	var calls []string
	record := func(name string) PublicKeyMiddleware {
		return func(next ssh.PublicKeyHandler) ssh.PublicKeyHandler {
			return func(ctx ssh.Context, key ssh.PublicKey) bool {
				calls = append(calls, name)
				return next(ctx, key)
			}
		}
	}

	handler := ChainPublicKeyMiddleware(
		func(ssh.Context, ssh.PublicKey) bool { calls = append(calls, "handler"); return true },
		record("a"), record("b"))

	assert.True(t, handler(nil, nil))
	assert.Equal(t, []string{"a", "b", "handler"}, calls)
}

func TestGtxMiddleware_EndToEnd(t *testing.T) {
	hostKeyPath, hostKey := gtxWriteHostKey(t)
	signer := gtxNewSigner(t)
	port := gtxFreePort(t)

	var (
		mu       sync.Mutex
		accepted []string
	)

	srv := &ssh.Server{
		Addr: net.JoinHostPort("127.0.0.1", strconv.Itoa(port)),
		Handler: ChainMiddleware(
			func(s ssh.Session) {
				switch s.RawCommand() {
				case "panic":
					panic("gtx handler panic")
				default:
					_, _ = fmt.Fprint(s, "ok:"+s.RawCommand())
				}
			},
			PanicRecoverMiddleware, HLogRequestIDHandler, HLogAccessLogHandler,
		),
		PublicKeyHandler: ChainPublicKeyMiddleware(
			func(ctx ssh.Context, _ ssh.PublicKey) bool {
				mu.Lock()
				defer mu.Unlock()
				accepted = append(accepted, ctx.User())

				return ctx.User() != "denied"
			},
			LogPublicKeyMiddleware,
		),
	}
	require.NoError(t, srv.SetOption(ssh.HostKeyFile(hostKeyPath)))

	go func() { _ = srv.ListenAndServe() }()
	t.Cleanup(func() { _ = srv.Close() })

	client := gtxDialEventually(t, port, "alice", hostKey, gossh.PublicKeys(signer))

	t.Run("requests are served through the middleware", func(t *testing.T) {
		out, err := gtxRun(t, client, "hello")
		require.NoError(t, err)
		assert.Equal(t, "ok:hello", out)
	})

	t.Run("panics are recovered", func(t *testing.T) {
		out, err := gtxRun(t, client, "panic")
		require.NoError(t, err)
		assert.Contains(t, out, "Internal server error. Please try again later.")

		// the connection is still usable.
		out, err = gtxRun(t, client, "again")
		require.NoError(t, err)
		assert.Equal(t, "ok:again", out)
	})

	t.Run("public key middleware passes the decision through", func(t *testing.T) {
		_, err := gtxDial(t, port, "denied", hostKey, gossh.PublicKeys(signer))
		require.Error(t, err)

		mu.Lock()
		defer mu.Unlock()
		assert.Contains(t, accepted, "alice")
		assert.Contains(t, accepted, "denied")
	})
}

func TestGtxGetLogger(t *testing.T) {
	assert.NotEmpty(t, getLoggerWithRequestID("0123456789012345678901234567890").GetLevel().String())
}
