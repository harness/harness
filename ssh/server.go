// Copyright 2023 Harness, Inc.
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
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/publickey"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog/log"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/exp/slices"
)

type contextKey string

const principalKey = contextKey("principalKey")

var (
	allowedCommands = []string{
		"git-upload-pack",
		"git-receive-pack",
	}
	defaultCiphers = []string{
		"chacha20-poly1305@openssh.com",
		"aes128-ctr",
		"aes192-ctr",
		"aes256-ctr",
		"aes128-gcm@openssh.com",
		"aes256-gcm@openssh.com",
	}
	defaultKeyExchanges = []string{
		"curve25519-sha256",
		"curve25519-sha256@libssh.org",
		"ecdh-sha2-nistp256",
		"ecdh-sha2-nistp384",
		"ecdh-sha2-nistp521",
		"diffie-hellman-group14-sha256",
		"diffie-hellman-group14-sha1",
	}
	defaultMACs = []string{
		"hmac-sha2-256-etm@openssh.com",
		"hmac-sha2-512-etm@openssh.com",
		"hmac-sha2-256",
		"hmac-sha2-512",
	}
	defaultServerKeyPath = "ssh/gitness.rsa"
	KeepAliveMsg         = "keepalive@openssh.com"
)

type Server struct {
	internal *ssh.Server

	Host        string
	Port        int
	DefaultUser string

	TrustedUserCAKeys       []string
	TrustedUserCAKeysParsed []gossh.PublicKey
	Ciphers                 []string
	KeyExchanges            []string
	MACs                    []string
	HostKeys                []string
	KeepAliveInterval       time.Duration

	Verifier publickey.Service
	RepoCtrl *repo.Controller
}

func (s *Server) sanitize() error {
	if s.Port == 0 {
		s.Port = 22
	}

	if len(s.Ciphers) == 0 {
		s.Ciphers = defaultCiphers
	}

	if len(s.KeyExchanges) == 0 {
		s.KeyExchanges = defaultKeyExchanges
	}

	if len(s.MACs) == 0 {
		s.MACs = defaultMACs
	}

	if s.KeepAliveInterval == 0 {
		s.KeepAliveInterval = 5000
	}

	if s.RepoCtrl == nil {
		return errors.InvalidArgument("repository controller is needed to run git service pack commands")
	}
	return nil
}

func (s *Server) ListenAndServe() error {
	err := s.sanitize()
	if err != nil {
		return fmt.Errorf("failed to sanitize server defaults: %w", err)
	}
	s.internal = &ssh.Server{
		Addr:             net.JoinHostPort(s.Host, strconv.Itoa(s.Port)),
		Handler:          s.sessionHandler,
		PublicKeyHandler: s.publicKeyHandler,
		PtyCallback: func(ssh.Context, ssh.Pty) bool {
			return false
		},
		ConnectionFailedCallback: sshConnectionFailed,
		ServerConfigCallback: func(ssh.Context) *gossh.ServerConfig {
			config := &gossh.ServerConfig{}
			config.KeyExchanges = s.KeyExchanges
			config.MACs = s.MACs
			config.Ciphers = s.Ciphers
			return config
		},
	}

	err = s.setupHostKeys()
	if err != nil {
		return fmt.Errorf("failed to setup host keys: %w", err)
	}

	log.Debug().Msgf("starting ssh service....: %v", s.internal.Addr)
	err = s.internal.ListenAndServe()
	if err != nil {
		return fmt.Errorf("ssh service not running: %w", err)
	}
	return nil
}

func (s *Server) setupHostKeys() error {
	keys := make([]string, 0, len(s.HostKeys))
	for _, key := range s.HostKeys {
		_, err := os.Stat(key)
		if err != nil {
			return fmt.Errorf("failed to read provided host key %q: %w", key, err)
		}
		keys = append(keys, key)
	}

	if len(keys) == 0 {
		log.Debug().Msg("no host key provided - setup default key if it doesn't exist yet")
		err := createKeyIfNotExists(defaultServerKeyPath)
		if err != nil {
			return fmt.Errorf("failed to setup default key %q: %w", defaultServerKeyPath, err)
		}
		keys = append(keys, defaultServerKeyPath)
	}

	// set keys to internal ssh server
	for _, key := range keys {
		err := s.internal.SetOption(ssh.HostKeyFile(key))
		if err != nil {
			log.Err(err).Msg("failed to set host key to ssh server")
		}
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Debug().Msgf("stopping ssh service: %v", s.internal.Addr)
	err := s.internal.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("failed to stop ssh service: %w", err)
	}
	return nil
}

func (s *Server) sessionHandler(session ssh.Session) {
	command := session.RawCommand()

	principal, ok := session.Context().Value(principalKey).(*types.PrincipalInfo)
	if !ok {
		_, _ = fmt.Fprintf(session.Stderr(), "principal not found or empty")
		return
	}

	parts := strings.Fields(command)
	if len(parts) < 2 {
		_, _ = fmt.Fprintf(session.Stderr(), "command %q must have an argument\n", command)
		return
	}

	// first part is git service pack command: git-upload-pack, git-receive-pack
	gitCommand := parts[0]
	if !slices.Contains(allowedCommands, gitCommand) {
		_, _ = fmt.Fprintf(session.Stderr(), "command not supported: %q\n", command)
		return
	}

	gitServicePack := strings.TrimPrefix(gitCommand, "git-")
	service, err := enum.ParseGitServiceType(gitServicePack)
	if err != nil {
		_, _ = fmt.Fprintf(session.Stderr(), "failed to parse service pack: %q\n", gitServicePack)
		return
	}

	// git command args
	gitArgs := parts[1:]

	// first git service pack cmd arg is path: 'space/repository.git' so we need to remove
	// single quotes.
	repoRef := strings.Trim(gitArgs[0], "'")
	// remove .git suffix
	repoRef = strings.TrimSuffix(repoRef, ".git")

	gitProtocol := ""
	for _, key := range session.Environ() {
		if strings.HasPrefix(key, "GIT_PROTOCOL=") {
			gitProtocol = key[len("GIT_PROTOCOL="):]
		}
	}

	ctx, cancel := context.WithCancel(session.Context())
	defer cancel()

	// set keep alive connection
	if s.KeepAliveInterval > 0 {
		go sendKeepAliveMsg(ctx, session, s.KeepAliveInterval)
	}

	err = s.RepoCtrl.GitServicePack(
		ctx,
		&auth.Session{
			Principal: types.Principal{
				ID:          principal.ID,
				UID:         principal.UID,
				Email:       principal.Email,
				Type:        principal.Type,
				DisplayName: principal.DisplayName,
				Created:     principal.Created,
				Updated:     principal.Updated,
			},
		},
		repoRef,
		api.ServicePackOptions{
			Service:  service,
			Stdout:   session,
			Stdin:    session,
			Stderr:   session.Stderr(),
			Protocol: gitProtocol,
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("git service pack failed")
		_, err = io.Copy(session.Stderr(), strings.NewReader(err.Error()))
		if err != nil {
			log.Error().Err(err).Msg("error writing to session stderr")
		}
	}
}

func sendKeepAliveMsg(ctx context.Context, session ssh.Session, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	log.Ctx(ctx).Debug().Str("remote_addr", session.RemoteAddr().String()).Msgf("sendKeepAliveMsg")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.Ctx(ctx).Debug().Msg("send keepalive message to ssh client")
			_, err := session.SendRequest(KeepAliveMsg, true, nil)
			if err != nil {
				log.Ctx(ctx).Debug().Err(err).Msg("failed to send keepalive message to ssh client")
			}
		}
	}
}

func (s *Server) publicKeyHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	if slices.Contains(publickey.DisallowedTypes, key.Type()) {
		log.Warn().Msgf("public key type not supported: %s", key.Type())
		return false
	}

	if s.DefaultUser != "" && ctx.User() != s.DefaultUser {
		log.Warn().Msgf("invalid SSH username %s - must use %s for all git operations via ssh",
			ctx.User(), s.DefaultUser)
		log.Warn().Msgf("failed authentication attempt from %s", ctx.RemoteAddr())
		return false
	}

	principal, err := s.Verifier.ValidateKey(ctx, key, enum.PublicKeyUsageAuth)
	if errors.IsNotFound(err) {
		log.Debug().Err(err).Msg("public key is unknown")
		return false
	}
	if err != nil {
		log.Warn().Err(err).Msg("failed to validate public key")
		return false
	}

	// check if we have a certificate
	if cert, ok := key.(*gossh.Certificate); ok {
		if len(s.TrustedUserCAKeys) == 0 {
			log.Warn().Msg("Certificate Rejected: No trusted certificate authorities for this server")
			log.Warn().Msgf("Failed authentication attempt from %s", ctx.RemoteAddr())
			return false
		}

		if cert.CertType != gossh.UserCert {
			log.Warn().Msg("Certificate Rejected: Not a user certificate")
			log.Warn().Msgf("Failed authentication attempt from %s", ctx.RemoteAddr())
			return false
		}

		certChecker := &gossh.CertChecker{}
		if err := certChecker.CheckCert(principal.UID, cert); err != nil {
			return false
		}
	}

	ctx.SetValue(principalKey, principal)
	return true
}

func sshConnectionFailed(conn net.Conn, err error) {
	log.Err(err).Msgf("failed connection from %s with error: %v", conn.RemoteAddr(), err)
}

func createKeyIfNotExists(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		// if the path already exists there's nothing we have to do
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check for for existence of key: %w", err)
	}

	log.Debug().Msgf("generate new key at %q", path)

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create dir %q for key: %w", dir, err)
	}

	err = GenerateKeyPair(path)
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	return nil
}

// GenerateKeyPair make a pair of public and private keys for SSH access.
func GenerateKeyPair(keyPath string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	f, err := os.OpenFile(keyPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := pem.Encode(f, privateKeyPEM); err != nil {
		return err
	}

	// generate public key
	pub, err := gossh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}

	public := gossh.MarshalAuthorizedKey(pub)
	p, err := os.OpenFile(keyPath+".pub", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer p.Close()

	_, err = p.Write(public)
	if err != nil {
		return fmt.Errorf("failed to write to public key: %w", err)
	}
	return nil
}
