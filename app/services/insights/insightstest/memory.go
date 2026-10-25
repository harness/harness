// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package insightstest provides in-memory Insights stores for tests.
package insightstest

import (
	"context"
	"sort"
	"sync"

	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
)

// Store implements the three Insights stores in memory.
type Store struct {
	mu       sync.Mutex
	nextID   int64
	Projects map[int64]*types.InsightsProject
	Issues   map[int64]*types.InsightsIssue
	Events   map[int64]*types.InsightsEvent
}

// New returns an empty Store.
func New() *Store {
	return &Store{
		Projects: map[int64]*types.InsightsProject{},
		Issues:   map[int64]*types.InsightsIssue{},
		Events:   map[int64]*types.InsightsEvent{},
	}
}

func (s *Store) id() int64 { s.nextID++; return s.nextID }

// Projects

func (s *Store) projectsAPI() *projects { return &projects{s} }
func (s *Store) issuesAPI() *issues     { return &issues{s} }
func (s *Store) eventsAPI() *events     { return &events{s} }

// ProjectStore returns the store.InsightsProjectStore view.
func (s *Store) ProjectStore() *projects { return s.projectsAPI() }

// IssueStore returns the store.InsightsIssueStore view.
func (s *Store) IssueStore() *issues { return s.issuesAPI() }

// EventStore returns the store.InsightsEventStore view.
func (s *Store) EventStore() *events { return s.eventsAPI() }

type projects struct{ s *Store }

func (p *projects) Create(_ context.Context, in *types.InsightsProject) error {
	p.s.mu.Lock()
	defer p.s.mu.Unlock()
	for _, e := range p.s.Projects {
		if e.PublicKey == in.PublicKey || (e.RepoID == in.RepoID && e.Identifier == in.Identifier) {
			return gitness_store.ErrDuplicate
		}
	}
	in.ID = p.s.id()
	c := *in
	p.s.Projects[in.ID] = &c
	return nil
}

func (p *projects) Find(_ context.Context, id int64) (*types.InsightsProject, error) {
	p.s.mu.Lock()
	defer p.s.mu.Unlock()
	e, ok := p.s.Projects[id]
	if !ok {
		return nil, gitness_store.ErrResourceNotFound
	}
	c := *e
	return &c, nil
}

func (p *projects) FindByIdentifier(_ context.Context, repoID int64, identifier string) (*types.InsightsProject, error) {
	p.s.mu.Lock()
	defer p.s.mu.Unlock()
	for _, e := range p.s.Projects {
		if e.RepoID == repoID && e.Identifier == identifier {
			c := *e
			return &c, nil
		}
	}
	return nil, gitness_store.ErrResourceNotFound
}

func (p *projects) List(_ context.Context, repoID int64) ([]*types.InsightsProject, error) {
	p.s.mu.Lock()
	defer p.s.mu.Unlock()
	var out []*types.InsightsProject
	for _, e := range p.s.Projects {
		if e.RepoID == repoID {
			c := *e
			out = append(out, &c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Identifier < out[j].Identifier })
	return out, nil
}

func (p *projects) Update(_ context.Context, in *types.InsightsProject) error {
	p.s.mu.Lock()
	defer p.s.mu.Unlock()
	if _, ok := p.s.Projects[in.ID]; !ok {
		return gitness_store.ErrResourceNotFound
	}
	c := *in
	p.s.Projects[in.ID] = &c
	return nil
}

func (p *projects) Delete(_ context.Context, id int64) error {
	p.s.mu.Lock()
	defer p.s.mu.Unlock()
	delete(p.s.Projects, id)
	return nil
}

type issues struct{ s *Store }

func (i *issues) Create(_ context.Context, in *types.InsightsIssue) error {
	i.s.mu.Lock()
	defer i.s.mu.Unlock()
	for _, e := range i.s.Issues {
		if e.ProjectID == in.ProjectID && e.Fingerprint == in.Fingerprint {
			return gitness_store.ErrDuplicate
		}
	}
	in.ID = i.s.id()
	c := *in
	i.s.Issues[in.ID] = &c
	return nil
}

func (i *issues) Find(_ context.Context, id int64) (*types.InsightsIssue, error) {
	i.s.mu.Lock()
	defer i.s.mu.Unlock()
	e, ok := i.s.Issues[id]
	if !ok {
		return nil, gitness_store.ErrResourceNotFound
	}
	c := *e
	return &c, nil
}

func (i *issues) FindByFingerprint(_ context.Context, projectID int64, fp string) (*types.InsightsIssue, error) {
	i.s.mu.Lock()
	defer i.s.mu.Unlock()
	for _, e := range i.s.Issues {
		if e.ProjectID == projectID && e.Fingerprint == fp {
			c := *e
			return &c, nil
		}
	}
	return nil, gitness_store.ErrResourceNotFound
}

func (i *issues) Update(_ context.Context, in *types.InsightsIssue) error {
	i.s.mu.Lock()
	defer i.s.mu.Unlock()
	e, ok := i.s.Issues[in.ID]
	if !ok || e.Version != in.Version {
		return gitness_store.ErrVersionConflict
	}
	in.Version++
	c := *in
	i.s.Issues[in.ID] = &c
	return nil
}

func (i *issues) List(_ context.Context, repoID int64, f *types.InsightsIssueFilter) ([]*types.InsightsIssue, error) {
	i.s.mu.Lock()
	defer i.s.mu.Unlock()
	var out []*types.InsightsIssue
	for _, e := range i.s.Issues {
		if e.RepoID != repoID {
			continue
		}
		if f != nil && len(f.Statuses) > 0 {
			match := false
			for _, st := range f.Statuses {
				match = match || st == e.Status
			}
			if !match {
				continue
			}
		}
		c := *e
		out = append(out, &c)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].LastSeen > out[b].LastSeen })
	return out, nil
}

func (i *issues) Count(ctx context.Context, repoID int64, f *types.InsightsIssueFilter) (int64, error) {
	l, err := i.List(ctx, repoID, f)
	return int64(len(l)), err
}

type events struct{ s *Store }

func (e *events) Create(_ context.Context, in *types.InsightsEvent) error {
	e.s.mu.Lock()
	defer e.s.mu.Unlock()
	for _, x := range e.s.Events {
		if x.ProjectID == in.ProjectID && x.EventID == in.EventID {
			return gitness_store.ErrDuplicate
		}
	}
	in.ID = e.s.id()
	c := *in
	e.s.Events[in.ID] = &c
	return nil
}

func (e *events) Exists(_ context.Context, projectID int64, eventID string) (bool, error) {
	e.s.mu.Lock()
	defer e.s.mu.Unlock()
	for _, x := range e.s.Events {
		if x.ProjectID == projectID && x.EventID == eventID {
			return true, nil
		}
	}
	return false, nil
}

func (e *events) ListByIssue(_ context.Context, issueID int64, limit int) ([]*types.InsightsEvent, error) {
	e.s.mu.Lock()
	defer e.s.mu.Unlock()
	var out []*types.InsightsEvent
	for _, x := range e.s.Events {
		if x.IssueID == issueID {
			c := *x
			out = append(out, &c)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Timestamp > out[b].Timestamp })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
