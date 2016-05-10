package dockerclient

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
)

// AuthConfig hold parameters for authenticating with the docker registry
type AuthConfig struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Email         string `json:"email,omitempty"`
	RegistryToken string `json:"registrytoken,omitempty"`
}

// encode the auth configuration struct into base64 for the X-Registry-Auth header
func (c *AuthConfig) encode() (string, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(c); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf.Bytes()), nil
}

// ConfigFile holds parameters for authenticating during a BuildImage request
type ConfigFile struct {
	Configs  map[string]AuthConfig `json:"configs,omitempty"`
	rootPath string
}

// encode the configuration struct into base64 for the X-Registry-Config header
func (c *ConfigFile) encode() (string, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(c); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf.Bytes()), nil
}
