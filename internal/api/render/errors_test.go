// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package render

import "testing"

func TestError(t *testing.T) {
	got, want := ErrNotFound.Message, ErrNotFound.Message
	if got != want {
		t.Errorf("Want error string %q, got %q", got, want)
	}
}
