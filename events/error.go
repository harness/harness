// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package events

import (
	"errors"
	"fmt"
)

var (
	errDiscardEvent = &discardEventError{}
)

// discardEventError is an error which, if returned by the event handler,
// causes the source event to be discarded despite any erros.
type discardEventError struct {
	inner error
}

func NewDiscardEventError(inner error) error {
	return &discardEventError{
		inner: inner,
	}
}

func NewDiscardEventErrorf(format string, args ...interface{}) error {
	return &discardEventError{
		inner: fmt.Errorf(format, args...),
	}
}

func (e *discardEventError) Error() string {
	return fmt.Sprintf("discarding requested due to: %s", e.inner)
}

func (e *discardEventError) Unwrap() error {
	return e.inner
}

func (e *discardEventError) Is(target error) bool {
	// NOTE: it's an internal event and we only ever check with the singleton instance
	return errors.Is(target, errDiscardEvent)
}
