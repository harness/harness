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

package store

import (
	"context"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type (
	// ShieldScanStore defines the Shield scan data storage.
	ShieldScanStore interface {
		// Create inserts the scan and sets its ID.
		Create(ctx context.Context, scan *types.ShieldScan) error
		// Update updates status, error, counters and finished of the scan.
		Update(ctx context.Context, scan *types.ShieldScan) error
		// Find finds the scan by id.
		Find(ctx context.Context, id int64) (*types.ShieldScan, error)
		// ListByRepo lists the scans of the repository, newest first.
		ListByRepo(ctx context.Context, repoID int64, page, size int) ([]*types.ShieldScan, error)
	}

	// ShieldFindingStore defines the Shield finding data storage.
	ShieldFindingStore interface {
		// Upsert inserts the finding or, if (repo_id, fingerprint) exists, refreshes its scanner
		// derived fields and last_seen. A fixed finding that reappears is reopened; ignored and
		// accepted_risk findings keep their state. ID, State, FirstSeen* and Version of the
		// argument are set from the stored row.
		Upsert(ctx context.Context, finding *types.ShieldFinding) error
		// Find finds the finding by id.
		Find(ctx context.Context, id int64) (*types.ShieldFinding, error)
		// List lists the findings of the repository ordered by risk score descending.
		List(ctx context.Context, repoID int64, filter *types.ShieldFindingFilter) ([]*types.ShieldFinding, error)
		// Count counts the findings of the repository matching the filter (pagination ignored).
		Count(ctx context.Context, repoID int64, filter *types.ShieldFindingFilter) (int64, error)
		// MarkFixedNotSeenIn marks open findings of the repository whose last seen scan is not
		// scanID as fixed at now. It returns the number of affected findings.
		MarkFixedNotSeenIn(ctx context.Context, repoID, scanID, now int64) (int64, error)
		// SetTask links the finding to a task (nil unlinks).
		SetTask(ctx context.Context, id int64, taskID *int64, now int64) error
		// UpdateState sets state and reason of the finding.
		UpdateState(ctx context.Context, id int64, state enum.ShieldFindingState, reason string, now int64) error
	}
)
