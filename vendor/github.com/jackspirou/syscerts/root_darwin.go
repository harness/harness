// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run root_darwin_arm_gen.go -output root_darwin_armx.go

package syscerts

import (
	"crypto/x509"
	"os"
	"os/exec"
)

func execSecurityRoots() (*x509.CertPool, error) {
	roots := x509.NewCertPool()
	cmd := exec.Command("/usr/bin/security", "find-certificate", "-a", "-p", "/System/Library/Keychains/SystemRootCertificates.keychain")
	data, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	roots.AppendCertsFromPEM(data)

	// if available add the Mac OSX System Keychain
	if _, err := os.Stat("/Library/Keychains/System.keychain"); err == nil {
		cmd = exec.Command("/usr/bin/security", "find-certificate", "-a", "-p", "/Library/Keychains/System.keychain")
		data, err = cmd.Output()
		if err != nil {
			return nil, err
		}
		roots.AppendCertsFromPEM(data)
	}

	return roots, nil
}

func initSystemRoots() {
	systemRoots, _ = execSecurityRoots()
}
