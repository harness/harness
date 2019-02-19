// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package admission

import (
	"context"
	"errors"

	"github.com/drone/drone/core"
)

// ErrClosed is returned when attempting to create a new
// user account and admissions are closed.
var ErrClosed = errors.New("User registration is disabled")

// Open enfoces an open admission policy by default unless
// disabled.
func Open(disabled bool) core.AdmissionService {
	return &closed{disabled: disabled}
}

type closed struct {
	disabled bool
}

func (s *closed) Admit(ctx context.Context, user *core.User) error {
	if s.disabled {
		return ErrClosed
	}
	return nil
}
