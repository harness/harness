package main

import (
	"encoding/json"
	"os"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/citadel/citadel"
)

type Config struct {
	SSLCertificate string            `json:"ssl-cert,omitempty"`
	SSLKey         string            `json:"ssl-key,omitempty"`
	CACertificate  string            `json:"ca-cert,omitempty"`
	ListenAddr     string            `json:"listen-addr,omitempty"`
	Engines        []*citadel.Engine `json:"engines,omitempty"`
}

func loadConfig() error {
	f, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewDecoder(f).Decode(&config)
}
