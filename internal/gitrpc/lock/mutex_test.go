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
