// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

//go:build !oss
// +build !oss

package dag

import (
	"reflect"
	"sort"
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
		t.Errorf("Expect vertexes with no dependencies has zero ancestors")
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

func TestDependencies(t *testing.T) {
	dag := New()
	dag.Add("backend")
	dag.Add("frontend")
	dag.Add("publish", "backend", "frontend")

	if deps := dag.Dependencies("backend"); len(deps) != 0 {
		t.Errorf("Expect zero dependencies")
	}
	if deps := dag.Dependencies("frontend"); len(deps) != 0 {
		t.Errorf("Expect zero dependencies")
	}

	got, want := dag.Dependencies("publish"), []string{"backend", "frontend"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unexpected dependencies, got %v", got)
	}
}

func TestDependencies_Skipped(t *testing.T) {
	dag := New()
	dag.Add("backend")
	dag.Add("frontend").Skip = true
	dag.Add("publish", "backend", "frontend")

	if deps := dag.Dependencies("backend"); len(deps) != 0 {
		t.Errorf("Expect zero dependencies")
	}
	if deps := dag.Dependencies("frontend"); len(deps) != 0 {
		t.Errorf("Expect zero dependencies")
	}

	got, want := dag.Dependencies("publish"), []string{"backend"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unexpected dependencies, got %v", got)
	}
}

func TestDependencies_Complex(t *testing.T) {
	dag := New()
	dag.Add("clone")
	dag.Add("backend")
	dag.Add("frontend", "backend").Skip = true
	dag.Add("publish", "frontend", "clone")
	dag.Add("notify", "publish")

	if deps := dag.Dependencies("clone"); len(deps) != 0 {
		t.Errorf("Expect zero dependencies for clone")
	}
	if deps := dag.Dependencies("backend"); len(deps) != 0 {
		t.Errorf("Expect zero dependencies for backend")
	}

	got, want := dag.Dependencies("frontend"), []string{"backend"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unexpected dependencies for frontend, got %v", got)
	}

	got, want = dag.Dependencies("publish"), []string{"backend", "clone"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unexpected dependencies for publish, got %v", got)
	}

	got, want = dag.Dependencies("notify"), []string{"publish"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unexpected dependencies for notify, got %v", got)
	}
}

func TestDescendants(t *testing.T) {
	dag := New()
	dag.Add("backend")
	dag.Add("frontend", "backend")
	dag.Add("notify", "frontend")
	if dag.DetectCycles() {
		t.Errorf("cycles detected")
	}

	got := dag.Descendants("backend")
	sort.Strings(got)
	want := []string{"frontend", "notify"}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Descendants(backend) = %v, want %v", got, want)
	}

	got = dag.Descendants("frontend")
	sort.Strings(got)
	want = []string{"notify"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Descendants(frontend) = %v, want %v", got, want)
	}

	if got := dag.Descendants("notify"); len(got) != 0 {
		t.Errorf("Expect zero descendants for leaf vertex, got %v", got)
	}
}

func TestDescendants_Parallel(t *testing.T) {
	dag := New()
	dag.Add("backend")
	dag.Add("frontend")
	dag.Add("publish", "backend", "frontend")
	dag.Add("notify", "publish")
	if dag.DetectCycles() {
		t.Errorf("cycles detected")
	}

	got := dag.Descendants("backend")
	sort.Strings(got)
	want := []string{"notify", "publish"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Descendants(backend) = %v, want %v", got, want)
	}

	got = dag.Descendants("frontend")
	sort.Strings(got)
	want = []string{"notify", "publish"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Descendants(frontend) = %v, want %v", got, want)
	}
}

func TestDescendants_NotFound(t *testing.T) {
	dag := New()
	dag.Add("backend")
	dag.Add("frontend", "backend")
	if dag.DetectCycles() {
		t.Errorf("cycles detected")
	}
	if got := dag.Descendants("does-not-exist"); len(got) != 0 {
		t.Errorf("Expect vertex not found returns no descendants, got %v", got)
	}
}

func TestDescendants_Complex(t *testing.T) {
	dag := New()
	dag.Add("clone")
	dag.Add("backend")
	dag.Add("frontend", "backend")
	dag.Add("publish", "frontend", "clone")
	dag.Add("notify", "publish")
	if dag.DetectCycles() {
		t.Errorf("cycles detected")
	}

	got := dag.Descendants("clone")
	sort.Strings(got)
	want := []string{"notify", "publish"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Descendants(clone) = %v, want %v", got, want)
	}

	got = dag.Descendants("backend")
	sort.Strings(got)
	want = []string{"frontend", "notify", "publish"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Descendants(backend) = %v, want %v", got, want)
	}

	if got := dag.Descendants("notify"); len(got) != 0 {
		t.Errorf("Expect zero descendants for notify, got %v", got)
	}
}

// TestDescendants_EmptyDAG ensures Descendants on an empty or unknown vertex does not panic.
func TestDescendants_EmptyDAG(t *testing.T) {
	dag := New()
	if got := dag.Descendants("any"); len(got) != 0 {
		t.Errorf("Descendants on empty DAG should return empty, got %v", got)
	}
}

// TestDescendants_SingleVertex ensures a single vertex with no edges has no descendants.
func TestDescendants_SingleVertex(t *testing.T) {
	dag := New()
	dag.Add("alone")
	if got := dag.Descendants("alone"); len(got) != 0 {
		t.Errorf("Single vertex with no dependents should have no descendants, got %v", got)
	}
}

// TestDescendants_Diamond ensures a diamond DAG (multi-path to same node) returns
// each descendant exactly once: A -> B -> D and A -> C -> D.
func TestDescendants_Diamond(t *testing.T) {
	dag := New()
	dag.Add("A")
	dag.Add("B", "A")
	dag.Add("C", "A")
	dag.Add("D", "B", "C")
	if dag.DetectCycles() {
		t.Fatal("diamond should have no cycles")
	}

	got := dag.Descendants("A")
	sort.Strings(got)
	want := []string{"B", "C", "D"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Descendants(A) = %v, want %v", got, want)
	}

	got = dag.Descendants("B")
	sort.Strings(got)
	want = []string{"D"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Descendants(B) = %v, want %v", got, want)
	}

	got = dag.Descendants("C")
	sort.Strings(got)
	want = []string{"D"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Descendants(C) = %v, want %v", got, want)
	}

	if got := dag.Descendants("D"); len(got) != 0 {
		t.Errorf("Descendants(D) should be empty, got %v", got)
	}
}

// TestDescendants_Disconnected ensures two disconnected subgraphs do not mix:
// subgraph1: a -> b; subgraph2: x -> y. Descendants(a) must not include x or y.
func TestDescendants_Disconnected(t *testing.T) {
	dag := New()
	dag.Add("a")
	dag.Add("b", "a")
	dag.Add("x")
	dag.Add("y", "x")
	if dag.DetectCycles() {
		t.Fatal("disconnected DAG should have no cycles")
	}

	got := dag.Descendants("a")
	sort.Strings(got)
	want := []string{"b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Descendants(a) = %v, want %v", got, want)
	}

	got = dag.Descendants("x")
	sort.Strings(got)
	want = []string{"y"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Descendants(x) = %v, want %v", got, want)
	}

	if got := dag.Descendants("b"); len(got) != 0 {
		t.Errorf("Descendants(b) should be empty, got %v", got)
	}
	if got := dag.Descendants("y"); len(got) != 0 {
		t.Errorf("Descendants(y) should be empty, got %v", got)
	}
}

// TestDescendants_LongChain ensures a linear chain A -> B -> C -> D -> E
// yields correct transitive descendants at each level.
func TestDescendants_LongChain(t *testing.T) {
	dag := New()
	dag.Add("A")
	dag.Add("B", "A")
	dag.Add("C", "B")
	dag.Add("D", "C")
	dag.Add("E", "D")
	if dag.DetectCycles() {
		t.Fatal("chain should have no cycles")
	}

	got := dag.Descendants("A")
	sort.Strings(got)
	want := []string{"B", "C", "D", "E"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Descendants(A) = %v, want %v", got, want)
	}

	got = dag.Descendants("C")
	sort.Strings(got)
	want = []string{"D", "E"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Descendants(C) = %v, want %v", got, want)
	}

	got = dag.Descendants("E")
	if len(got) != 0 {
		t.Errorf("Descendants(E) should be empty, got %v", got)
	}
}

// TestDescendants_Star ensures one root with multiple direct dependents:
// B, C, D all depend on A. Descendants(A) = B, C, D.
func TestDescendants_Star(t *testing.T) {
	dag := New()
	dag.Add("A")
	dag.Add("B", "A")
	dag.Add("C", "A")
	dag.Add("D", "A")
	if dag.DetectCycles() {
		t.Fatal("star should have no cycles")
	}

	got := dag.Descendants("A")
	sort.Strings(got)
	want := []string{"B", "C", "D"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Descendants(A) = %v, want %v", got, want)
	}

	for _, leaf := range []string{"B", "C", "D"} {
		if got := dag.Descendants(leaf); len(got) != 0 {
			t.Errorf("Descendants(%s) should be empty, got %v", leaf, got)
		}
	}
}

// TestDescendants_MissingVertexReferencedAsDep: when a vertex is not in the graph
// but is listed as a dependency by another vertex, Descendants(missing) returns
// all vertices that (transitively) list it as a dependency (buildReverseDep still
// records them). This documents current behavior for malformed pipelines.
func TestDescendants_MissingVertexReferencedAsDep(t *testing.T) {
	dag := New()
	dag.Add("child", "missing")
	dag.Add("grandchild", "child")
	// "missing" is never added via Add(), but "child" depends on it.

	got := dag.Descendants("missing")
	sort.Strings(got)
	want := []string{"child", "grandchild"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Descendants(missing) = %v, want %v (vertices that reference missing)", got, want)
	}
}

// TestDescendants_DoesNotIncludeSelf ensures the queried vertex is never in the result.
func TestDescendants_DoesNotIncludeSelf(t *testing.T) {
	dag := New()
	dag.Add("A")
	dag.Add("B", "A")
	dag.Add("C", "B")
	got := dag.Descendants("A")
	for _, n := range got {
		if n == "A" {
			t.Error("Descendants(A) must not include A itself")
		}
	}
	got = dag.Descendants("B")
	for _, n := range got {
		if n == "B" {
			t.Error("Descendants(B) must not include B itself")
		}
	}
}
