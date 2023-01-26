// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package lock

import (
	"context"
	"fmt"
)

// KindError enum displays human readable message
// in error.
type KindError string

const (
	LockHeld            KindError = "lock already held"
	LockNotHeld         KindError = "lock not held"
	ProviderError       KindError = "lock provider error"
	CannotLock          KindError = "timeout while trying to acquire lock"
	Context             KindError = "context error while trying to acquire lock"
	MaxRetriesExceeded  KindError = "max retries exceeded to acquire lock"
	GenerateTokenFailed KindError = "token generation failed"
)

// Error is custom unique type for all type of errors.
type Error struct {
	Kind KindError
	Key  string
	Err  error
}

func NewError(kind KindError, key string, err error) *Error {
	return &Error{
		Kind: kind,
		Key:  key,
		Err:  err,
	}
}

// Error implements error interface.
func (e Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s on key %s with err: %v", e.Kind, e.Key, e.Err)
	}
	return fmt.Sprintf("%s on key %s", e.Kind, e.Key)
}

// MutexManager describes a Distributed Lock Manager.
type MutexManager interface {
	// NewMutex creates a mutex for the given key. The returned mutex is not held
	// and must be acquired with a call to .Lock.
	NewMutex(key string, options ...Option) (Mutex, error)
}

type Mutex interface {
	// Key returns the key to be locked.
	Key() string

	// Lock acquires the lock. It fails with error if the lock is already held.
	Lock(ctx context.Context) error

	// Unlock releases the lock. It fails with error if the lock is not currently held.
	Unlock(ctx context.Context) error
}
