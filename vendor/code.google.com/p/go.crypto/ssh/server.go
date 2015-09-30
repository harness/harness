// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssh

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	_ "crypto/sha1"
)

type ServerConfig struct {
	hostKeys []Signer

	// Rand provides the source of entropy for key exchange. If Rand is
	// nil, the cryptographic random reader in package crypto/rand will
	// be used.
	Rand io.Reader

	// NoClientAuth is true if clients are allowed to connect without
	// authenticating.
	NoClientAuth bool

	// PasswordCallback, if non-nil, is called when a user attempts to
	// authenticate using a password. It may be called concurrently from
	// several goroutines.
	PasswordCallback func(conn *ServerConn, user, password string) bool

	// PublicKeyCallback, if non-nil, is called when a client attempts public
	// key authentication. It must return true if the given public key is
	// valid for the given user.
	PublicKeyCallback func(conn *ServerConn, user, algo string, pubkey []byte) bool

	// KeyboardInteractiveCallback, if non-nil, is called when
	// keyboard-interactive authentication is selected (RFC
	// 4256). The client object's Challenge function should be
	// used to query the user. The callback may offer multiple
	// Challenge rounds. To avoid information leaks, the client
	// should be presented a challenge even if the user is
	// unknown.
	KeyboardInteractiveCallback func(conn *ServerConn, user string, client ClientKeyboardInteractive) bool

	// Cryptographic-related configuration.
	Crypto CryptoConfig
}

func (c *ServerConfig) rand() io.Reader {
	if c.Rand == nil {
		return rand.Reader
	}
	return c.Rand
}

// AddHostKey adds a private key as a host key. If an existing host
// key exists with the same algorithm, it is overwritten.
func (s *ServerConfig) AddHostKey(key Signer) {
	for i, k := range s.hostKeys {
		if k.PublicKey().PublicKeyAlgo() == key.PublicKey().PublicKeyAlgo() {
			s.hostKeys[i] = key
			return
		}
	}

	s.hostKeys = append(s.hostKeys, key)
}

// SetRSAPrivateKey sets the private key for a Server. A Server must have a
// private key configured in order to accept connections. The private key must
// be in the form of a PEM encoded, PKCS#1, RSA private key. The file "id_rsa"
// typically contains such a key.
func (s *ServerConfig) SetRSAPrivateKey(pemBytes []byte) error {
	priv, err := ParsePrivateKey(pemBytes)
	if err != nil {
		return err
	}
	s.AddHostKey(priv)
	return nil
}

// cachedPubKey contains the results of querying whether a public key is
// acceptable for a user. The cache only applies to a single ServerConn.
type cachedPubKey struct {
	user, algo string
	pubKey     []byte
	result     bool
}

const maxCachedPubKeys = 16

// A ServerConn represents an incoming connection.
type ServerConn struct {
	transport *transport
	config    *ServerConfig

	channels   map[uint32]*serverChan
	nextChanId uint32

	// lock protects err and channels.
	lock sync.Mutex
	err  error

	// cachedPubKeys contains the cache results of tests for public keys.
	// Since SSH clients will query whether a public key is acceptable
	// before attempting to authenticate with it, we end up with duplicate
	// queries for public key validity.
	cachedPubKeys []cachedPubKey

	// User holds the successfully authenticated user name.
	// It is empty if no authentication is used.  It is populated before
	// any authentication callback is called and not assigned to after that.
	User string

	// ClientVersion is the client's version, populated after
	// Handshake is called. It should not be modified.
	ClientVersion []byte

	// Our version.
	serverVersion []byte
}

// Server returns a new SSH server connection
// using c as the underlying transport.
func Server(c net.Conn, config *ServerConfig) *ServerConn {
	return &ServerConn{
		transport: newTransport(c, config.rand(), false /* not client */),
		channels:  make(map[uint32]*serverChan),
		config:    config,
	}
}

// signAndMarshal signs the data with the appropriate algorithm,
// and serializes the result in SSH wire format.
func signAndMarshal(k Signer, rand io.Reader, data []byte) ([]byte, error) {
	sig, err := k.Sign(rand, data)
	if err != nil {
		return nil, err
	}

	return serializeSignature(k.PublicKey().PrivateKeyAlgo(), sig), nil
}

// Close closes the connection.
func (s *ServerConn) Close() error { return s.transport.Close() }

// LocalAddr returns the local network address.
func (c *ServerConn) LocalAddr() net.Addr { return c.transport.LocalAddr() }

// RemoteAddr returns the remote network address.
func (c *ServerConn) RemoteAddr() net.Addr { return c.transport.RemoteAddr() }

// Handshake performs an SSH transport and client authentication on the given ServerConn.
func (s *ServerConn) Handshake() error {
	var err error
	s.serverVersion = []byte(packageVersion)
	s.ClientVersion, err = exchangeVersions(s.transport.Conn, s.serverVersion)
	if err != nil {
		return err
	}
	if err := s.clientInitHandshake(nil, nil); err != nil {
		return err
	}

	var packet []byte
	if packet, err = s.transport.readPacket(); err != nil {
		return err
	}
	var serviceRequest serviceRequestMsg
	if err := unmarshal(&serviceRequest, packet, msgServiceRequest); err != nil {
		return err
	}
	if serviceRequest.Service != serviceUserAuth {
		return errors.New("ssh: requested service '" + serviceRequest.Service + "' before authenticating")
	}
	serviceAccept := serviceAcceptMsg{
		Service: serviceUserAuth,
	}
	if err := s.transport.writePacket(marshal(msgServiceAccept, serviceAccept)); err != nil {
		return err
	}

	if err := s.authenticate(); err != nil {
		return err
	}
	return err
}

func (s *ServerConn) clientInitHandshake(clientKexInit *kexInitMsg, clientKexInitPacket []byte) (err error) {
	serverKexInit := kexInitMsg{
		KexAlgos:                s.config.Crypto.kexes(),
		CiphersClientServer:     s.config.Crypto.ciphers(),
		CiphersServerClient:     s.config.Crypto.ciphers(),
		MACsClientServer:        s.config.Crypto.macs(),
		MACsServerClient:        s.config.Crypto.macs(),
		CompressionClientServer: supportedCompressions,
		CompressionServerClient: supportedCompressions,
	}
	for _, k := range s.config.hostKeys {
		serverKexInit.ServerHostKeyAlgos = append(
			serverKexInit.ServerHostKeyAlgos, k.PublicKey().PublicKeyAlgo())
	}

	serverKexInitPacket := marshal(msgKexInit, serverKexInit)
	if err = s.transport.writePacket(serverKexInitPacket); err != nil {
		return
	}

	if clientKexInitPacket == nil {
		clientKexInit = new(kexInitMsg)
		if clientKexInitPacket, err = s.transport.readPacket(); err != nil {
			return
		}
		if err = unmarshal(clientKexInit, clientKexInitPacket, msgKexInit); err != nil {
			return
		}
	}

	algs := findAgreedAlgorithms(clientKexInit, &serverKexInit)
	if algs == nil {
		return errors.New("ssh: no common algorithms")
	}

	if clientKexInit.FirstKexFollows && algs.kex != clientKexInit.KexAlgos[0] {
		// The client sent a Kex message for the wrong algorithm,
		// which we have to ignore.
		if _, err = s.transport.readPacket(); err != nil {
			return
		}
	}

	var hostKey Signer
	for _, k := range s.config.hostKeys {
		if algs.hostKey == k.PublicKey().PublicKeyAlgo() {
			hostKey = k
		}
	}

	kex, ok := kexAlgoMap[algs.kex]
	if !ok {
		return fmt.Errorf("ssh: unexpected key exchange algorithm %v", algs.kex)
	}

	magics := handshakeMagics{
		serverVersion: s.serverVersion,
		clientVersion: s.ClientVersion,
		serverKexInit: marshal(msgKexInit, serverKexInit),
		clientKexInit: clientKexInitPacket,
	}
	result, err := kex.Server(s.transport, s.config.rand(), &magics, hostKey)
	if err != nil {
		return err
	}

	if err = s.transport.prepareKeyChange(algs, result); err != nil {
		return err
	}

	if err = s.transport.writePacket([]byte{msgNewKeys}); err != nil {
		return
	}
	if packet, err := s.transport.readPacket(); err != nil {
		return err
	} else if packet[0] != msgNewKeys {
		return UnexpectedMessageError{msgNewKeys, packet[0]}
	}

	return
}

func isAcceptableAlgo(algo string) bool {
	switch algo {
	case KeyAlgoRSA, KeyAlgoDSA, KeyAlgoECDSA256, KeyAlgoECDSA384, KeyAlgoECDSA521,
		CertAlgoRSAv01, CertAlgoDSAv01, CertAlgoECDSA256v01, CertAlgoECDSA384v01, CertAlgoECDSA521v01:
		return true
	}
	return false
}

// testPubKey returns true if the given public key is acceptable for the user.
func (s *ServerConn) testPubKey(user, algo string, pubKey []byte) bool {
	if s.config.PublicKeyCallback == nil || !isAcceptableAlgo(algo) {
		return false
	}

	for _, c := range s.cachedPubKeys {
		if c.user == user && c.algo == algo && bytes.Equal(c.pubKey, pubKey) {
			return c.result
		}
	}

	result := s.config.PublicKeyCallback(s, user, algo, pubKey)
	if len(s.cachedPubKeys) < maxCachedPubKeys {
		c := cachedPubKey{
			user:   user,
			algo:   algo,
			pubKey: make([]byte, len(pubKey)),
			result: result,
		}
		copy(c.pubKey, pubKey)
		s.cachedPubKeys = append(s.cachedPubKeys, c)
	}

	return result
}

func (s *ServerConn) authenticate() error {
	var userAuthReq userAuthRequestMsg
	var err error
	var packet []byte

userAuthLoop:
	for {
		if packet, err = s.transport.readPacket(); err != nil {
			return err
		}
		if err = unmarshal(&userAuthReq, packet, msgUserAuthRequest); err != nil {
			return err
		}

		if userAuthReq.Service != serviceSSH {
			return errors.New("ssh: client attempted to negotiate for unknown service: " + userAuthReq.Service)
		}

		switch userAuthReq.Method {
		case "none":
			if s.config.NoClientAuth {
				break userAuthLoop
			}
		case "password":
			if s.config.PasswordCallback == nil {
				break
			}
			payload := userAuthReq.Payload
			if len(payload) < 1 || payload[0] != 0 {
				return ParseError{msgUserAuthRequest}
			}
			payload = payload[1:]
			password, payload, ok := parseString(payload)
			if !ok || len(payload) > 0 {
				return ParseError{msgUserAuthRequest}
			}

			s.User = userAuthReq.User
			if s.config.PasswordCallback(s, userAuthReq.User, string(password)) {
				break userAuthLoop
			}
		case "keyboard-interactive":
			if s.config.KeyboardInteractiveCallback == nil {
				break
			}

			s.User = userAuthReq.User
			if s.config.KeyboardInteractiveCallback(s, s.User, &sshClientKeyboardInteractive{s}) {
				break userAuthLoop
			}
		case "publickey":
			if s.config.PublicKeyCallback == nil {
				break
			}
			payload := userAuthReq.Payload
			if len(payload) < 1 {
				return ParseError{msgUserAuthRequest}
			}
			isQuery := payload[0] == 0
			payload = payload[1:]
			algoBytes, payload, ok := parseString(payload)
			if !ok {
				return ParseError{msgUserAuthRequest}
			}
			algo := string(algoBytes)

			pubKey, payload, ok := parseString(payload)
			if !ok {
				return ParseError{msgUserAuthRequest}
			}
			if isQuery {
				// The client can query if the given public key
				// would be okay.
				if len(payload) > 0 {
					return ParseError{msgUserAuthRequest}
				}
				if s.testPubKey(userAuthReq.User, algo, pubKey) {
					okMsg := userAuthPubKeyOkMsg{
						Algo:   algo,
						PubKey: string(pubKey),
					}
					if err = s.transport.writePacket(marshal(msgUserAuthPubKeyOk, okMsg)); err != nil {
						return err
					}
					continue userAuthLoop
				}
			} else {
				sig, payload, ok := parseSignature(payload)
				if !ok || len(payload) > 0 {
					return ParseError{msgUserAuthRequest}
				}
				// Ensure the public key algo and signature algo
				// are supported.  Compare the private key
				// algorithm name that corresponds to algo with
				// sig.Format.  This is usually the same, but
				// for certs, the names differ.
				if !isAcceptableAlgo(algo) || !isAcceptableAlgo(sig.Format) || pubAlgoToPrivAlgo(algo) != sig.Format {
					break
				}
				signedData := buildDataSignedForAuth(s.transport.sessionID, userAuthReq, algoBytes, pubKey)
				key, _, ok := ParsePublicKey(pubKey)
				if !ok {
					return ParseError{msgUserAuthRequest}
				}

				if !key.Verify(signedData, sig.Blob) {
					return ParseError{msgUserAuthRequest}
				}
				// TODO(jmpittman): Implement full validation for certificates.
				s.User = userAuthReq.User
				if s.testPubKey(userAuthReq.User, algo, pubKey) {
					break userAuthLoop
				}
			}
		}

		var failureMsg userAuthFailureMsg
		if s.config.PasswordCallback != nil {
			failureMsg.Methods = append(failureMsg.Methods, "password")
		}
		if s.config.PublicKeyCallback != nil {
			failureMsg.Methods = append(failureMsg.Methods, "publickey")
		}
		if s.config.KeyboardInteractiveCallback != nil {
			failureMsg.Methods = append(failureMsg.Methods, "keyboard-interactive")
		}

		if len(failureMsg.Methods) == 0 {
			return errors.New("ssh: no authentication methods configured but NoClientAuth is also false")
		}

		if err = s.transport.writePacket(marshal(msgUserAuthFailure, failureMsg)); err != nil {
			return err
		}
	}

	packet = []byte{msgUserAuthSuccess}
	if err = s.transport.writePacket(packet); err != nil {
		return err
	}

	return nil
}

// sshClientKeyboardInteractive implements a ClientKeyboardInteractive by
// asking the client on the other side of a ServerConn.
type sshClientKeyboardInteractive struct {
	*ServerConn
}

func (c *sshClientKeyboardInteractive) Challenge(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
	if len(questions) != len(echos) {
		return nil, errors.New("ssh: echos and questions must have equal length")
	}

	var prompts []byte
	for i := range questions {
		prompts = appendString(prompts, questions[i])
		prompts = appendBool(prompts, echos[i])
	}

	if err := c.transport.writePacket(marshal(msgUserAuthInfoRequest, userAuthInfoRequestMsg{
		Instruction: instruction,
		NumPrompts:  uint32(len(questions)),
		Prompts:     prompts,
	})); err != nil {
		return nil, err
	}

	packet, err := c.transport.readPacket()
	if err != nil {
		return nil, err
	}
	if packet[0] != msgUserAuthInfoResponse {
		return nil, UnexpectedMessageError{msgUserAuthInfoResponse, packet[0]}
	}
	packet = packet[1:]

	n, packet, ok := parseUint32(packet)
	if !ok || int(n) != len(questions) {
		return nil, &ParseError{msgUserAuthInfoResponse}
	}

	for i := uint32(0); i < n; i++ {
		ans, rest, ok := parseString(packet)
		if !ok {
			return nil, &ParseError{msgUserAuthInfoResponse}
		}

		answers = append(answers, string(ans))
		packet = rest
	}
	if len(packet) != 0 {
		return nil, errors.New("ssh: junk at end of message")
	}

	return answers, nil
}

const defaultWindowSize = 32768

// Accept reads and processes messages on a ServerConn. It must be called
// in order to demultiplex messages to any resulting Channels.
func (s *ServerConn) Accept() (Channel, error) {
	// TODO(dfc) s.lock is not held here so visibility of s.err is not guaranteed.
	if s.err != nil {
		return nil, s.err
	}

	for {
		packet, err := s.transport.readPacket()
		if err != nil {

			s.lock.Lock()
			s.err = err
			s.lock.Unlock()

			// TODO(dfc) s.lock protects s.channels but isn't being held here.
			for _, c := range s.channels {
				c.setDead()
				c.handleData(nil)
			}

			return nil, err
		}

		switch packet[0] {
		case msgChannelData:
			if len(packet) < 9 {
				// malformed data packet
				return nil, ParseError{msgChannelData}
			}
			remoteId := binary.BigEndian.Uint32(packet[1:5])
			s.lock.Lock()
			c, ok := s.channels[remoteId]
			if !ok {
				s.lock.Unlock()
				continue
			}
			if length := binary.BigEndian.Uint32(packet[5:9]); length > 0 {
				packet = packet[9:]
				c.handleData(packet[:length])
			}
			s.lock.Unlock()
		default:
			decoded, err := decode(packet)
			if err != nil {
				return nil, err
			}
			switch msg := decoded.(type) {
			case *channelOpenMsg:
				if msg.MaxPacketSize < minPacketLength || msg.MaxPacketSize > 1<<31 {
					return nil, errors.New("ssh: invalid MaxPacketSize from peer")
				}
				c := &serverChan{
					channel: channel{
						packetConn: s.transport,
						remoteId:   msg.PeersId,
						remoteWin:  window{Cond: newCond()},
						maxPacket:  msg.MaxPacketSize,
					},
					chanType:    msg.ChanType,
					extraData:   msg.TypeSpecificData,
					myWindow:    defaultWindowSize,
					serverConn:  s,
					cond:        newCond(),
					pendingData: make([]byte, defaultWindowSize),
				}
				c.remoteWin.add(msg.PeersWindow)
				s.lock.Lock()
				c.localId = s.nextChanId
				s.nextChanId++
				s.channels[c.localId] = c
				s.lock.Unlock()
				return c, nil

			case *channelRequestMsg:
				s.lock.Lock()
				c, ok := s.channels[msg.PeersId]
				if !ok {
					s.lock.Unlock()
					continue
				}
				c.handlePacket(msg)
				s.lock.Unlock()

			case *windowAdjustMsg:
				s.lock.Lock()
				c, ok := s.channels[msg.PeersId]
				if !ok {
					s.lock.Unlock()
					continue
				}
				c.handlePacket(msg)
				s.lock.Unlock()

			case *channelEOFMsg:
				s.lock.Lock()
				c, ok := s.channels[msg.PeersId]
				if !ok {
					s.lock.Unlock()
					continue
				}
				c.handlePacket(msg)
				s.lock.Unlock()

			case *channelCloseMsg:
				s.lock.Lock()
				c, ok := s.channels[msg.PeersId]
				if !ok {
					s.lock.Unlock()
					continue
				}
				c.handlePacket(msg)
				s.lock.Unlock()

			case *globalRequestMsg:
				if msg.WantReply {
					if err := s.transport.writePacket([]byte{msgRequestFailure}); err != nil {
						return nil, err
					}
				}

			case *kexInitMsg:
				s.lock.Lock()
				if err := s.clientInitHandshake(msg, packet); err != nil {
					s.lock.Unlock()
					return nil, err
				}
				s.lock.Unlock()
			case *disconnectMsg:
				return nil, io.EOF
			default:
				// Unknown message. Ignore.
			}
		}
	}

	panic("unreachable")
}

// A Listener implements a network listener (net.Listener) for SSH connections.
type Listener struct {
	listener net.Listener
	config   *ServerConfig
}

// Addr returns the listener's network address.
func (l *Listener) Addr() net.Addr {
	return l.listener.Addr()
}

// Close closes the listener.
func (l *Listener) Close() error {
	return l.listener.Close()
}

// Accept waits for and returns the next incoming SSH connection.
// The receiver should call Handshake() in another goroutine
// to avoid blocking the accepter.
func (l *Listener) Accept() (*ServerConn, error) {
	c, err := l.listener.Accept()
	if err != nil {
		return nil, err
	}
	return Server(c, l.config), nil
}

// Listen creates an SSH listener accepting connections on
// the given network address using net.Listen.
func Listen(network, addr string, config *ServerConfig) (*Listener, error) {
	l, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}
	return &Listener{
		l,
		config,
	}, nil
}
