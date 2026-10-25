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

package insights

import (
	"context"
	"errors"

	"github.com/harness/gitness/types"
)

// IssueObserver is notified when ingestion creates a new issue or flips a
// resolved issue to regressed. It is the extension point for later phases
// (creating tasks, calling the AI). Implementations must be fast or hand
// the work off; errors are logged by the service and never fail ingestion.
// The issue is a snapshot as stored; implementations may update it
// (e.g. set TaskID) through the issue store.
type IssueObserver interface {
	// OnIssueCreated is called after the first event of a new issue was stored.
	OnIssueCreated(ctx context.Context, project *types.InsightsProject, issue *types.InsightsIssue,
		event *types.InsightsEvent) error
	// OnIssueRegressed is called after a resolved issue received an event of another release.
	OnIssueRegressed(ctx context.Context, project *types.InsightsProject, issue *types.InsightsIssue,
		event *types.InsightsEvent) error
}

// NoopObserver is the default IssueObserver and does nothing.
type NoopObserver struct{}

func (NoopObserver) OnIssueCreated(context.Context, *types.InsightsProject, *types.InsightsIssue,
	*types.InsightsEvent) error {
	return nil
}

func (NoopObserver) OnIssueRegressed(context.Context, *types.InsightsProject, *types.InsightsIssue,
	*types.InsightsEvent) error {
	return nil
}

// MultiObserver notifies several observers in order. Every observer is called even if an
// earlier one failed; the errors are joined.
type MultiObserver []IssueObserver

func (m MultiObserver) OnIssueCreated(ctx context.Context, project *types.InsightsProject,
	issue *types.InsightsIssue, event *types.InsightsEvent) error {
	var errs []error
	for _, o := range m {
		errs = append(errs, o.OnIssueCreated(ctx, project, issue, event))
	}
	return errors.Join(errs...)
}

func (m MultiObserver) OnIssueRegressed(ctx context.Context, project *types.InsightsProject,
	issue *types.InsightsIssue, event *types.InsightsEvent) error {
	var errs []error
	for _, o := range m {
		errs = append(errs, o.OnIssueRegressed(ctx, project, issue, event))
	}
	return errors.Join(errs...)
}

// AddObserver registers an additional observer next to the current one, so several
// features (triage, verification) can listen without replacing each other.
// SetObserver still replaces everything.
func (s *Service) AddObserver(o IssueObserver) {
	if o == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch cur := s.observer.(type) {
	case NoopObserver, nil:
		s.observer = o
	case MultiObserver:
		s.observer = append(append(MultiObserver{}, cur...), o)
	default:
		s.observer = MultiObserver{cur, o}
	}
}
