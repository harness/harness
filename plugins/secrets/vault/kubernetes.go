package vault

import (
	"fmt"
	"github.com/drone/drone/plugins/internal"
	"io/ioutil"
	"time"
)

/*
Vault JSON Response
{
  "auth": {
    "client_token" = "token",
    "lease_duration" = "1234s"
  }
}
*/
type vaultAuth struct {
	Token string `json:"client_token"`
	Lease string `json:"lease_duration"`
}
type vaultResp struct {
	Auth vaultAuth
}

func getKubernetesToken(addr, role, mount, tokenFile string) (string, time.Duration, error) {
	b, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return "", 0, err
	}

	var resp vaultResp
	path := fmt.Sprintf("%s/v1/auth/%s/login", addr, mount)
	data := map[string]string{
		"jwt":  string(b),
		"role": role,
	}

	err = internal.Send("POST", path, data, &resp)
	if err != nil {
		return "", 0, err
	}

	ttl, err := time.ParseDuration(resp.Auth.Lease)
	if err != nil {
		return "", 0, err
	}

	return resp.Auth.Token, ttl, nil
}
