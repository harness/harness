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

func TestWithAuth(t *testing.T) {
	v := new(vault)
	method := "kubernetes"
	opt := WithAuth(method)
	opt(v)
	if got, want := v.auth, method; got != want {
		t.Errorf("Want auth %v, got %v", want, got)
	}
}

func TestWithKubernetesAuth(t *testing.T) {
	v := new(vault)
	addr := "https://address.fake"
	role := "fakeRole"
	mount := "kubernetes"
	opt := WithKubernetesAuth(addr, role, mount)
	opt(v)
	if got, want := v.kubeAuth.addr, addr; got != want {
		t.Errorf("Want addr %v, got %v", want, got)
	}
	if got, want := v.kubeAuth.role, role; got != want {
		t.Errorf("Want role %v, got %v", want, got)
	}
	if got, want := v.kubeAuth.mount, mount; got != want {
		t.Errorf("Want mount %v, got %v", want, got)
	}
}
