// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssh

import (
	"bufio"
	"crypto/cipher"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"hash"
	"io"
	"net"
	"sync"
)

const (
	packetSizeMultiple = 16 // TODO(huin) this should be determined by the cipher.

	// RFC 4253 section 6.1 defines a minimum packet size of 32768 that implementations
	// MUST be able to process (plus a few more kilobytes for padding and mac). The RFC
	// indicates implementations SHOULD be able to handle larger packet sizes, but then
	// waffles on about reasonable limits.
	//
	// OpenSSH caps their maxPacket at 256kb so we choose to do the same.
	maxPacket = 256 * 1024
)

// packetConn represents a transport that implements packet based
// operations.
type packetConn interface {
	// Encrypt and send a packet of data to the remote peer.
	writePacket(packet []byte) error

	// Read a packet from the connection
	readPacket() ([]byte, error)

	// Close closes the write-side of the connection.
	Close() error
}

// transport represents the SSH connection to the remote peer.
type transport struct {
	reader
	writer

	net.Conn

	// Initial H used for the session ID. Once assigned this does
	// not change, even during subsequent key exchanges.
	sessionID []byte
}

// reader represents the incoming connection state.
type reader struct {
	io.Reader
	common
}

// writer represents the outgoing connection state.
type writer struct {
	sync.Mutex // protects writer.Writer from concurrent writes
	*bufio.Writer
	rand io.Reader
	common
}

// prepareKeyChange sets up key material for a keychange. The key changes in
// both directions are triggered by reading and writing a msgNewKey packet
// respectively.
func (t *transport) prepareKeyChange(algs *algorithms, kexResult *kexResult) error {
	t.writer.cipherAlgo = algs.wCipher
	t.writer.macAlgo = algs.wMAC
	t.writer.compressionAlgo = algs.wCompression

	t.reader.cipherAlgo = algs.rCipher
	t.reader.macAlgo = algs.rMAC
	t.reader.compressionAlgo = algs.rCompression

	if t.sessionID == nil {
		t.sessionID = kexResult.H
	}

	kexResult.SessionID = t.sessionID
	t.reader.pendingKeyChange <- kexResult
	t.writer.pendingKeyChange <- kexResult
	return nil
}

// common represents the cipher state needed to process messages in a single
// direction.
type common struct {
	seqNum uint32
	mac    hash.Hash
	cipher cipher.Stream

	cipherAlgo      string
	macAlgo         string
	compressionAlgo string

	dir              direction
	pendingKeyChange chan *kexResult
}

// Read and decrypt a single packet from the remote peer.
func (r *reader) readPacket() ([]byte, error) {
	var lengthBytes = make([]byte, 5)
	var macSize uint32
	if _, err := io.ReadFull(r, lengthBytes); err != nil {
		return nil, err
	}

	r.cipher.XORKeyStream(lengthBytes, lengthBytes)

	if r.mac != nil {
		r.mac.Reset()
		seqNumBytes := []byte{
			byte(r.seqNum >> 24),
			byte(r.seqNum >> 16),
			byte(r.seqNum >> 8),
			byte(r.seqNum),
		}
		r.mac.Write(seqNumBytes)
		r.mac.Write(lengthBytes)
		macSize = uint32(r.mac.Size())
	}

	length := binary.BigEndian.Uint32(lengthBytes[0:4])
	paddingLength := uint32(lengthBytes[4])

	if length <= paddingLength+1 {
		return nil, errors.New("ssh: invalid packet length, packet too small")
	}

	if length > maxPacket {
		return nil, errors.New("ssh: invalid packet length, packet too large")
	}

	packet := make([]byte, length-1+macSize)
	if _, err := io.ReadFull(r, packet); err != nil {
		return nil, err
	}
	mac := packet[length-1:]
	r.cipher.XORKeyStream(packet, packet[:length-1])

	if r.mac != nil {
		r.mac.Write(packet[:length-1])
		if subtle.ConstantTimeCompare(r.mac.Sum(nil), mac) != 1 {
			return nil, errors.New("ssh: MAC failure")
		}
	}

	r.seqNum++
	packet = packet[:length-paddingLength-1]

	if len(packet) > 0 && packet[0] == msgNewKeys {
		select {
		case k := <-r.pendingKeyChange:
			if err := r.setupKeys(r.dir, k); err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("ssh: got bogus newkeys message.")
		}
	}
	return packet, nil
}

// Read and decrypt next packet discarding debug and noop messages.
func (t *transport) readPacket() ([]byte, error) {
	for {
		packet, err := t.reader.readPacket()
		if err != nil {
			return nil, err
		}
		if len(packet) == 0 {
			return nil, errors.New("ssh: zero length packet")
		}

		if packet[0] != msgIgnore && packet[0] != msgDebug {
			return packet, nil
		}
	}
	panic("unreachable")
}

// Encrypt and send a packet of data to the remote peer.
func (w *writer) writePacket(packet []byte) error {
	changeKeys := len(packet) > 0 && packet[0] == msgNewKeys

	if len(packet) > maxPacket {
		return errors.New("ssh: packet too large")
	}
	w.Mutex.Lock()
	defer w.Mutex.Unlock()

	paddingLength := packetSizeMultiple - (5+len(packet))%packetSizeMultiple
	if paddingLength < 4 {
		paddingLength += packetSizeMultiple
	}

	length := len(packet) + 1 + paddingLength
	lengthBytes := []byte{
		byte(length >> 24),
		byte(length >> 16),
		byte(length >> 8),
		byte(length),
		byte(paddingLength),
	}
	padding := make([]byte, paddingLength)
	_, err := io.ReadFull(w.rand, padding)
	if err != nil {
		return err
	}

	if w.mac != nil {
		w.mac.Reset()
		seqNumBytes := []byte{
			byte(w.seqNum >> 24),
			byte(w.seqNum >> 16),
			byte(w.seqNum >> 8),
			byte(w.seqNum),
		}
		w.mac.Write(seqNumBytes)
		w.mac.Write(lengthBytes)
		w.mac.Write(packet)
		w.mac.Write(padding)
	}

	// TODO(dfc) lengthBytes, packet and padding should be
	// subslices of a single buffer
	w.cipher.XORKeyStream(lengthBytes, lengthBytes)
	w.cipher.XORKeyStream(packet, packet)
	w.cipher.XORKeyStream(padding, padding)

	if _, err := w.Write(lengthBytes); err != nil {
		return err
	}
	if _, err := w.Write(packet); err != nil {
		return err
	}
	if _, err := w.Write(padding); err != nil {
		return err
	}

	if w.mac != nil {
		if _, err := w.Write(w.mac.Sum(nil)); err != nil {
			return err
		}
	}

	w.seqNum++
	if err = w.Flush(); err != nil {
		return err
	}

	if changeKeys {
		select {
		case k := <-w.pendingKeyChange:
			err = w.setupKeys(w.dir, k)
		default:
			panic("ssh: no key material for msgNewKeys")
		}
	}
	return err
}

func newTransport(conn net.Conn, rand io.Reader, isClient bool) *transport {
	t := &transport{
		reader: reader{
			Reader: bufio.NewReader(conn),
			common: common{
				cipher:           noneCipher{},
				pendingKeyChange: make(chan *kexResult, 1),
			},
		},
		writer: writer{
			Writer: bufio.NewWriter(conn),
			rand:   rand,
			common: common{
				cipher:           noneCipher{},
				pendingKeyChange: make(chan *kexResult, 1),
			},
		},
		Conn: conn,
	}
	if isClient {
		t.reader.dir = serverKeys
		t.writer.dir = clientKeys
	} else {
		t.reader.dir = clientKeys
		t.writer.dir = serverKeys
	}

	return t
}

type direction struct {
	ivTag     []byte
	keyTag    []byte
	macKeyTag []byte
}

// TODO(dfc) can this be made a constant ?
var (
	serverKeys = direction{[]byte{'B'}, []byte{'D'}, []byte{'F'}}
	clientKeys = direction{[]byte{'A'}, []byte{'C'}, []byte{'E'}}
)

// setupKeys sets the cipher and MAC keys from kex.K, kex.H and sessionId, as
// described in RFC 4253, section 6.4. direction should either be serverKeys
// (to setup server->client keys) or clientKeys (for client->server keys).
func (c *common) setupKeys(d direction, r *kexResult) error {
	cipherMode := cipherModes[c.cipherAlgo]
	macMode := macModes[c.macAlgo]

	iv := make([]byte, cipherMode.ivSize)
	key := make([]byte, cipherMode.keySize)
	macKey := make([]byte, macMode.keySize)

	h := r.Hash.New()
	generateKeyMaterial(iv, d.ivTag, r.K, r.H, r.SessionID, h)
	generateKeyMaterial(key, d.keyTag, r.K, r.H, r.SessionID, h)
	generateKeyMaterial(macKey, d.macKeyTag, r.K, r.H, r.SessionID, h)

	c.mac = macMode.new(macKey)

	var err error
	c.cipher, err = cipherMode.createCipher(key, iv)
	return err
}

// generateKeyMaterial fills out with key material generated from tag, K, H
// and sessionId, as specified in RFC 4253, section 7.2.
func generateKeyMaterial(out, tag []byte, K, H, sessionId []byte, h hash.Hash) {
	var digestsSoFar []byte

	for len(out) > 0 {
		h.Reset()
		h.Write(K)
		h.Write(H)

		if len(digestsSoFar) == 0 {
			h.Write(tag)
			h.Write(sessionId)
		} else {
			h.Write(digestsSoFar)
		}

		digest := h.Sum(nil)
		n := copy(out, digest)
		out = out[n:]
		if len(out) > 0 {
			digestsSoFar = append(digestsSoFar, digest...)
		}
	}
}

const packageVersion = "SSH-2.0-Go"

// Sends and receives a version line.  The versionLine string should
// be US ASCII, start with "SSH-2.0-", and should not include a
// newline. exchangeVersions returns the other side's version line.
func exchangeVersions(rw io.ReadWriter, versionLine []byte) (them []byte, err error) {
	// Contrary to the RFC, we do not ignore lines that don't
	// start with "SSH-2.0-" to make the library usable with
	// nonconforming servers.
	for _, c := range versionLine {
		// The spec disallows non US-ASCII chars, and
		// specifically forbids null chars.
		if c < 32 {
			return nil, errors.New("ssh: junk character in version line")
		}
	}
	if _, err = rw.Write(append(versionLine, '\r', '\n')); err != nil {
		return
	}

	them, err = readVersion(rw)
	return them, err
}

// maxVersionStringBytes is the maximum number of bytes that we'll
// accept as a version string. RFC 4253 section 4.2 limits this at 255
// chars
const maxVersionStringBytes = 255

// Read version string as specified by RFC 4253, section 4.2.
func readVersion(r io.Reader) ([]byte, error) {
	versionString := make([]byte, 0, 64)
	var ok bool
	var buf [1]byte

	for len(versionString) < maxVersionStringBytes {
		_, err := io.ReadFull(r, buf[:])
		if err != nil {
			return nil, err
		}
		// The RFC says that the version should be terminated with \r\n
		// but several SSH servers actually only send a \n.
		if buf[0] == '\n' {
			ok = true
			break
		}

		// non ASCII chars are disallowed, but we are lenient,
		// since Go doesn't use null-terminated strings.

		// The RFC allows a comment after a space, however,
		// all of it (version and comments) goes into the
		// session hash.
		versionString = append(versionString, buf[0])
	}

	if !ok {
		return nil, errors.New("ssh: overflow reading version string")
	}

	// There might be a '\r' on the end which we should remove.
	if len(versionString) > 0 && versionString[len(versionString)-1] == '\r' {
		versionString = versionString[:len(versionString)-1]
	}
	return versionString, nil
}
