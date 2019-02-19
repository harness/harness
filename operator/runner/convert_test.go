// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package runner

import (
	"testing"

	"github.com/drone/drone-runtime/engine"
	"github.com/drone/drone-runtime/runtime"
	"github.com/drone/drone/core"
	"github.com/google/go-cmp/cmp"
)

// func Test_convertSecrets(t *testing.T) {
// 	secrets := []*core.Secret{
// 		{Name: "docker_username", Data: "octocat"},
// 		{Name: "docker_password", Data: "password"},
// 	}
// 	got := convertSecrets(secrets)

// 	want := []compiler.Secret{
// 		{Name: "docker_username", Value: "octocat"},
// 		{Name: "docker_password", Value: "password"},
// 	}

// 	if diff := cmp.Diff(got, want); len(diff) != 0 {
// 		t.Errorf(diff)
// 	}
// }

func Test_convertRegistry(t *testing.T) {
	list := []*core.Registry{
		{
			Address:  "docker.io",
			Username: "octocat",
			Password: "password",
		},
	}
	got := convertRegistry(list)
	want := []*engine.DockerAuth{
		{
			Address:  "docker.io",
			Username: "octocat",
			Password: "password",
		},
	}
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}

func Test_convertLines(t *testing.T) {
	lines := []*runtime.Line{
		{
			Number:    1,
			Message:   "ping google.com",
			Timestamp: 1257894000,
		},
		{
			Number:    1,
			Message:   "PING google.com (1.2.3.4): 56 data bytes",
			Timestamp: 1257894000,
		},
	}
	got := convertLines(lines)
	want := []*core.Line{
		{
			Number:    1,
			Message:   "ping google.com",
			Timestamp: 1257894000,
		},
		{
			Number:    1,
			Message:   "PING google.com (1.2.3.4): 56 data bytes",
			Timestamp: 1257894000,
		},
	}
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}

func Test_convertLine(t *testing.T) {
	line := &runtime.Line{
		Number:    1,
		Message:   "ping google.com",
		Timestamp: 1257894000,
	}
	got := convertLine(line)
	want := &core.Line{
		Number:    1,
		Message:   "ping google.com",
		Timestamp: 1257894000,
	}
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}
