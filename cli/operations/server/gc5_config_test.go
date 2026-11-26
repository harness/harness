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

package server

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/harness/gitness/blob"
	"github.com/harness/gitness/events"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

// gc5LoadConfig loads the config with a clean environment and an isolated home directory.
func gc5LoadConfig(t *testing.T, env map[string]string) (*types.Config, string) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	for k, v := range env {
		t.Setenv(k, v)
	}

	cfg, err := LoadConfig()
	require.NoError(t, err)

	return cfg, home
}

func TestGc5LoadConfigDefaults(t *testing.T) {
	cfg, home := gc5LoadConfig(t, nil)

	require.Equal(t, filepath.Join(home, ".gitness"), cfg.Git.Root)
	require.NotEmpty(t, cfg.Git.HookPath)
	require.Equal(t, "main", cfg.Git.DefaultBranch)
	require.NotEmpty(t, cfg.InstanceID)
	require.Regexp(t, regexp.MustCompile(`^[a-z0-9_.-]+$`), cfg.InstanceID)
}

func TestGc5LoadConfigExplicitValues(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	cfg, _ := gc5LoadConfig(t, map[string]string{
		"GITNESS_GIT_ROOT":      root,
		"GITNESS_GIT_HOOK_PATH": "/opt/gc5-hook",
	})

	require.Equal(t, root, cfg.Git.Root)
	require.Equal(t, "/opt/gc5-hook", cfg.Git.HookPath)
}

func TestGc5LoadConfigMigratesLegacyGitRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	legacy := filepath.Join(home, ".gitrpc")
	require.NoError(t, os.MkdirAll(legacy, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(legacy, "marker"), []byte("x"), 0o600))

	cfg, err := LoadConfig()
	require.NoError(t, err)

	require.Equal(t, filepath.Join(home, ".gitness"), cfg.Git.Root)
	require.FileExists(t, filepath.Join(cfg.Git.Root, "marker"))
	require.NoDirExists(t, legacy)
}

func TestGc5LoadConfigErrors(t *testing.T) {
	t.Setenv("GITNESS_DATABASE_MAX_OPEN_CONNS", "not-a-number")

	_, err := LoadConfig()
	require.Error(t, err)
}

func TestGc5LoadConfigBadURL(t *testing.T) {
	t.Setenv("GITNESS_URL_BASE", "ftp://example.com")
	t.Setenv("HOME", t.TempDir())

	_, err := LoadConfig()
	require.ErrorContains(t, err, "backfil urls")
}

func TestGc5SanitizedMachineName(t *testing.T) {
	name, err := getSanitizedMachineName()

	require.NoError(t, err)
	require.NotEmpty(t, name)
	require.Regexp(t, regexp.MustCompile(`^[a-z0-9_.-]+$`), name)
}

func TestGc5ProvideConfigs(t *testing.T) {
	cfg, _ := gc5LoadConfig(t, map[string]string{
		"GITNESS_DATABASE_DRIVER":                    "postgres",
		"GITNESS_DATABASE_DATASOURCE":                "postgres://gc5",
		"GITNESS_DATABASE_MAX_OPEN_CONNS":            "11",
		"GITNESS_DATABASE_MAX_IDLE_CONNS":            "3",
		"GITNESS_DATABASE_CONN_MAX_LIFETIME":         "7m",
		"GITNESS_GIT_TRACE":                          "true",
		"GITNESS_GIT_TMP_DIR":                        "/tmp/gc5-git",
		"GITNESS_GIT_ROOT":                           "/var/gc5-git",
		"GITNESS_GIT_HOOK_PATH":                      "/opt/gc5-hook",
		"GITNESS_GIT_LAST_COMMIT_CACHE_MODE":         "none",
		"GITNESS_GIT_LAST_COMMIT_CACHE_DURATION":     "2h",
		"GITNESS_EVENTS_MODE":                        "redis",
		"GITNESS_EVENTS_NAMESPACE":                   "gc5ns",
		"GITNESS_EVENTS_MAX_STREAM_LENGTH":           "55",
		"GITNESS_EVENTS_APPROX_MAX_STREAM_LENGTH":    "false",
		"GITNESS_EVENTS_PRODUCER_MAX_ATTEMPTS":       "9",
		"GITNESS_EVENTS_PRODUCER_INITIAL_BACKOFF":    "2s",
		"GITNESS_EVENTS_PRODUCER_BACKOFF_FACTOR":     "2.5",
		"GITNESS_EVENTS_PRODUCER_MAX_BACKOFF":        "20s",
		"GITNESS_WEBHOOK_USER_AGENT_IDENTITY":        "Gc5Agent",
		"GITNESS_WEBHOOK_HEADER_IDENTITY":            "Gc5Header",
		"GITNESS_WEBHOOK_CONCURRENCY":                "6",
		"GITNESS_WEBHOOK_MAX_RETRIES":                "5",
		"GITNESS_WEBHOOK_ALLOW_PRIVATE_NETWORK":      "true",
		"GITNESS_WEBHOOK_ALLOW_LOOPBACK":             "true",
		"GITNESS_WEBHOOK_ALLOW_LINK_LOCAL":           "true",
		"GITNESS_WEBHOOK_INTERNAL_SECRET":            "gc5-internal",
		"GITNESS_WEBHOOK_RETENTION_TIME":             "48h",
		"GITNESS_NOTIFICATION_CONCURRENCY":           "8",
		"GITNESS_NOTIFICATION_MAX_RETRIES":           "2",
		"GITNESS_BRANCH_CONCURRENCY":                 "9",
		"GITNESS_BRANCH_MAX_RETRIES":                 "4",
		"GITNESS_LOCK_PROVIDER":                      "redis",
		"GITNESS_LOCK_EXPIRE":                        "9s",
		"GITNESS_LOCK_TRIES":                         "12",
		"GITNESS_LOCK_RETRY_DELAY":                   "300ms",
		"GITNESS_LOCK_DRIFT_FACTOR":                  "0.02",
		"GITNESS_LOCK_TIMEOUT_FACTOR":                "0.5",
		"GITNESS_LOCK_APP_NAMESPACE":                 "gc5app",
		"GITNESS_LOCK_DEFAULT_NAMESPACE":             "gc5lock",
		"GITNESS_PUBSUB_PROVIDER":                    "redis",
		"GITNESS_PUBSUB_APP_NAMESPACE":               "gc5app",
		"GITNESS_PUBSUB_DEFAULT_NAMESPACE":           "gc5pubsub",
		"GITNESS_PUBSUB_HEALTH_INTERVAL":             "4s",
		"GITNESS_PUBSUB_SEND_TIMEOUT":                "70s",
		"GITNESS_PUBSUB_CHANNEL_SIZE":                "123",
		"GITNESS_REPOS_DELETED_RETENTION_TIME":       "24h",
		"GITNESS_REPOS_DELETED_CLEANUP_CRON":         "1 2 * * *",
		"GITNESS_REPOS_DELETED_CLEANUP_MAX_DURATION": "3m",
		"GITNESS_CODEOWNERS_FILEPATH":                "OWNERS,docs/OWNERS",
		"GITNESS_KEYWORD_SEARCH_CONCURRENCY":         "13",
		"GITNESS_KEYWORD_SEARCH_MAX_RETRIES":         "14",
		"GITNESS_GITSPACE_EVENTS_CONCURRENCY":        "15",
		"GITNESS_GITSPACE_EVENTS_MAX_RETRIES":        "16",
		"GITNESS_GITSPACE_EVENTS_TIMEOUT_IN_MINS":    "17",
		"OTEL_TRACES_EXPORTER":                       "otlp",
		"GITNESS_HTTP_GRPC_TRACING_ENABLED":          "true",
	})

	t.Run("database", func(t *testing.T) {
		dbCfg := ProvideDatabaseConfig(cfg)
		require.Equal(t, "postgres", dbCfg.Driver)
		require.Equal(t, "postgres://gc5", dbCfg.Datasource)
		require.Equal(t, 11, dbCfg.MaxOpenConns)
		require.Equal(t, 3, dbCfg.MaxIdleConns)
		require.Equal(t, 7*time.Minute, dbCfg.ConnMaxLifetime)
	})

	t.Run("git", func(t *testing.T) {
		gitCfg := ProvideGitConfig(cfg)
		require.True(t, gitCfg.Trace)
		require.Equal(t, "/var/gc5-git", gitCfg.Root)
		require.Equal(t, "/tmp/gc5-git", gitCfg.TmpDir)
		require.Equal(t, "/opt/gc5-hook", gitCfg.HookPath)
		require.Equal(t, gitenum.LastCommitCacheMode("none"), gitCfg.LastCommitCache.Mode)
		require.Equal(t, 2*time.Hour, gitCfg.LastCommitCache.Duration)
	})

	t.Run("events", func(t *testing.T) {
		evCfg := ProvideEventsConfig(cfg)
		require.Equal(t, events.Mode("redis"), evCfg.Mode)
		require.Equal(t, "gc5ns", evCfg.Namespace)
		require.Equal(t, int64(55), evCfg.MaxStreamLength)
		require.False(t, evCfg.ApproxMaxStreamLength)
		require.Equal(t, 9, evCfg.ProducerRetry.MaxAttempts)
		require.Equal(t, 2*time.Second, evCfg.ProducerRetry.InitialBackoff)
		require.InDelta(t, 2.5, evCfg.ProducerRetry.BackoffFactor, 0.0001)
		require.Equal(t, 20*time.Second, evCfg.ProducerRetry.MaxBackoff)
	})

	t.Run("webhook and trigger", func(t *testing.T) {
		whCfg := ProvideWebhookConfig(cfg)
		require.Equal(t, "Gc5Agent", whCfg.UserAgentIdentity)
		require.Equal(t, "Gc5Header", whCfg.HeaderIdentity)
		require.Equal(t, cfg.InstanceID, whCfg.EventReaderName)
		require.Equal(t, 6, whCfg.Concurrency)
		require.Equal(t, 5, whCfg.MaxRetries)
		require.True(t, whCfg.AllowPrivateNetwork)
		require.True(t, whCfg.AllowLoopback)
		require.True(t, whCfg.AllowLinkLocal)
		require.Equal(t, "gc5-internal", whCfg.InternalSecret)
		require.True(t, whCfg.TraceHTTP)

		trCfg := ProvideTriggerConfig(cfg)
		require.Equal(t, cfg.InstanceID, trCfg.EventReaderName)
		require.Equal(t, 6, trCfg.Concurrency)
		require.Equal(t, 5, trCfg.MaxRetries)
	})

	t.Run("notification branch and repo activity", func(t *testing.T) {
		nCfg := ProvideNotificationConfig(cfg)
		require.Equal(t, 8, nCfg.Concurrency)
		require.Equal(t, 2, nCfg.MaxRetries)
		require.Equal(t, cfg.InstanceID, nCfg.EventReaderName)

		bCfg := ProvideBranchConfig(cfg)
		require.Equal(t, 9, bCfg.Concurrency)
		require.Equal(t, 4, bCfg.MaxRetries)

		raCfg := ProvideRepoActivityConfig(cfg)
		require.Equal(t, 9, raCfg.Concurrency)
		require.Equal(t, 4, raCfg.MaxRetries)
		require.Equal(t, cfg.InstanceID, raCfg.EventReaderName)
	})

	t.Run("lock and pubsub", func(t *testing.T) {
		lCfg := ProvideLockConfig(cfg)
		require.Equal(t, "gc5app", lCfg.App)
		require.Equal(t, "gc5lock", lCfg.Namespace)
		require.Equal(t, lock.Provider("redis"), lCfg.Provider)
		require.Equal(t, 9*time.Second, lCfg.Expiry)
		require.Equal(t, 12, lCfg.Tries)
		require.Equal(t, 300*time.Millisecond, lCfg.RetryDelay)
		require.InDelta(t, 0.02, lCfg.DriftFactor, 0.0001)
		require.InDelta(t, 0.5, lCfg.TimeoutFactor, 0.0001)

		pCfg := ProvidePubsubConfig(cfg)
		require.Equal(t, "gc5app", pCfg.App)
		require.Equal(t, "gc5pubsub", pCfg.Namespace)
		require.Equal(t, pubsub.Provider("redis"), pCfg.Provider)
		require.Equal(t, 4*time.Second, pCfg.HealthInterval)
		require.Equal(t, 70*time.Second, pCfg.SendTimeout)
		require.Equal(t, 123, pCfg.ChannelSize)
	})

	t.Run("cleanup codeowners and keyword search", func(t *testing.T) {
		cCfg := ProvideCleanupConfig(cfg)
		require.Equal(t, 48*time.Hour, cCfg.WebhookExecutionsRetentionTime)
		require.Equal(t, 24*time.Hour, cCfg.DeletedRepositoriesRetentionTime)
		require.Equal(t, "1 2 * * *", cCfg.DeletedRepositoriesCleanupCron)
		require.Equal(t, 3*time.Minute, cCfg.DeletedRepositoriesCleanupMaxDuration)

		require.Equal(t, []string{"OWNERS", "docs/OWNERS"}, ProvideCodeOwnerConfig(cfg).FilePaths)

		kCfg := ProvideKeywordSearchConfig(cfg)
		require.Equal(t, 13, kCfg.Concurrency)
		require.Equal(t, 14, kCfg.MaxRetries)
		require.Equal(t, cfg.InstanceID, kCfg.EventReaderName)
	})

	t.Run("jobs", func(t *testing.T) {
		jCfg := ProvideJobsConfig(cfg)
		require.Equal(t, cfg.InstanceID, jCfg.InstanceID)
		require.Equal(t, cfg.BackgroundJobs.MaxRunning, jCfg.BackgroundJobsMaxRunning)
		require.Equal(t, cfg.BackgroundJobs.RetentionTime, jCfg.BackgroundJobsRetentionTime)
	})

	t.Run("gitspace events", func(t *testing.T) {
		gCfg := ProvideGitspaceEventConfig(cfg)
		require.Equal(t, 15, gCfg.Concurrency)
		require.Equal(t, 16, gCfg.MaxRetries)
		require.Equal(t, 17, gCfg.TimeoutInMins)
		require.Equal(t, cfg.InstanceID, gCfg.EventReaderName)

		gdCfg := ProvideGitspaceDeleteEventConfig(cfg)
		require.Equal(t, 15, gdCfg.Concurrency)
		require.Equal(t, 16, gdCfg.MaxRetries)
		require.Equal(t, 17, gdCfg.TimeoutInMins)
		require.Equal(t, cfg.InstanceID, gdCfg.EventReaderName)
	})
}

func TestGc5ProvideBlobStoreConfig(t *testing.T) {
	t.Run("filesystem without bucket uses the home directory", func(t *testing.T) {
		cfg, home := gc5LoadConfig(t, map[string]string{"GITNESS_BLOBSTORE_PROVIDER": "filesystem"})

		blobCfg, err := ProvideBlobStoreConfig(cfg)
		require.NoError(t, err)
		require.Equal(t, blob.ProviderFileSystem, blobCfg.Provider)
		require.Equal(t, filepath.Join(home, ".gitness", "blob"), blobCfg.Bucket)
	})

	t.Run("explicit values are passed through", func(t *testing.T) {
		cfg, _ := gc5LoadConfig(t, map[string]string{
			"GITNESS_BLOBSTORE_PROVIDER":               "gcs",
			"GITNESS_BLOBSTORE_BUCKET":                 "gc5-bucket",
			"GITNESS_BLOBSTORE_KEY_PATH":               "/keys/gc5.json",
			"GITNESS_BLOBSTORE_TARGET_PRINCIPAL":       "gc5@example.com",
			"GITNESS_BLOBSTORE_IMPERSONATION_LIFETIME": "3h",
		})

		blobCfg, err := ProvideBlobStoreConfig(cfg)
		require.NoError(t, err)
		require.Equal(t, blob.Provider("gcs"), blobCfg.Provider)
		require.Equal(t, "gc5-bucket", blobCfg.Bucket)
		require.Equal(t, "/keys/gc5.json", blobCfg.KeyPath)
		require.Equal(t, "gc5@example.com", blobCfg.TargetPrincipal)
		require.Equal(t, 3*time.Hour, blobCfg.ImpersonationLifetime)
	})
}
