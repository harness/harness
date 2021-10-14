// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package core

import (
	"testing"

	"github.com/drone/drone-go/drone"
)

func TestStepIsDone(t *testing.T) {
	for _, status := range statusDone {
		v := drone.Step{Status: status}
		if StepIsDone(&v) == false {
			t.Errorf("Expect status %s is done", status)
		}
	}

	for _, status := range statusNotDone {
		v := drone.Step{Status: status}
		if StepIsDone(&v) == true {
			t.Errorf("Expect status %s is not done", status)
		}
	}
}
