// Copyright 2019 Drone IO, Inc.
// Copyright 2018 natessilva
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dag

// Dag is a directed acyclic graph.
type Dag struct {
	graph map[string]*Vertex
}

// Vertex is a vertex in the graph.
type Vertex struct {
	Name  string
	Skip  bool
	graph []string
}

// New creates a new directed acyclic graph (dag) that can
// determinate if a stage has dependencies.
func New() *Dag {
	return &Dag{
		graph: make(map[string]*Vertex),
	}
}

// Add establishes a dependency between two vertices in the graph.
func (d *Dag) Add(from string, to ...string) *Vertex {
	vertex := new(Vertex)
	vertex.Name = from
	vertex.Skip = false
	vertex.graph = to
	d.graph[from] = vertex
	return vertex
}

// Get returns the vertex from the graph.
func (d *Dag) Get(name string) (*Vertex, bool) {
	vertex, ok := d.graph[name]
	return vertex, ok
}

// Dependencies returns the direct dependencies accounting for
// skipped dependencies.
func (d *Dag) Dependencies(name string) []string {
	vertex := d.graph[name]
	return d.dependencies(vertex)
}

// Ancestors returns the ancestors of the vertex.
func (d *Dag) Ancestors(name string) []*Vertex {
	vertex := d.graph[name]
	return d.ancestors(vertex)
}

// Descendants returns the names of all vertices that transitively depend on
// the given vertex (downstream in the DAG). Used e.g. to reset downstream
// stages when restarting a single stage.
func (d *Dag) Descendants(name string) []string {
	reverseDep := d.buildReverseDep()
	set := d.descendantsWithIndex(name, reverseDep, nil)
	out := make([]string, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	return out
}

// buildReverseDep returns a map: for each vertex name, the set of vertex
// names that list it in their dependencies (direct dependents). Built once in
// O(V*D) also know as O(V+E), then descendant lookups are O(1) per vertex.
func (d *Dag) buildReverseDep() map[string]map[string]struct{} {
	reverseDep := make(map[string]map[string]struct{})
	for vName, vertex := range d.graph {
		if vertex == nil {
			continue
		}
		for _, dep := range vertex.graph {
			if reverseDep[dep] == nil {
				reverseDep[dep] = make(map[string]struct{})
			}
			reverseDep[dep][vName] = struct{}{}
		}
	}
	return reverseDep
}

func (d *Dag) descendantsWithIndex(name string, reverseDep map[string]map[string]struct{}, seen map[string]struct{}) map[string]struct{} {
	if seen == nil {
		seen = make(map[string]struct{})
	}
	if _, ok := seen[name]; ok {
		return nil
	}
	seen[name] = struct{}{}
	out := make(map[string]struct{})
	for vName := range reverseDep[name] {
		out[vName] = struct{}{}
		for k := range d.descendantsWithIndex(vName, reverseDep, seen) {
			out[k] = struct{}{}
		}
	}
	return out
}

// DetectCycles returns true if cycles are detected in the graph.
func (d *Dag) DetectCycles() bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for vertex := range d.graph {
		if !visited[vertex] {
			if d.detectCycles(vertex, visited, recStack) {
				return true
			}
		}
	}
	return false
}

// helper function returns the list of ancestors for the vertex.
func (d *Dag) ancestors(parent *Vertex) []*Vertex {
	if parent == nil {
		return nil
	}
	var combined []*Vertex
	for _, name := range parent.graph {
		vertex, found := d.graph[name]
		if !found {
			continue
		}
		if !vertex.Skip {
			combined = append(combined, vertex)
		}
		combined = append(combined, d.ancestors(vertex)...)
	}
	return combined
}

// helper function returns the list of dependencies for the,
// vertex taking into account skipped dependencies.
func (d *Dag) dependencies(parent *Vertex) []string {
	if parent == nil {
		return nil
	}
	var combined []string
	for _, name := range parent.graph {
		vertex, found := d.graph[name]
		if !found {
			continue
		}
		if vertex.Skip {
			// if the vertex is skipped we should move up the
			// graph and check direct ancestors.
			combined = append(combined, d.dependencies(vertex)...)
		} else {
			combined = append(combined, vertex.Name)
		}
	}
	return combined
}

// helper function returns true if the vertex is cyclical.
func (d *Dag) detectCycles(name string, visited, recStack map[string]bool) bool {
	visited[name] = true
	recStack[name] = true

	vertex, ok := d.graph[name]
	if !ok {
		return false
	}
	for _, v := range vertex.graph {
		// only check cycles on a vertex one time
		if !visited[v] {
			if d.detectCycles(v, visited, recStack) {
				return true
			}
			// if we've visited this vertex in this recursion
			// stack, then we have a cycle
		} else if recStack[v] {
			return true
		}

	}
	recStack[name] = false
	return false
}
