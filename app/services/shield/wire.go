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

package shield

import (
	"context"
	"fmt"
	"github.com/harness/gitness/app/services/aidecide"
	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/app/services/settings"
	"path/filepath"
	"time"

	gitevents "github.com/harness/gitness/app/events/git"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideSyncer,
	ProvideService,
	ProvideWiredTriager,
)

// dataDir returns the configured data directory or its default next to the repositories root.
func dataDir(config *types.Config) string {
	if config.Shield.DataDir != "" {
		return config.Shield.DataDir
	}
	if config.Git.Root != "" {
		return filepath.Join(filepath.Dir(config.Git.Root), "shield")
	}
	return "shield"
}

// ProvideSyncer provides the vulnerability data syncer.
func ProvideSyncer(config *types.Config) *Syncer {
	return NewSyncer(SyncConfig{
		DataDir:    dataDir(config),
		Ecosystems: config.Shield.Ecosystems,
		Interval:   config.Shield.SyncInterval,
		OSVBaseURL: config.Shield.OSVBaseURL,
		EPSSURL:    config.Shield.EPSSURL,
		KEVURL:     config.Shield.KEVURL,
	}, nil)
}

// ProvideService provides the Shield scan service, registers its jobs and, if enabled, the push
// listener and the recurring vulnerability data sync.
func ProvideService(
	ctx context.Context,
	config *types.Config,
	scans store.ShieldScanStore,
	findings store.ShieldFindingStore,
	taskSvc *task.Service,
	gitSvc git.Interface,
	syncer *Syncer,
	publicAccess publicaccess.Service,
	repoFinder refcache.RepoFinder,
	scheduler *job.Scheduler,
	executor *job.Executor,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
) (*Service, error) {
	svc := NewService(
		Config{Enabled: config.Shield.Enabled, PatchThreshold: config.Shield.PatchThreshold},
		scans, findings, taskSvc, gitSvc, syncer, publicAccess, repoFinder, scheduler,
	)

	interval := config.Shield.SyncInterval
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	if err := Register(ctx, svc.Enabled(), interval, scheduler, executor, syncer, svc); err != nil {
		return nil, err
	}
	if !svc.Enabled() {
		return svc, nil
	}
	if err := RegisterEventListeners(ctx, config.InstanceID, gitReaderFactory, svc); err != nil {
		return nil, fmt.Errorf("failed to register shield event listeners: %w", err)
	}
	return svc, nil
}

// triageFindingStore joins the finding store with the triage outcome store.
type triageFindingStore struct {
	store.ShieldFindingStore
	store.ShieldFindingTriageStore
}

// ProvideWiredTriager provides the AI triage of Shield findings and hooks it into the scan service.
func ProvideWiredTriager(
	config *types.Config,
	svc *Service,
	findings store.ShieldFindingStore,
	triageStore store.ShieldFindingTriageStore,
	taskSvc *task.Service,
	decide *aidecide.Service,
	modes *aigateway.ModeResolver,
	settingsSvc *settings.Service,
) *Triager {
	return ProvideTriager(config, svc, triageFindingStore{findings, triageStore}, taskSvc, decide, modes, settingsSvc)
}
