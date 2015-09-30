// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssh

import (
	"bytes"
	"crypto"
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
)

// These constants represent the algorithm names for key types supported by this
// package.
const (
	KeyAlgoRSA      = "ssh-rsa"
	KeyAlgoDSA      = "ssh-dss"
	KeyAlgoECDSA256 = "ecdsa-sha2-nistp256"
	KeyAlgoECDSA384 = "ecdsa-sha2-nistp384"
	KeyAlgoECDSA521 = "ecdsa-sha2-nistp521"
)

// parsePubKey parses a public key of the given algorithm.
// Use ParsePublicKey for keys with prepended algorithm.
func parsePubKey(in []byte, algo string) (pubKey PublicKey, rest []byte, ok bool) {
	switch algo {
	case KeyAlgoRSA:
		return parseRSA(in)
	case KeyAlgoDSA:
		return parseDSA(in)
	case KeyAlgoECDSA256, KeyAlgoECDSA384, KeyAlgoECDSA521:
		return parseECDSA(in)
	case CertAlgoRSAv01, CertAlgoDSAv01, CertAlgoECDSA256v01, CertAlgoECDSA384v01, CertAlgoECDSA521v01:
		return parseOpenSSHCertV01(in, algo)
	}
	return nil, nil, false
}

// parseAuthorizedKey parses a public key in OpenSSH authorized_keys format
// (see sshd(8) manual page) once the options and key type fields have been
// removed.
func parseAuthorizedKey(in []byte) (out PublicKey, comment string, ok bool) {
	in = bytes.TrimSpace(in)

	i := bytes.IndexAny(in, " \t")
	if i == -1 {
		i = len(in)
	}
	base64Key := in[:i]

	key := make([]byte, base64.StdEncoding.DecodedLen(len(base64Key)))
	n, err := base64.StdEncoding.Decode(key, base64Key)
	if err != nil {
		return
	}
	key = key[:n]
	out, _, ok = ParsePublicKey(key)
	if !ok {
		return nil, "", false
	}
	comment = string(bytes.TrimSpace(in[i:]))
	return
}

// ParseAuthorizedKeys parses a public key from an authorized_keys
// file used in OpenSSH according to the sshd(8) manual page.
func ParseAuthorizedKey(in []byte) (out PublicKey, comment string, options []string, rest []byte, ok bool) {
	for len(in) > 0 {
		end := bytes.IndexByte(in, '\n')
		if end != -1 {
			rest = in[end+1:]
			in = in[:end]
		} else {
			rest = nil
		}

		end = bytes.IndexByte(in, '\r')
		if end != -1 {
			in = in[:end]
		}

		in = bytes.TrimSpace(in)
		if len(in) == 0 || in[0] == '#' {
			in = rest
			continue
		}

		i := bytes.IndexAny(in, " \t")
		if i == -1 {
			in = rest
			continue
		}

		if out, comment, ok = parseAuthorizedKey(in[i:]); ok {
			return
		}

		// No key type recognised. Maybe there's an options field at
		// the beginning.
		var b byte
		inQuote := false
		var candidateOptions []string
		optionStart := 0
		for i, b = range in {
			isEnd := !inQuote && (b == ' ' || b == '\t')
			if (b == ',' && !inQuote) || isEnd {
				if i-optionStart > 0 {
					candidateOptions = append(candidateOptions, string(in[optionStart:i]))
				}
				optionStart = i + 1
			}
			if isEnd {
				break
			}
			if b == '"' && (i == 0 || (i > 0 && in[i-1] != '\\')) {
				inQuote = !inQuote
			}
		}
		for i < len(in) && (in[i] == ' ' || in[i] == '\t') {
			i++
		}
		if i == len(in) {
			// Invalid line: unmatched quote
			in = rest
			continue
		}

		in = in[i:]
		i = bytes.IndexAny(in, " \t")
		if i == -1 {
			in = rest
			continue
		}

		if out, comment, ok = parseAuthorizedKey(in[i:]); ok {
			options = candidateOptions
			return
		}

		in = rest
		continue
	}

	return
}

// ParsePublicKey parses an SSH public key formatted for use in
// the SSH wire protocol according to RFC 4253, section 6.6.
func ParsePublicKey(in []byte) (out PublicKey, rest []byte, ok bool) {
	algo, in, ok := parseString(in)
	if !ok {
		return
	}

	return parsePubKey(in, string(algo))
}

// MarshalAuthorizedKey returns a byte stream suitable for inclusion
// in an OpenSSH authorized_keys file following the format specified
// in the sshd(8) manual page.
func MarshalAuthorizedKey(key PublicKey) []byte {
	b := &bytes.Buffer{}
	b.WriteString(key.PublicKeyAlgo())
	b.WriteByte(' ')
	e := base64.NewEncoder(base64.StdEncoding, b)
	e.Write(MarshalPublicKey(key))
	e.Close()
	b.WriteByte('\n')
	return b.Bytes()
}

// PublicKey is an abstraction of different types of public keys.
type PublicKey interface {
	// PrivateKeyAlgo returns the name of the encryption system.
	PrivateKeyAlgo() string

	// PublicKeyAlgo returns the algorithm for the public key,
	// which may be different from PrivateKeyAlgo for certificates.
	PublicKeyAlgo() string

	// Marshal returns the serialized key data in SSH wire format,
	// without the name prefix.  Callers should typically use
	// MarshalPublicKey().
	Marshal() []byte

	// Verify that sig is a signature on the given data using this
	// key. This function will hash the data appropriately first.
	Verify(data []byte, sigBlob []byte) bool
}

// A Signer is can create signatures that verify against a public key.
type Signer interface {
	// PublicKey returns an associated PublicKey instance.
	PublicKey() PublicKey

	// Sign returns raw signature for the given data. This method
	// will apply the hash specified for the keytype to the data.
	Sign(rand io.Reader, data []byte) ([]byte, error)
}

type rsaPublicKey rsa.PublicKey

func (r *rsaPublicKey) PrivateKeyAlgo() string {
	return "ssh-rsa"
}

func (r *rsaPublicKey) PublicKeyAlgo() string {
	return r.PrivateKeyAlgo()
}

// parseRSA parses an RSA key according to RFC 4253, section 6.6.
func parseRSA(in []byte) (out PublicKey, rest []byte, ok bool) {
	key := new(rsa.PublicKey)

	bigE, in, ok := parseInt(in)
	if !ok || bigE.BitLen() > 24 {
		return
	}
	e := bigE.Int64()
	if e < 3 || e&1 == 0 {
		ok = false
		return
	}
	key.E = int(e)

	if key.N, in, ok = parseInt(in); !ok {
		return
	}

	ok = true
	return (*rsaPublicKey)(key), in, ok
}

func (r *rsaPublicKey) Marshal() []byte {
	// See RFC 4253, section 6.6.
	e := new(big.Int).SetInt64(int64(r.E))
	length := intLength(e)
	length += intLength(r.N)

	ret := make([]byte, length)
	rest := marshalInt(ret, e)
	marshalInt(rest, r.N)

	return ret
}

func (r *rsaPublicKey) Verify(data []byte, sig []byte) bool {
	h := crypto.SHA1.New()
	h.Write(data)
	digest := h.Sum(nil)
	return rsa.VerifyPKCS1v15((*rsa.PublicKey)(r), crypto.SHA1, digest, sig) == nil
}

type rsaPrivateKey struct {
	*rsa.PrivateKey
}

func (r *rsaPrivateKey) PublicKey() PublicKey {
	return (*rsaPublicKey)(&r.PrivateKey.PublicKey)
}

func (r *rsaPrivateKey) Sign(rand io.Reader, data []byte) ([]byte, error) {
	h := crypto.SHA1.New()
	h.Write(data)
	digest := h.Sum(nil)
	return rsa.SignPKCS1v15(rand, r.PrivateKey, crypto.SHA1, digest)
}

type dsaPublicKey dsa.PublicKey

func (r *dsaPublicKey) PrivateKeyAlgo() string {
	return "ssh-dss"
}

func (r *dsaPublicKey) PublicKeyAlgo() string {
	return r.PrivateKeyAlgo()
}

// parseDSA parses an DSA key according to RFC 4253, section 6.6.
func parseDSA(in []byte) (out PublicKey, rest []byte, ok bool) {
	key := new(dsa.PublicKey)

	if key.P, in, ok = parseInt(in); !ok {
		return
	}

	if key.Q, in, ok = parseInt(in); !ok {
		return
	}

	if key.G, in, ok = parseInt(in); !ok {
		return
	}

	if key.Y, in, ok = parseInt(in); !ok {
		return
	}

	ok = true
	return (*dsaPublicKey)(key), in, ok
}

func (r *dsaPublicKey) Marshal() []byte {
	// See RFC 4253, section 6.6.
	length := intLength(r.P)
	length += intLength(r.Q)
	length += intLength(r.G)
	length += intLength(r.Y)

	ret := make([]byte, length)
	rest := marshalInt(ret, r.P)
	rest = marshalInt(rest, r.Q)
	rest = marshalInt(rest, r.G)
	marshalInt(rest, r.Y)

	return ret
}

func (k *dsaPublicKey) Verify(data []byte, sigBlob []byte) bool {
	h := crypto.SHA1.New()
	h.Write(data)
	digest := h.Sum(nil)

	// Per RFC 4253, section 6.6,
	// The value for 'dss_signature_blob' is encoded as a string containing
	// r, followed by s (which are 160-bit integers, without lengths or
	// padding, unsigned, and in network byte order).
	// For DSS purposes, sig.Blob should be exactly 40 bytes in length.
	if len(sigBlob) != 40 {
		return false
	}
	r := new(big.Int).SetBytes(sigBlob[:20])
	s := new(big.Int).SetBytes(sigBlob[20:])
	return dsa.Verify((*dsa.PublicKey)(k), digest, r, s)
}

type dsaPrivateKey struct {
	*dsa.PrivateKey
}

func (k *dsaPrivateKey) PublicKey() PublicKey {
	return (*dsaPublicKey)(&k.PrivateKey.PublicKey)
}

func (k *dsaPrivateKey) Sign(rand io.Reader, data []byte) ([]byte, error) {
	h := crypto.SHA1.New()
	h.Write(data)
	digest := h.Sum(nil)
	r, s, err := dsa.Sign(rand, k.PrivateKey, digest)
	if err != nil {
		return nil, err
	}

	sig := make([]byte, 40)
	copy(sig[:20], r.Bytes())
	copy(sig[20:], s.Bytes())
	return sig, nil
}

type ecdsaPublicKey ecdsa.PublicKey

func (key *ecdsaPublicKey) PrivateKeyAlgo() string {
	return "ecdsa-sha2-" + key.nistID()
}

func (key *ecdsaPublicKey) nistID() string {
	switch key.Params().BitSize {
	case 256:
		return "nistp256"
	case 384:
		return "nistp384"
	case 521:
		return "nistp521"
	}
	panic("ssh: unsupported ecdsa key size")
}

func supportedEllipticCurve(curve elliptic.Curve) bool {
	return (curve == elliptic.P256() || curve == elliptic.P384() || curve == elliptic.P521())
}

// ecHash returns the hash to match the given elliptic curve, see RFC
// 5656, section 6.2.1
func ecHash(curve elliptic.Curve) crypto.Hash {
	bitSize := curve.Params().BitSize
	switch {
	case bitSize <= 256:
		return crypto.SHA256
	case bitSize <= 384:
		return crypto.SHA384
	}
	return crypto.SHA512
}

func (key *ecdsaPublicKey) PublicKeyAlgo() string {
	return key.PrivateKeyAlgo()
}

// parseECDSA parses an ECDSA key according to RFC 5656, section 3.1.
func parseECDSA(in []byte) (out PublicKey, rest []byte, ok bool) {
	var identifier []byte
	if identifier, in, ok = parseString(in); !ok {
		return
	}

	key := new(ecdsa.PublicKey)

	switch string(identifier) {
	case "nistp256":
		key.Curve = elliptic.P256()
	case "nistp384":
		key.Curve = elliptic.P384()
	case "nistp521":
		key.Curve = elliptic.P521()
	default:
		ok = false
		return
	}

	var keyBytes []byte
	if keyBytes, in, ok = parseString(in); !ok {
		return
	}

	key.X, key.Y = elliptic.Unmarshal(key.Curve, keyBytes)
	if key.X == nil || key.Y == nil {
		ok = false
		return
	}
	return (*ecdsaPublicKey)(key), in, ok
}

func (key *ecdsaPublicKey) Marshal() []byte {
	// See RFC 5656, section 3.1.
	keyBytes := elliptic.Marshal(key.Curve, key.X, key.Y)

	ID := key.nistID()
	length := stringLength(len(ID))
	length += stringLength(len(keyBytes))

	ret := make([]byte, length)
	r := marshalString(ret, []byte(ID))
	r = marshalString(r, keyBytes)
	return ret
}

func (key *ecdsaPublicKey) Verify(data []byte, sigBlob []byte) bool {
	h := ecHash(key.Curve).New()
	h.Write(data)
	digest := h.Sum(nil)

	// Per RFC 5656, section 3.1.2,
	// The ecdsa_signature_blob value has the following specific encoding:
	//    mpint    r
	//    mpint    s
	r, rest, ok := parseInt(sigBlob)
	if !ok {
		return false
	}
	s, rest, ok := parseInt(rest)
	if !ok || len(rest) > 0 {
		return false
	}
	return ecdsa.Verify((*ecdsa.PublicKey)(key), digest, r, s)
}

type ecdsaPrivateKey struct {
	*ecdsa.PrivateKey
}

func (k *ecdsaPrivateKey) PublicKey() PublicKey {
	return (*ecdsaPublicKey)(&k.PrivateKey.PublicKey)
}

func (k *ecdsaPrivateKey) Sign(rand io.Reader, data []byte) ([]byte, error) {
	h := ecHash(k.PrivateKey.PublicKey.Curve).New()
	h.Write(data)
	digest := h.Sum(nil)
	r, s, err := ecdsa.Sign(rand, k.PrivateKey, digest)
	if err != nil {
		return nil, err
	}

	sig := make([]byte, intLength(r)+intLength(s))
	rest := marshalInt(sig, r)
	marshalInt(rest, s)
	return sig, nil
}

// NewPrivateKey takes a pointer to rsa, dsa or ecdsa PrivateKey
// returns a corresponding Signer instance. EC keys should use P256,
// P384 or P521.
func NewSignerFromKey(k interface{}) (Signer, error) {
	var sshKey Signer
	switch t := k.(type) {
	case *rsa.PrivateKey:
		sshKey = &rsaPrivateKey{t}
	case *dsa.PrivateKey:
		sshKey = &dsaPrivateKey{t}
	case *ecdsa.PrivateKey:
		if !supportedEllipticCurve(t.Curve) {
			return nil, errors.New("ssh: only P256, P384 and P521 EC keys are supported.")
		}

		sshKey = &ecdsaPrivateKey{t}
	default:
		return nil, fmt.Errorf("ssh: unsupported key type %T", k)
	}
	return sshKey, nil
}

// NewPublicKey takes a pointer to rsa, dsa or ecdsa PublicKey
// and returns a corresponding ssh PublicKey instance. EC keys should use P256, P384 or P521.
func NewPublicKey(k interface{}) (PublicKey, error) {
	var sshKey PublicKey
	switch t := k.(type) {
	case *rsa.PublicKey:
		sshKey = (*rsaPublicKey)(t)
	case *ecdsa.PublicKey:
		if !supportedEllipticCurve(t.Curve) {
			return nil, errors.New("ssh: only P256, P384 and P521 EC keys are supported.")
		}
		sshKey = (*ecdsaPublicKey)(t)
	case *dsa.PublicKey:
		sshKey = (*dsaPublicKey)(t)
	default:
		return nil, fmt.Errorf("ssh: unsupported key type %T", k)
	}
	return sshKey, nil
}

// ParsePublicKey parses a PEM encoded private key. It supports
// PKCS#1, RSA, DSA and ECDSA private keys.
func ParsePrivateKey(pemBytes []byte) (Signer, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("ssh: no key found")
	}

	var rawkey interface{}
	switch block.Type {
	case "RSA PRIVATE KEY":
		rsa, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rawkey = rsa
	case "EC PRIVATE KEY":
		ec, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rawkey = ec
	case "DSA PRIVATE KEY":
		ec, err := parseDSAPrivate(block.Bytes)
		if err != nil {
			return nil, err
		}
		rawkey = ec
	default:
		return nil, fmt.Errorf("ssh: unsupported key type %q", block.Type)
	}

	return NewSignerFromKey(rawkey)
}

// parseDSAPrivate parses a DSA key in ASN.1 DER encoding, as
// documented in the OpenSSL DSA manpage.
// TODO(hanwen): move this in to crypto/x509 after the Go 1.2 freeze.
func parseDSAPrivate(p []byte) (*dsa.PrivateKey, error) {
	k := struct {
		Version int
		P       *big.Int
		Q       *big.Int
		G       *big.Int
		Priv    *big.Int
		Pub     *big.Int
	}{}
	rest, err := asn1.Unmarshal(p, &k)
	if err != nil {
		return nil, errors.New("ssh: failed to parse DSA key: " + err.Error())
	}
	if len(rest) > 0 {
		return nil, errors.New("ssh: garbage after DSA key")
	}

	return &dsa.PrivateKey{
		PublicKey: dsa.PublicKey{
			Parameters: dsa.Parameters{
				P: k.P,
				Q: k.Q,
				G: k.G,
			},
			Y: k.Priv,
		},
		X: k.Pub,
	}, nil
}
