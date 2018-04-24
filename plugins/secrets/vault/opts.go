// Copyright 2018 Drone.IO Inc
// Use of this software is governed by the Drone Enterpise License
// that can be found in the LICENSE file.

package vault

import (
	"github.com/Sirupsen/logrus"
	"os"
	"time"
)

// Opts sets custom options for the vault client.
type Opts func(v *vault)

// WithTTL returns an options that sets a TTL used to
// refresh periodic tokens.
func WithTTL(d time.Duration) Opts {
	return func(v *vault) {
		v.ttl = d
	}
}

// WithRenewal returns an options that sets the renewal
// period used to refresh periodic tokens
func WithRenewal(d time.Duration) Opts {
	return func(v *vault) {
		v.renew = d
	}
}

func WithKubernetesAuth() Opts {
	return func(v *vault) {
		addr := os.Getenv("VAULT_ADDR")
		role := os.Getenv("DRONE_VAULT_KUBERNETES_ROLE")
		mount := os.Getenv("DRONE_VAULT_AUTH_MOUNT_POINT")
		jwtFile := "/var/run/secrets/kubernetes.io/serviceaccount/token"
		token, ttl, err := getKubernetesToken(addr, role, mount, jwtFile)
		if err != nil {
			logrus.Debugf("vault: failed to obtain token via kubernetes-auth backend: %s", err)
			return
		}

		v.client.SetToken(token)
		v.ttl = ttl
		v.renew = ttl / 2
	}
}
