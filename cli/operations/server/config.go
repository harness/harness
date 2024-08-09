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
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/harness/gitness/app/gitspace/infrastructure"
	"github.com/harness/gitness/app/gitspace/orchestrator"
	"github.com/harness/gitness/app/gitspace/orchestrator/ide"
	"github.com/harness/gitness/app/services/cleanup"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/gitspaceevent"
	"github.com/harness/gitness/app/services/keywordsearch"
	"github.com/harness/gitness/app/services/notification"
	"github.com/harness/gitness/app/services/trigger"
	"github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/blob"
	"github.com/harness/gitness/events"
	gittypes "github.com/harness/gitness/git/types"
	"github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/types"

	"github.com/kelseyhightower/envconfig"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (
	schemeHTTP     = "http"
	schemeHTTPS    = "https"
	gitnessHomeDir = ".gitness"
	blobDir        = "blob"
)

// LoadConfig returns the system configuration from the
// host environment.
func LoadConfig() (*types.Config, error) {
	config := new(types.Config)
	err := envconfig.Process("", config)
	if err != nil {
		return nil, err
	}

	config.InstanceID, err = getSanitizedMachineName()
	if err != nil {
		return nil, fmt.Errorf("unable to ensure that instance ID is set in config: %w", err)
	}

	err = backfillURLs(config)
	if err != nil {
		return nil, fmt.Errorf("failed to backfil urls: %w", err)
	}

	if config.Git.HookPath == "" {
		executablePath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("failed to get path of current executable: %w", err)
		}

		config.Git.HookPath = executablePath
	}

	if config.Git.Root == "" {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}

		newPath := filepath.Join(homedir, gitnessHomeDir)
		config.Git.Root = newPath

		oldPath := filepath.Join(homedir, ".gitrpc")
		if _, err := os.Stat(oldPath); err == nil {
			if err := os.Rename(oldPath, newPath); err != nil {
				config.Git.Root = oldPath
			}
		}
	}

	return config, nil
}

//nolint:gocognit // refactor if required
func backfillURLs(config *types.Config) error {
	// default base url
	// TODO: once we actually use the config.Server.HTTP.Proto, we have to update that here.
	scheme, host, port, path := schemeHTTP, "localhost", "", ""

	// by default drop scheme's default port
	if (scheme != schemeHTTP || config.Server.HTTP.Port != 80) &&
		(scheme != schemeHTTPS || config.Server.HTTP.Port != 443) {
		port = fmt.Sprint(config.Server.HTTP.Port)
	}

	// backfil internal URLS before continuing override with user provided base (which is external facing)
	if config.URL.Internal == "" {
		config.URL.Internal = combineToRawURL(scheme, "localhost", port, "")
	}
	if config.URL.Container == "" {
		config.URL.Container = combineToRawURL(scheme, "host.docker.internal", port, "")
	}

	// override base with whatever user explicit override
	//nolint:nestif // simple conditional override of all elements
	if config.URL.Base != "" {
		u, err := url.Parse(config.URL.Base)
		if err != nil {
			return fmt.Errorf("failed to parse base url '%s': %w", config.URL.Base, err)
		}
		if u.Scheme != schemeHTTP && u.Scheme != schemeHTTPS {
			return fmt.Errorf(
				"base url scheme '%s' is not supported (valid values: %v)",
				u.Scheme,
				[]string{
					schemeHTTP,
					schemeHTTPS,
				},
			)
		}
		// url parsing allows empty hostname - we don't want that
		if u.Hostname() == "" {
			return fmt.Errorf("a non-empty base url host has to be provided")
		}

		// take everything as is (e.g. if user explicitly adds port 80 for http we take it)
		scheme = u.Scheme
		host = u.Hostname()
		port = u.Port()
		path = u.Path
	}

	// create base URL object
	baseURLRaw := combineToRawURL(scheme, host, port, path)
	baseURL, err := url.Parse(baseURLRaw)
	if err != nil {
		return fmt.Errorf("failed to parse derived base url '%s': %w", baseURLRaw, err)
	}

	// backfill all external URLs that weren't explicitly overwritten
	if config.URL.Base == "" {
		config.URL.Base = baseURL.String()
	}
	if config.URL.API == "" {
		config.URL.API = baseURL.JoinPath("api").String()
	}
	if config.URL.Git == "" {
		config.URL.Git = baseURL.JoinPath("git").String()
	}
	if config.URL.UI == "" {
		config.URL.UI = baseURL.String()
	}

	return nil
}

func combineToRawURL(scheme, host, port, path string) string {
	urlRAW := scheme + "://" + host

	// only add port if explicitly provided
	if port != "" {
		urlRAW += ":" + port
	}

	// only add path if it's not empty and non-root
	path = strings.Trim(path, "/")
	if path != "" {
		urlRAW += "/" + path
	}

	return urlRAW
}

// getSanitizedMachineName gets the name of the machine and returns it in sanitized format.
func getSanitizedMachineName() (string, error) {
	// use the hostname as default id of the instance
	hostName, err := os.Hostname()
	if err != nil {
		return "", err
	}

	// Always cast to lower and remove all unwanted chars
	// NOTE: this could theoretically lead to overlaps, then it should be passed explicitly
	// NOTE: for k8s names/ids below modifications are all noops
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/names/

	// The following code will:
	// * remove invalid runes
	// * remove diacritical marks (ie "smörgåsbord" to "smorgasbord")
	// * lowercase A-Z to a-z
	// * leave only a-z, 0-9, '-', '.' and replace everything else with '_'
	hostName, _, err = transform.String(
		transform.Chain(
			norm.NFD,
			runes.ReplaceIllFormed(),
			runes.Remove(runes.In(unicode.Mn)),
			runes.Map(func(r rune) rune {
				switch {
				case 'A' <= r && r <= 'Z':
					return r + 32
				case 'a' <= r && r <= 'z':
					return r
				case '0' <= r && r <= '9':
					return r
				case r == '-', r == '.':
					return r
				default:
					return '_'
				}
			}),
			norm.NFC),
		hostName)
	if err != nil {
		return "", err
	}

	return hostName, nil
}

// ProvideDatabaseConfig loads the database config from the main config.
func ProvideDatabaseConfig(config *types.Config) database.Config {
	return database.Config{
		Driver:     config.Database.Driver,
		Datasource: config.Database.Datasource,
	}
}

// ProvideBlobStoreConfig loads the blob store config from the main config.
func ProvideBlobStoreConfig(config *types.Config) (blob.Config, error) {
	// Prefix home directory in case of filesystem blobstore
	if config.BlobStore.Provider == blob.ProviderFileSystem && config.BlobStore.Bucket == "" {
		var homedir string
		homedir, err := os.UserHomeDir()
		if err != nil {
			return blob.Config{}, err
		}

		config.BlobStore.Bucket = filepath.Join(homedir, gitnessHomeDir, blobDir)
	}
	return blob.Config{
		Provider:              config.BlobStore.Provider,
		Bucket:                config.BlobStore.Bucket,
		KeyPath:               config.BlobStore.KeyPath,
		TargetPrincipal:       config.BlobStore.TargetPrincipal,
		ImpersonationLifetime: config.BlobStore.ImpersonationLifetime,
	}, nil
}

// ProvideGitConfig loads the git config from the main config.
func ProvideGitConfig(config *types.Config) gittypes.Config {
	return gittypes.Config{
		Trace:    config.Git.Trace,
		Root:     config.Git.Root,
		TmpDir:   config.Git.TmpDir,
		HookPath: config.Git.HookPath,
		LastCommitCache: gittypes.LastCommitCacheConfig{
			Mode:     config.Git.LastCommitCache.Mode,
			Duration: config.Git.LastCommitCache.Duration,
		},
	}
}

// ProvideEventsConfig loads the events config from the main config.
func ProvideEventsConfig(config *types.Config) events.Config {
	return events.Config{
		Mode:                  config.Events.Mode,
		Namespace:             config.Events.Namespace,
		MaxStreamLength:       config.Events.MaxStreamLength,
		ApproxMaxStreamLength: config.Events.ApproxMaxStreamLength,
	}
}

// ProvideWebhookConfig loads the webhook service config from the main config.
func ProvideWebhookConfig(config *types.Config) webhook.Config {
	return webhook.Config{
		UserAgentIdentity:   config.Webhook.UserAgentIdentity,
		HeaderIdentity:      config.Webhook.HeaderIdentity,
		EventReaderName:     config.InstanceID,
		Concurrency:         config.Webhook.Concurrency,
		MaxRetries:          config.Webhook.MaxRetries,
		AllowPrivateNetwork: config.Webhook.AllowPrivateNetwork,
		AllowLoopback:       config.Webhook.AllowLoopback,
	}
}

func ProvideNotificationConfig(config *types.Config) notification.Config {
	return notification.Config{
		EventReaderName: config.InstanceID,
		Concurrency:     config.Notification.Concurrency,
		MaxRetries:      config.Notification.MaxRetries,
	}
}

// ProvideTriggerConfig loads the trigger service config from the main config.
func ProvideTriggerConfig(config *types.Config) trigger.Config {
	return trigger.Config{
		EventReaderName: config.InstanceID,
		Concurrency:     config.Webhook.Concurrency,
		MaxRetries:      config.Webhook.MaxRetries,
	}
}

// ProvideLockConfig generates the `lock` package config from the gitness config.
func ProvideLockConfig(config *types.Config) lock.Config {
	return lock.Config{
		App:           config.Lock.AppNamespace,
		Namespace:     config.Lock.DefaultNamespace,
		Provider:      config.Lock.Provider,
		Expiry:        config.Lock.Expiry,
		Tries:         config.Lock.Tries,
		RetryDelay:    config.Lock.RetryDelay,
		DriftFactor:   config.Lock.DriftFactor,
		TimeoutFactor: config.Lock.TimeoutFactor,
	}
}

// ProvidePubsubConfig loads the pubsub config from the main config.
func ProvidePubsubConfig(config *types.Config) pubsub.Config {
	return pubsub.Config{
		App:            config.PubSub.AppNamespace,
		Namespace:      config.PubSub.DefaultNamespace,
		Provider:       config.PubSub.Provider,
		HealthInterval: config.PubSub.HealthInterval,
		SendTimeout:    config.PubSub.SendTimeout,
		ChannelSize:    config.PubSub.ChannelSize,
	}
}

// ProvideCleanupConfig loads the cleanup service config from the main config.
func ProvideCleanupConfig(config *types.Config) cleanup.Config {
	return cleanup.Config{
		WebhookExecutionsRetentionTime:   config.Webhook.RetentionTime,
		DeletedRepositoriesRetentionTime: config.Repos.DeletedRetentionTime,
	}
}

// ProvideCodeOwnerConfig loads the codeowner config from the main config.
func ProvideCodeOwnerConfig(config *types.Config) codeowners.Config {
	return codeowners.Config{
		FilePaths: config.CodeOwners.FilePaths,
	}
}

// ProvideKeywordSearchConfig loads the keyword search service config from the main config.
func ProvideKeywordSearchConfig(config *types.Config) keywordsearch.Config {
	return keywordsearch.Config{
		EventReaderName: config.InstanceID,
		Concurrency:     config.KeywordSearch.Concurrency,
		MaxRetries:      config.KeywordSearch.MaxRetries,
	}
}

func ProvideJobsConfig(config *types.Config) job.Config {
	return job.Config{
		InstanceID:                  config.InstanceID,
		BackgroundJobsMaxRunning:    config.BackgroundJobs.MaxRunning,
		BackgroundJobsRetentionTime: config.BackgroundJobs.RetentionTime,
	}
}

// ProvideDockerConfig loads config for Docker.
func ProvideDockerConfig(config *types.Config) (*infraprovider.DockerConfig, error) {
	if config.Docker.MachineHostName == "" {
		gitnessBaseURL, err := url.Parse(config.URL.Base)
		if err != nil {
			return nil, fmt.Errorf("unable to parse Gitness base URL %s: %w", gitnessBaseURL, err)
		}
		config.Docker.MachineHostName = gitnessBaseURL.Hostname()
	}

	return &infraprovider.DockerConfig{
		DockerHost:            config.Docker.Host,
		DockerAPIVersion:      config.Docker.APIVersion,
		DockerCertPath:        config.Docker.CertPath,
		DockerTLSVerify:       config.Docker.TLSVerify,
		DockerMachineHostName: config.Docker.MachineHostName,
	}, nil
}

// ProvideIDEVSCodeWebConfig loads the VSCode Web IDE config from the main config.
func ProvideIDEVSCodeWebConfig(config *types.Config) *ide.VSCodeWebConfig {
	return &ide.VSCodeWebConfig{
		Port: config.IDE.VSCodeWeb.Port,
	}
}

// ProvideIDEVSCodeConfig loads the VSCode IDE config from the main config.
func ProvideIDEVSCodeConfig(config *types.Config) *ide.VSCodeConfig {
	return &ide.VSCodeConfig{
		Port: config.IDE.VSCode.Port,
	}
}

// ProvideGitspaceOrchestratorConfig loads the Gitspace orchestrator config from the main config.
func ProvideGitspaceOrchestratorConfig(config *types.Config) *orchestrator.Config {
	return &orchestrator.Config{
		DefaultBaseImage: config.Gitspace.DefaultBaseImage,
	}
}

// ProvideGitspaceInfraProvisionerConfig loads the Gitspace infra provisioner config from the main config.
func ProvideGitspaceInfraProvisionerConfig(config *types.Config) *infrastructure.Config {
	return &infrastructure.Config{
		AgentPort: config.Gitspace.AgentPort,
	}
}

// ProvideGitspaceEventConfig loads the gitspace event service config from the main config.
func ProvideGitspaceEventConfig(config *types.Config) *gitspaceevent.Config {
	return &gitspaceevent.Config{
		EventReaderName: config.InstanceID,
		Concurrency:     config.Gitspace.Events.Concurrency,
		MaxRetries:      config.Gitspace.Events.MaxRetries,
	}
}
