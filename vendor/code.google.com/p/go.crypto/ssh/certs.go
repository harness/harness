// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssh

import (
	"time"
)

// These constants from [PROTOCOL.certkeys] represent the algorithm names
// for certificate types supported by this package.
const (
	CertAlgoRSAv01      = "ssh-rsa-cert-v01@openssh.com"
	CertAlgoDSAv01      = "ssh-dss-cert-v01@openssh.com"
	CertAlgoECDSA256v01 = "ecdsa-sha2-nistp256-cert-v01@openssh.com"
	CertAlgoECDSA384v01 = "ecdsa-sha2-nistp384-cert-v01@openssh.com"
	CertAlgoECDSA521v01 = "ecdsa-sha2-nistp521-cert-v01@openssh.com"
)

// Certificate types are used to specify whether a certificate is for identification
// of a user or a host.  Current identities are defined in [PROTOCOL.certkeys].
const (
	UserCert = 1
	HostCert = 2
)

type signature struct {
	Format string
	Blob   []byte
}

type tuple struct {
	Name string
	Data string
}

const (
	maxUint64 = 1<<64 - 1
	maxInt64  = 1<<63 - 1
)

// CertTime represents an unsigned 64-bit time value in seconds starting from
// UNIX epoch.  We use CertTime instead of time.Time in order to properly handle
// the "infinite" time value ^0, which would become negative when expressed as
// an int64.
type CertTime uint64

func (ct CertTime) Time() time.Time {
	if ct > maxInt64 {
		return time.Unix(maxInt64, 0)
	}
	return time.Unix(int64(ct), 0)
}

func (ct CertTime) IsInfinite() bool {
	return ct == maxUint64
}

// An OpenSSHCertV01 represents an OpenSSH certificate as defined in
// [PROTOCOL.certkeys]?rev=1.8.
type OpenSSHCertV01 struct {
	Nonce                   []byte
	Key                     PublicKey
	Serial                  uint64
	Type                    uint32
	KeyId                   string
	ValidPrincipals         []string
	ValidAfter, ValidBefore CertTime
	CriticalOptions         []tuple
	Extensions              []tuple
	Reserved                []byte
	SignatureKey            PublicKey
	Signature               *signature
}

// validateOpenSSHCertV01Signature uses the cert's SignatureKey to verify that
// the cert's Signature.Blob is the result of signing the cert bytes starting
// from the algorithm string and going up to and including the SignatureKey.
func validateOpenSSHCertV01Signature(cert *OpenSSHCertV01) bool {
	return cert.SignatureKey.Verify(cert.BytesForSigning(), cert.Signature.Blob)
}

var certAlgoNames = map[string]string{
	KeyAlgoRSA:      CertAlgoRSAv01,
	KeyAlgoDSA:      CertAlgoDSAv01,
	KeyAlgoECDSA256: CertAlgoECDSA256v01,
	KeyAlgoECDSA384: CertAlgoECDSA384v01,
	KeyAlgoECDSA521: CertAlgoECDSA521v01,
}

// certToPrivAlgo returns the underlying algorithm for a certificate algorithm.
// Panics if a non-certificate algorithm is passed.
func certToPrivAlgo(algo string) string {
	for privAlgo, pubAlgo := range certAlgoNames {
		if pubAlgo == algo {
			return privAlgo
		}
	}
	panic("unknown cert algorithm")
}

func (cert *OpenSSHCertV01) marshal(includeAlgo, includeSig bool) []byte {
	algoName := cert.PublicKeyAlgo()
	pubKey := cert.Key.Marshal()
	sigKey := MarshalPublicKey(cert.SignatureKey)

	var length int
	if includeAlgo {
		length += stringLength(len(algoName))
	}
	length += stringLength(len(cert.Nonce))
	length += len(pubKey)
	length += 8 // Length of Serial
	length += 4 // Length of Type
	length += stringLength(len(cert.KeyId))
	length += lengthPrefixedNameListLength(cert.ValidPrincipals)
	length += 8 // Length of ValidAfter
	length += 8 // Length of ValidBefore
	length += tupleListLength(cert.CriticalOptions)
	length += tupleListLength(cert.Extensions)
	length += stringLength(len(cert.Reserved))
	length += stringLength(len(sigKey))
	if includeSig {
		length += signatureLength(cert.Signature)
	}

	ret := make([]byte, length)
	r := ret
	if includeAlgo {
		r = marshalString(r, []byte(algoName))
	}
	r = marshalString(r, cert.Nonce)
	copy(r, pubKey)
	r = r[len(pubKey):]
	r = marshalUint64(r, cert.Serial)
	r = marshalUint32(r, cert.Type)
	r = marshalString(r, []byte(cert.KeyId))
	r = marshalLengthPrefixedNameList(r, cert.ValidPrincipals)
	r = marshalUint64(r, uint64(cert.ValidAfter))
	r = marshalUint64(r, uint64(cert.ValidBefore))
	r = marshalTupleList(r, cert.CriticalOptions)
	r = marshalTupleList(r, cert.Extensions)
	r = marshalString(r, cert.Reserved)
	r = marshalString(r, sigKey)
	if includeSig {
		r = marshalSignature(r, cert.Signature)
	}
	if len(r) > 0 {
		panic("ssh: internal error, marshaling certificate did not fill the entire buffer")
	}
	return ret
}

func (cert *OpenSSHCertV01) BytesForSigning() []byte {
	return cert.marshal(true, false)
}

func (cert *OpenSSHCertV01) Marshal() []byte {
	return cert.marshal(false, true)
}

func (c *OpenSSHCertV01) PublicKeyAlgo() string {
	algo, ok := certAlgoNames[c.Key.PublicKeyAlgo()]
	if !ok {
		panic("unknown cert key type")
	}
	return algo
}

func (c *OpenSSHCertV01) PrivateKeyAlgo() string {
	return c.Key.PrivateKeyAlgo()
}

func (c *OpenSSHCertV01) Verify(data []byte, sig []byte) bool {
	return c.Key.Verify(data, sig)
}

func parseOpenSSHCertV01(in []byte, algo string) (out *OpenSSHCertV01, rest []byte, ok bool) {
	cert := new(OpenSSHCertV01)

	if cert.Nonce, in, ok = parseString(in); !ok {
		return
	}

	privAlgo := certToPrivAlgo(algo)
	cert.Key, in, ok = parsePubKey(in, privAlgo)
	if !ok {
		return
	}

	// We test PublicKeyAlgo to make sure we don't use some weird sub-cert.
	if cert.Key.PublicKeyAlgo() != privAlgo {
		ok = false
		return
	}

	if cert.Serial, in, ok = parseUint64(in); !ok {
		return
	}

	if cert.Type, in, ok = parseUint32(in); !ok {
		return
	}

	keyId, in, ok := parseString(in)
	if !ok {
		return
	}
	cert.KeyId = string(keyId)

	if cert.ValidPrincipals, in, ok = parseLengthPrefixedNameList(in); !ok {
		return
	}

	va, in, ok := parseUint64(in)
	if !ok {
		return
	}
	cert.ValidAfter = CertTime(va)

	vb, in, ok := parseUint64(in)
	if !ok {
		return
	}
	cert.ValidBefore = CertTime(vb)

	if cert.CriticalOptions, in, ok = parseTupleList(in); !ok {
		return
	}

	if cert.Extensions, in, ok = parseTupleList(in); !ok {
		return
	}

	if cert.Reserved, in, ok = parseString(in); !ok {
		return
	}

	sigKey, in, ok := parseString(in)
	if !ok {
		return
	}
	if cert.SignatureKey, _, ok = ParsePublicKey(sigKey); !ok {
		return
	}

	if cert.Signature, in, ok = parseSignature(in); !ok {
		return
	}

	ok = true
	return cert, in, ok
}

func lengthPrefixedNameListLength(namelist []string) int {
	length := 4 // length prefix for list
	for _, name := range namelist {
		length += 4 // length prefix for name
		length += len(name)
	}
	return length
}

func marshalLengthPrefixedNameList(to []byte, namelist []string) []byte {
	length := uint32(lengthPrefixedNameListLength(namelist) - 4)
	to = marshalUint32(to, length)
	for _, name := range namelist {
		to = marshalString(to, []byte(name))
	}
	return to
}

func parseLengthPrefixedNameList(in []byte) (out []string, rest []byte, ok bool) {
	list, rest, ok := parseString(in)
	if !ok {
		return
	}

	for len(list) > 0 {
		var next []byte
		if next, list, ok = parseString(list); !ok {
			return nil, nil, false
		}
		out = append(out, string(next))
	}
	ok = true
	return
}

func tupleListLength(tupleList []tuple) int {
	length := 4 // length prefix for list
	for _, t := range tupleList {
		length += 4 // length prefix for t.Name
		length += len(t.Name)
		length += 4 // length prefix for t.Data
		length += len(t.Data)
	}
	return length
}

func marshalTupleList(to []byte, tuplelist []tuple) []byte {
	length := uint32(tupleListLength(tuplelist) - 4)
	to = marshalUint32(to, length)
	for _, t := range tuplelist {
		to = marshalString(to, []byte(t.Name))
		to = marshalString(to, []byte(t.Data))
	}
	return to
}

func parseTupleList(in []byte) (out []tuple, rest []byte, ok bool) {
	list, rest, ok := parseString(in)
	if !ok {
		return
	}

	for len(list) > 0 {
		var name, data []byte
		var ok bool
		name, list, ok = parseString(list)
		if !ok {
			return nil, nil, false
		}
		data, list, ok = parseString(list)
		if !ok {
			return nil, nil, false
		}
		out = append(out, tuple{string(name), string(data)})
	}
	ok = true
	return
}

func signatureLength(sig *signature) int {
	length := 4 // length prefix for signature
	length += stringLength(len(sig.Format))
	length += stringLength(len(sig.Blob))
	return length
}

func marshalSignature(to []byte, sig *signature) []byte {
	length := uint32(signatureLength(sig) - 4)
	to = marshalUint32(to, length)
	to = marshalString(to, []byte(sig.Format))
	to = marshalString(to, sig.Blob)
	return to
}

func parseSignatureBody(in []byte) (out *signature, rest []byte, ok bool) {
	var format []byte
	if format, in, ok = parseString(in); !ok {
		return
	}

	out = &signature{
		Format: string(format),
	}

	if out.Blob, in, ok = parseString(in); !ok {
		return
	}

	return out, in, ok
}

func parseSignature(in []byte) (out *signature, rest []byte, ok bool) {
	var sigBytes []byte
	if sigBytes, rest, ok = parseString(in); !ok {
		return
	}

	out, sigBytes, ok = parseSignatureBody(sigBytes)
	if !ok || len(sigBytes) > 0 {
		return nil, nil, false
	}
	return
}
