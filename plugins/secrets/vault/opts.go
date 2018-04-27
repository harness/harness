// Copyright 2018 Drone.IO Inc
// Use of this software is governed by the Drone Enterpise License
// that can be found in the LICENSE file.

package vault

import "time"

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

// WithAuth returns an options that sets the vault
// method to use for authentication
func WithAuth(method string) Opts {
	return func(v *vault) {
		v.auth = method
	}
}

// WithKubernetes returns an options that sets
// kubernetes-auth parameters required to retrieve
// an initial vault token
func WithKubernetesAuth(addr, role, mount string) Opts {
	return func(v *vault) {
		v.kubeAuth.addr = addr
		v.kubeAuth.role = role
		v.kubeAuth.mount = mount
	}
}
