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

package agentguard

import (
	"context"
	"sync"
	"time"
)

// AttemptStatus is the state of an agent attempt on a task.
type AttemptStatus string

const (
	AttemptRunning   AttemptStatus = "running"
	AttemptFailed    AttemptStatus = "failed"
	AttemptSucceeded AttemptStatus = "succeeded"
)

// Attempt is a single run of an agent for a task.
type Attempt struct {
	ID        int64
	TaskKey   string
	RepoID    int64
	StartedAt time.Time
	Status    AttemptStatus
}

// AttemptStore keeps the attempt bookkeeping of the guard. It is an interface so that
// it can later be backed by the ai_tasks table.
type AttemptStore interface {
	// Start records a new running attempt.
	Start(ctx context.Context, taskKey string, repoID int64, at time.Time) (Attempt, error)
	// Finish sets the final status of an attempt.
	Finish(ctx context.Context, attemptID int64, status AttemptStatus) error
	// ListByTask returns all attempts of a task.
	ListByTask(ctx context.Context, taskKey string) ([]Attempt, error)
	// CountActive returns the number of running attempts for a repo.
	CountActive(ctx context.Context, repoID int64) (int, error)
}

// MemoryAttemptStore is an in-memory AttemptStore.
type MemoryAttemptStore struct {
	mu       sync.Mutex
	nextID   int64
	attempts []Attempt
}

func NewMemoryAttemptStore() *MemoryAttemptStore {
	return &MemoryAttemptStore{}
}

func (s *MemoryAttemptStore) Start(
	_ context.Context,
	taskKey string,
	repoID int64,
	at time.Time,
) (Attempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	a := Attempt{ID: s.nextID, TaskKey: taskKey, RepoID: repoID, StartedAt: at, Status: AttemptRunning}
	s.attempts = append(s.attempts, a)
	return a, nil
}

func (s *MemoryAttemptStore) Finish(_ context.Context, attemptID int64, status AttemptStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.attempts {
		if s.attempts[i].ID == attemptID {
			s.attempts[i].Status = status
			return nil
		}
	}
	return nil
}

func (s *MemoryAttemptStore) ListByTask(_ context.Context, taskKey string) ([]Attempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []Attempt
	for _, a := range s.attempts {
		if a.TaskKey == taskKey {
			out = append(out, a)
		}
	}
	return out, nil
}

func (s *MemoryAttemptStore) CountActive(_ context.Context, repoID int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	n := 0
	for _, a := range s.attempts {
		if a.RepoID == repoID && a.Status == AttemptRunning {
			n++
		}
	}
	return n, nil
}
