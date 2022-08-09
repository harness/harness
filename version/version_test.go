// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package version

import "testing"

func TestVersion(t *testing.T) {
	if got, want := Version.String(), "1.0.0"; got != want {
		t.Errorf("Want version %s, got %s", want, got)
	}
}
