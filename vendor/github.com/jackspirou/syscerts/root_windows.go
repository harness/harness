// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syscerts

import (
	"crypto/x509"
	"errors"
	"syscall"
	"unsafe"
)

// extractSimpleChain extracts the final certificate chain from a CertSimpleChain.
func extractSimpleChain(simpleChain **syscall.CertSimpleChain, count int) (chain []*x509.Certificate, err error) {
	if simpleChain == nil || count == 0 {
		return nil, errors.New("x509: invalid simple chain")
	}

	simpleChains := (*[1 << 20]*syscall.CertSimpleChain)(unsafe.Pointer(simpleChain))[:]
	lastChain := simpleChains[count-1]
	elements := (*[1 << 20]*syscall.CertChainElement)(unsafe.Pointer(lastChain.Elements))[:]
	for i := 0; i < int(lastChain.NumElements); i++ {
		// Copy the buf, since ParseCertificate does not create its own copy.
		cert := elements[i].CertContext
		encodedCert := (*[1 << 20]byte)(unsafe.Pointer(cert.EncodedCert))[:]
		buf := make([]byte, cert.Length)
		copy(buf, encodedCert[:])
		parsedCert, err := x509.ParseCertificate(buf)
		if err != nil {
			return nil, err
		}
		chain = append(chain, parsedCert)
	}

	return chain, nil
}

func initSystemRoots() {
}
