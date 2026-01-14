// Source: https://github.com/goharbor/harbor

// Copyright 2016 Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http

import (
	"crypto/tls"
	"fmt"
	"os"
	"strings"
)

const (
	// Internal TLS ENV.
	internalTLSEnable        = "GITNESS_INTERNAL_TLS_ENABLED"
	internalVerifyClientCert = "GITNESS_INTERNAL_VERIFY_CLIENT_CERT"
	internalTLSKeyPath       = "GITNESS_INTERNAL_TLS_KEY_PATH"
	internalTLSCertPath      = "GITNESS_INTERNAL_TLS_CERT_PATH"
)

// InternalTLSEnabled returns true if internal TLS enabled.
func InternalTLSEnabled() bool {
	return strings.ToLower(os.Getenv(internalTLSEnable)) == "true"
}

// InternalEnableVerifyClientCert returns true if mTLS enabled.
func InternalEnableVerifyClientCert() bool {
	return strings.ToLower(os.Getenv(internalVerifyClientCert)) == "true"
}

// GetInternalCertPair used to get internal cert and key pair from environment.
func GetInternalCertPair() (tls.Certificate, error) {
	crtPath := os.Getenv(internalTLSCertPath)
	keyPath := os.Getenv(internalTLSKeyPath)
	return tls.LoadX509KeyPair(crtPath, keyPath)
}

// GetInternalTLSConfig return a tls.Config for internal https communicate.
func GetInternalTLSConfig() (*tls.Config, error) {
	// genrate key pair
	cert, err := GetInternalCertPair()
	if err != nil {
		return nil, fmt.Errorf("internal TLS enabled but can't get cert file %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// NewServerTLSConfig returns a modern tls config,
// refer to https://blog.cloudflare.com/exposing-go-on-the-internet/
func NewServerTLSConfig() *tls.Config {
	return &tls.Config{
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}
}
