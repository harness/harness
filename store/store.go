// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package store

import (
	"github.com/drone/drone/core"
)

// Stores provides all database stores.
type Stores struct {
	Batch   core.Batcher
	Builds  core.BuildStore
	Crons   core.CronStore
	Logs    core.LogStore
	Perms   core.PermStore
	Secrets core.SecretStore
	Stages  core.StageStore
	Steps   core.StepStore
	Repos   core.RepositoryStore
	Users   core.UserStore
}
