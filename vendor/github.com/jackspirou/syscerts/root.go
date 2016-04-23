// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syscerts

import (
	"crypto/x509"
	"sync"
)

var (
	once        sync.Once
	systemRoots *x509.CertPool
)

// SystemRootsPool attempts to find and return a pool of all all installed
// system certificates.
func SystemRootsPool() *x509.CertPool {
	once.Do(initSystemRoots)
	return systemRoots
}
