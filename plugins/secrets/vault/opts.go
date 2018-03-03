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
