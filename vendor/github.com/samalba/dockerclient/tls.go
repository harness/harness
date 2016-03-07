package dockerclient

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"path/filepath"
)

// TLSConfigFromCertPath returns a configuration based on PEM files in the directory
//
// path is usually what is set by the environment variable `DOCKER_CERT_PATH`,
// or `$HOME/.docker`.
func TLSConfigFromCertPath(path string) (*tls.Config, error) {
	cert, err := ioutil.ReadFile(filepath.Join(path, "cert.pem"))
	if err != nil {
		return nil, err
	}
	key, err := ioutil.ReadFile(filepath.Join(path, "key.pem"))
	if err != nil {
		return nil, err
	}
	ca, err := ioutil.ReadFile(filepath.Join(path, "ca.pem"))
	if err != nil {
		return nil, err
	}
	tlsCert, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{tlsCert}}
	tlsConfig.RootCAs = x509.NewCertPool()
	if !tlsConfig.RootCAs.AppendCertsFromPEM(ca) {
		return nil, errors.New("Could not add RootCA pem")
	}
	return tlsConfig, nil
}
