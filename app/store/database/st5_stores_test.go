// Copyright 2026 Harness, Inc.
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

package database_test

import (
	"context"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/app/store/database/migrate"
	storedb "github.com/harness/gitness/store/database"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const st5SqliteDriver = "sqlite3"

// st5Stores builds every store provided by the package wire set on top of the given database.
// Dependencies that are only used after a database call succeeded are left nil.
//
//nolint:funlen
func st5Stores(db *sqlx.DB) map[string]any {
	return map[string]any{
		"PrincipalStore":                 database.ProvidePrincipalStore(db, store.ToLowerPrincipalUIDTransformation),
		"UserGroupStore":                 database.ProvideUserGroupStore(db),
		"UserGroupReviewerStore":         database.ProvideUserGroupReviewerStore(db, nil, nil),
		"PrincipalInfoView":              database.ProvidePrincipalInfoView(db),
		"InfraProviderResourceView":      database.ProvideInfraProviderResourceView(db, nil),
		"SpacePathStore":                 database.ProvideSpacePathStore(db, store.ToLowerSpacePathTransformation),
		"SpaceStore":                     database.ProvideSpaceStore(db, nil, nil),
		"RepoStore":                      database.ProvideRepoStore(db, nil, nil, nil),
		"RepoLangStore":                  database.ProvideRepoLangStore(db),
		"LinkRepoStore":                  database.ProvideLinkRepoStore(db),
		"LinkedPullReqStore":             database.ProvideLinkedPullReqStore(db),
		"RuleStore":                      database.ProvideRuleStore(db, nil),
		"JobStore":                       database.ProvideJobStore(db),
		"PipelineStore":                  database.ProvidePipelineStore(db),
		"InfraProviderConfigStore":       database.ProvideInfraProviderConfigStore(db, nil),
		"InfraProviderResourceStore":     database.ProvideInfraProviderResourceStore(db, nil),
		"GitspaceConfigStore":            database.ProvideGitspaceConfigStore(db, nil, nil, nil),
		"GitspaceSettingsStore":          database.ProvideGitspaceSettingsStore(db),
		"GitspaceInstanceStore":          database.ProvideGitspaceInstanceStore(db, nil),
		"StageStore":                     database.ProvideStageStore(db),
		"StepStore":                      database.ProvideStepStore(db),
		"SecretStore":                    database.ProvideSecretStore(db),
		"ConnectorStore":                 database.ProvideConnectorStore(db, nil),
		"TemplateStore":                  database.ProvideTemplateStore(db),
		"TriggerStore":                   database.ProvideTriggerStore(db),
		"ExecutionStore":                 database.ProvideExecutionStore(db),
		"PluginStore":                    database.ProvidePluginStore(db),
		"MembershipStore":                database.ProvideMembershipStore(db, nil, nil, nil),
		"TokenStore":                     database.ProvideTokenStore(db),
		"PullReqStore":                   database.ProvidePullReqStore(db, nil),
		"PullReqActivityStore":           database.ProvidePullReqActivityStore(db, nil),
		"CodeCommentView":                database.ProvideCodeCommentView(db),
		"PullReqReviewStore":             database.ProvidePullReqReviewStore(db),
		"PullReqReviewerStore":           database.ProvidePullReqReviewerStore(db, nil),
		"PullReqWatcherStore":            database.ProvidePullReqWatcherStore(db, nil),
		"PullReqReviewerSuggestionStore": database.ProvidePullReqReviewerSuggestionStore(db),
		"PullReqFileViewStore":           database.ProvidePullReqFileViewStore(db),
		"PullReqFileGroupStore":          database.ProvidePullReqFileGroupStore(db),
		"AutoMergeStore":                 database.ProvideAutoMergeStore(db),
		"MergeQueueStore":                database.ProvideMergeQueueStore(db),
		"MergeQueueEntryStore":           database.ProvideMergeQueueEntryStore(db),
		"WebhookStore":                   database.ProvideWebhookStore(db),
		"WebhookExecutionStore":          database.ProvideWebhookExecutionStore(db),
		"CheckStore":                     database.ProvideCheckStore(db, nil),
		"SettingsStore":                  database.ProvideSettingsStore(db),
		"PublicAccessStore":              database.ProvidePublicAccessStore(db),
		"PublicKeyStore":                 database.ProvidePublicKeyStore(db),
		"PublicKeySubKeyStore":           database.ProvidePublicKeySubKeyStore(db),
		"GitSignatureResultStore":        database.ProvideGitSignatureResultStore(db),
		"BranchStore":                    database.ProvideBranchStore(db),
		"RepoActivityStore":              database.ProvideRepoActivityStore(db),
		"GitspaceEventStore":             database.ProvideGitspaceEventStore(db),
		"LabelStore":                     database.ProvideLabelStore(db),
		"LabelValueStore":                database.ProvideLabelValueStore(db),
		"PullReqLabelStore":              database.ProvidePullReqLabelStore(db),
		"LFSObjectStore":                 database.ProvideLFSObjectStore(db),
		"InfraProviderTemplateStore":     database.ProvideInfraProviderTemplateStore(db),
		"InfraProvisionedStore":          database.ProvideInfraProvisionedStore(db),
		"UsageMetricStore":               database.ProvideUsageMetricStore(db),
		"CDEGatewayStore":                database.ProvideCDEGatewayStore(db),
		"FavoriteStore":                  database.ProvideFavoriteStore(db),
		"AutolinkStore":                  database.ProvideAutolinkStore(db),
		"AITaskStore":                    database.ProvideAITaskStore(db),
		"PullReqLabelSuggestionStore":    database.ProvidePullReqLabelSuggestionStore(db),
	}
}

func st5OpenDB(t *testing.T, migrated bool) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Connect(st5SqliteDriver, filepath.Join(t.TempDir(), "st5.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	if migrated {
		require.NoError(t, migrate.Migrate(context.Background(), db))
	}

	return db
}

func TestSt5ProvideStores(t *testing.T) {
	db := st5OpenDB(t, false)

	stores := st5Stores(db)
	require.Greater(t, len(stores), 60)
	for name, s := range stores {
		require.NotNil(t, s, name)
	}
}

func TestSt5ProvideDatabase(t *testing.T) {
	ctx := context.Background()

	t.Run("sqlite", func(t *testing.T) {
		db, err := database.ProvideDatabase(ctx, storedb.Config{
			Driver:          st5SqliteDriver,
			Datasource:      filepath.Join(t.TempDir(), "provide.db"),
			MaxOpenConns:    2,
			MaxIdleConns:    1,
			ConnMaxLifetime: time.Minute,
		})
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })

		// the migrations ran: a migrated table can be queried.
		var count int
		require.NoError(t, db.GetContext(ctx, &count, `SELECT count(*) FROM principals`))
	})

	t.Run("invalid pool config", func(t *testing.T) {
		_, err := database.ProvideDatabase(ctx, storedb.Config{
			Driver:     st5SqliteDriver,
			Datasource: filepath.Join(t.TempDir(), "pool.db"),
		})
		require.ErrorContains(t, err, "MaxOpenConns")
	})

	t.Run("unknown driver", func(t *testing.T) {
		_, err := database.ProvideDatabase(ctx, storedb.Config{Driver: "st5-unknown", Datasource: "x"})
		require.Error(t, err)
	})
}

// st5Sweep calls every exported method of every store with zero value arguments and returns the number of calls
// and the number of calls that returned an error. Panics caused by zero value dereferences are recovered.
func st5Sweep(ctx context.Context, db *sqlx.DB) (calls, failed int) {
	names := make([]string, 0)
	stores := st5Stores(db)
	for name := range stores {
		names = append(names, name)
	}
	sort.Strings(names)

	ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()
	errType := reflect.TypeOf((*error)(nil)).Elem()

	for _, name := range names {
		v := reflect.ValueOf(stores[name])
		for i := range v.NumMethod() {
			// streaming methods run in a goroutine, where a zero value panic cannot be recovered.
			if strings.Contains(v.Type().Method(i).Name, "Stream") {
				continue
			}
			mt := v.Method(i).Type()
			numIn := mt.NumIn()
			if mt.IsVariadic() {
				numIn--
			}
			args := make([]reflect.Value, numIn)
			for j := range numIn {
				switch at := mt.In(j); {
				case at.Implements(ctxType) && at.Kind() == reflect.Interface:
					args[j] = reflect.ValueOf(ctx)
				case at.Kind() == reflect.Ptr:
					args[j] = reflect.New(at.Elem())
				default:
					args[j] = reflect.Zero(at)
				}
			}

			calls++
			if st5Call(v.Method(i), args, errType) {
				failed++
			}
		}
	}

	return calls, failed
}

// TestSt5StoresClosedDatabase makes every store method fail on a closed database.
func TestSt5StoresClosedDatabase(t *testing.T) {
	db := st5OpenDB(t, true)
	require.NoError(t, db.Close())

	calls, failed := st5Sweep(context.Background(), db)
	require.Greater(t, calls, 300)
	require.Greater(t, failed, calls*8/10)
}

// TestSt5StoresEmptyDatabase calls every store method on a migrated database without data.
func TestSt5StoresEmptyDatabase(t *testing.T) {
	for _, foreignKeys := range []string{"on", "off"} {
		t.Run("foreign_keys_"+foreignKeys, func(t *testing.T) {
			db, err := sqlx.Connect(st5SqliteDriver,
				"file:"+filepath.Join(t.TempDir(), "st5.db")+"?_foreign_keys="+foreignKeys)
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })
			require.NoError(t, migrate.Migrate(context.Background(), db))
			_, err = db.Exec("PRAGMA foreign_keys = " + foreignKeys)
			require.NoError(t, err)

			calls, _ := st5Sweep(context.Background(), db)
			require.Greater(t, calls, 300)
		})
	}
}

// st5Call invokes fn and reports whether it returned a non nil error. Panics are recovered.
func st5Call(fn reflect.Value, args []reflect.Value, errType reflect.Type) (hasErr bool) {
	defer func() { _ = recover() }()

	for _, out := range fn.Call(args) {
		if out.Type() == errType && !out.IsNil() {
			hasErr = true
		}
	}

	return hasErr
}
