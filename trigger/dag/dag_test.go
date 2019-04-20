// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package dag

import (
	"testing"
)

func TestDag(t *testing.T) {
	dag := New()
	dag.Add("backend")
	dag.Add("frontend")
	dag.Add("notify", "backend", "frontend")
	if dag.DetectCycles() {
		t.Errorf("cycles detected")
	}

	dag = New()
	dag.Add("notify", "backend", "frontend")
	if dag.DetectCycles() {
		t.Errorf("cycles detected")
	}

	dag = New()
	dag.Add("backend", "frontend")
	dag.Add("frontend", "backend")
	dag.Add("notify", "backend", "frontend")
	if dag.DetectCycles() == false {
		t.Errorf("Expect cycles detected")
	}

	dag = New()
	dag.Add("backend", "backend")
	dag.Add("frontend", "backend")
	dag.Add("notify", "backend", "frontend")
	if dag.DetectCycles() == false {
		t.Errorf("Expect cycles detected")
	}

	dag = New()
	dag.Add("backend")
	dag.Add("frontend")
	dag.Add("notify", "backend", "frontend", "notify")
	if dag.DetectCycles() == false {
		t.Errorf("Expect cycles detected")
	}
}

func TestAncestors(t *testing.T) {
	dag := New()
	v := dag.Add("backend")
	dag.Add("frontend", "backend")
	dag.Add("notify", "frontend")

	ancestors := dag.Ancestors("frontend")
	if got, want := len(ancestors), 1; got != want {
		t.Errorf("Want %d ancestors, got %d", want, got)
	}
	if ancestors[0] != v {
		t.Errorf("Unexpected ancestor")
	}

	if v := dag.Ancestors("backend"); len(v) != 0 {
		t.Errorf("Expect vertexes with no dependences has zero ancestors")
	}
}

func TestAncestors_Skipped(t *testing.T) {
	dag := New()
	dag.Add("backend").Skip = true
	dag.Add("frontend", "backend").Skip = true
	dag.Add("notify", "frontend")

	if v := dag.Ancestors("frontend"); len(v) != 0 {
		t.Errorf("Expect skipped vertexes excluded")
	}
	if v := dag.Ancestors("notify"); len(v) != 0 {
		t.Errorf("Expect skipped vertexes excluded")
	}
}

func TestAncestors_NotFound(t *testing.T) {
	dag := New()
	dag.Add("backend")
	dag.Add("frontend", "backend")
	dag.Add("notify", "frontend")
	if dag.DetectCycles() {
		t.Errorf("cycles detected")
	}
	if v := dag.Ancestors("does-not-exist"); len(v) != 0 {
		t.Errorf("Expect vertex not found does not panic")
	}
}

func TestAncestors_Malformed(t *testing.T) {
	dag := New()
	dag.Add("backend")
	dag.Add("frontend", "does-not-exist")
	dag.Add("notify", "frontend")
	if dag.DetectCycles() {
		t.Errorf("cycles detected")
	}
	if v := dag.Ancestors("frontend"); len(v) != 0 {
		t.Errorf("Expect invalid dependency does not panic")
	}
}

func TestAncestors_Complex(t *testing.T) {
	dag := New()
	dag.Add("backend")
	dag.Add("frontend")
	dag.Add("publish", "backend", "frontend")
	dag.Add("deploy", "publish")
	last := dag.Add("notify", "deploy")
	if dag.DetectCycles() {
		t.Errorf("cycles detected")
	}

	ancestors := dag.Ancestors("notify")
	if got, want := len(ancestors), 4; got != want {
		t.Errorf("Want %d ancestors, got %d", want, got)
		return
	}
	for _, ancestor := range ancestors {
		if ancestor == last {
			t.Errorf("Unexpected ancestor")
		}
	}

	v, _ := dag.Get("publish")
	v.Skip = true
	ancestors = dag.Ancestors("notify")
	if got, want := len(ancestors), 3; got != want {
		t.Errorf("Want %d ancestors, got %d", want, got)
		return
	}
}
