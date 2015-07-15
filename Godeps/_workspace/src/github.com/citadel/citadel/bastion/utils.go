package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/url"
	"os"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/citadel/citadel"
)

func getTLSConfig() (*tls.Config, error) {
	// TLS config
	var tlsConfig tls.Config
	tlsConfig.InsecureSkipVerify = true
	certPool := x509.NewCertPool()

	file, err := ioutil.ReadFile(config.CACertificate)
	if err != nil {
		return nil, err
	}

	certPool.AppendCertsFromPEM(file)
	tlsConfig.RootCAs = certPool
	_, errCert := os.Stat(config.SSLCertificate)
	_, errKey := os.Stat(config.SSLKey)
	if errCert == nil && errKey == nil {
		cert, err := tls.LoadX509KeyPair(config.SSLCertificate, config.SSLKey)
		if err != nil {
			return &tlsConfig, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return &tlsConfig, nil
}

func setEngineClient(docker *citadel.Engine, tlsConfig *tls.Config) error {
	var tc *tls.Config
	u, err := url.Parse(docker.Addr)
	if err != nil {
		return err
	}

	if u.Scheme == "https" {
		tc = tlsConfig
	}

	return docker.Connect(tc)
}
