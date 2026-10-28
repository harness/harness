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

package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

func TestProvideRedisTLSDisabled(t *testing.T) {
	client, err := ProvideRedis(&types.Config{})
	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestRedisTLSConfigSystemPool(t *testing.T) {
	config := &types.Config{}
	config.Redis.SSLEnabled = true

	tlsConfig, err := redisTLSConfig(config)
	require.NoError(t, err)
	require.NotNil(t, tlsConfig)
	require.Nil(t, tlsConfig.RootCAs)
	require.Equal(t, uint16(tls.VersionTLS12), tlsConfig.MinVersion)
}

func TestRedisTLSConfigPEM(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "ca.crt")
	require.NoError(t, os.WriteFile(good, testCertPEM(t), 0o600))

	config := &types.Config{}
	config.Redis.SSLEnabled = true
	config.Redis.CertPath = good

	tlsConfig, err := redisTLSConfig(config)
	require.NoError(t, err)
	require.NotNil(t, tlsConfig.RootCAs)

	config.Redis.CertPath = filepath.Join(dir, "missing.crt")
	_, err = redisTLSConfig(config)
	require.Error(t, err)

	bad := filepath.Join(dir, "bad.crt")
	require.NoError(t, os.WriteFile(bad, []byte("not a cert"), 0o600))
	config.Redis.CertPath = bad
	_, err = redisTLSConfig(config)
	require.Error(t, err)
}

func testCertPEM(t *testing.T) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "redis-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IsCA:         true,
		KeyUsage:     x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}
