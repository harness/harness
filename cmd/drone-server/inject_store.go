// Copyright 2019 Drone IO, Inc.
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

package main

import (
	"github.com/drone/drone/cmd/drone-server/config"
	"github.com/drone/drone/core"
	"github.com/drone/drone/metric"
	"github.com/drone/drone/store/batch"
	"github.com/drone/drone/store/batch2"
	"github.com/drone/drone/store/build"
	"github.com/drone/drone/store/card"
	"github.com/drone/drone/store/cron"
	"github.com/drone/drone/store/logs"
	"github.com/drone/drone/store/perm"
	"github.com/drone/drone/store/repos"
	"github.com/drone/drone/store/secret"
	"github.com/drone/drone/store/secret/global"
	"github.com/drone/drone/store/shared/db"
	"github.com/drone/drone/store/shared/encrypt"
	"github.com/drone/drone/store/stage"
	"github.com/drone/drone/store/step"
	"github.com/drone/drone/store/template"
	"github.com/drone/drone/store/user"

	"github.com/google/wire"
	"github.com/sirupsen/logrus"
)

// wire set for loading the stores.
var storeSet = wire.NewSet(
	provideDatabase,
	provideEncrypter,
	provideBuildStore,
	provideLogStore,
	provideRepoStore,
	provideStageStore,
	provideUserStore,
	provideBatchStore,
	// batch.New,
	cron.New,
	card.New,
	perm.New,
	secret.New,
	global.New,
	step.New,
	template.New,
)

// provideDatabase is a Wire provider function that provides a
// database connection, configured from the environment.
func provideDatabase(config config.Config) (*db.DB, error) {
	return db.Connect(
		config.Database.Driver,
		config.Database.Datasource,
		config.Database.MaxConnections,
	)
}

// provideEncrypter is a Wire provider function that provides a
// database encrypter, configured from the environment.
func provideEncrypter(config config.Config) (encrypt.Encrypter, error) {
	enc, err := encrypt.New(config.Database.Secret)
	// mixed-content mode should be set to true if the database
	// originally had encryption disabled and therefore has
	// plaintext entries. This prevents Drone from returning an
	// error if decryption fails; on failure, the ciphertext is
	// returned as-is and the error is ignored.
	if aesgcm, ok := enc.(*encrypt.Aesgcm); ok {
		logrus.Debugln("main: database encryption enabled")
		if config.Database.EncryptMixedContent {
			logrus.Debugln("main: database encryption mixed-mode enabled")
			aesgcm.Compat = true
		}
	}
	return enc, err
}

// provideBuildStore is a Wire provider function that provides a
// build datastore, configured from the environment, with metrics
// enabled.
func provideBuildStore(db *db.DB) core.BuildStore {
	builds := build.New(db)
	metric.BuildCount(builds)
	metric.PendingBuildCount(builds)
	metric.RunningBuildCount(builds)
	return builds
}

// provideLogStore is a Wire provider function that provides a
// log datastore, configured from the environment.
func provideLogStore(db *db.DB, config config.Config) core.LogStore {
	s := logs.New(db)
	if config.S3.Bucket != "" {
		p := logs.NewS3Env(
			config.S3.Bucket,
			config.S3.Prefix,
			config.S3.Endpoint,
			config.S3.PathStyle,
		)
		return logs.NewCombined(p, s)
	}
	if config.AzureBlob.ContainerName != "" {
		p := logs.NewAzureBlobEnv(
			config.AzureBlob.ContainerName,
			config.AzureBlob.StorageAccountName,
			config.AzureBlob.StorageAccessKey,
		)
		return logs.NewCombined(p, s)
	}
	return s
}

// provideStageStore is a Wire provider function that provides a
// stage datastore, configured from the environment, with metrics
// enabled.
func provideStageStore(db *db.DB) core.StageStore {
	stages := stage.New(db)
	metric.PendingJobCount(stages)
	metric.RunningJobCount(stages)
	return stages
}

// provideRepoStore is a Wire provider function that provides a
// user datastore, configured from the environment, with metrics
// enabled.
func provideRepoStore(db *db.DB) core.RepositoryStore {
	repos := repos.New(db)
	metric.RepoCount(repos)
	return repos
}

// provideBatchStore is a Wire provider function that provides a
// batcher. If the experimental batcher is enabled it is returned.
func provideBatchStore(db *db.DB, config config.Config) core.Batcher {
	if config.Database.LegacyBatch {
		return batch.New(db)
	}
	return batch2.New(db)
}

// provideUserStore is a Wire provider function that provides a
// user datastore, configured from the environment, with metrics
// enabled.
func provideUserStore(db *db.DB, enc encrypt.Encrypter, config config.Config) core.UserStore {
	// create the user store with encryption iff the user
	// encryption feature flag is enabled.
	//
	// why not enable by default?  because the user table is
	// accessed on every http request and we are unsure what,
	// if any performance implications user table encryption
	// may have on the system.
	//
	// it is very possible there are zero material performance
	// implications, however, if there is a performance regression
	// we could look at implementing in-memory lru caching, which
	// we already employ in other areas of the software.
	if config.Database.EncryptUserTable {
		logrus.Debugln("main: database encryption enabled for user table")
		users := user.New(db, enc)
		metric.UserCount(users)
		return users
	}

	noenc, _ := encrypt.New("")
	users := user.New(db, noenc)
	metric.UserCount(users)
	return users
}
