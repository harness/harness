// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package lock

import (
	"context"
	"testing"
)

func TestClient_AcquireLock(t *testing.T) {
	c := Mutex{}
	lock, err := c.AcquireLock(context.Background(), "simple")
	if err != nil {
		t.Error(err)
	}
	lock1, err := c.AcquireLock(context.Background(), "simple")
	if err != nil {
		t.Error(err)
	}
	lock.Release()
	lock1.Release()
}
