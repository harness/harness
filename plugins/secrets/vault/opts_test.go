// Copyright 2018 Drone.IO Inc
// Use of this software is governed by the Drone Enterpise License
// that can be found in the LICENSE file.

package vault

import (
	"testing"
	"time"
)

func TestWithTTL(t *testing.T) {
	v := new(vault)
	opt := WithTTL(time.Hour)
	opt(v)
	if got, want := v.ttl, time.Hour; got != want {
		t.Errorf("Want ttl %v, got %v", want, got)
	}
}

func TestWithRenewal(t *testing.T) {
	v := new(vault)
	opt := WithRenewal(time.Hour)
	opt(v)
	if got, want := v.renew, time.Hour; got != want {
		t.Errorf("Want renewal %v, got %v", want, got)
	}
}
