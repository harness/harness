// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/gitrpc/server"
	"github.com/harness/gitness/internal/webhook"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
	"github.com/kelseyhightower/envconfig"
)

// load returns the system configuration from the
// host environment.
func load() (*types.Config, error) {
	config := new(types.Config)
	// read the configuration from the environment and
	// populate the configuration structure.
	err := envconfig.Process("", config)
	if err != nil {
		return nil, err
	}

	err = ensureInstanceIDIsSet(config)
	if err != nil {
		return nil, fmt.Errorf("unable to ensure that instance ID is set in config: %w", err)
	}

	err = ensureGitRootIsSet(config)
	if err != nil {
		return nil, fmt.Errorf("unable to ensure that git root is set in config: %w", err)
	}

	return config, nil
}

func ensureInstanceIDIsSet(config *types.Config) error {
	if config.InstanceID == "" {
		// use the hostname as default id of the instance
		hostName, err := os.Hostname()
		if err != nil {
			return err
		}

		// Always cast to lower and remove all unwanted chars
		// NOTE: this could theoretically lead to overlaps, then it should be passed explicitly
		// NOTE: for k8s names/ids below modifications are all noops
		// https://kubernetes.io/docs/concepts/overview/working-with-objects/names/
		hostName = strings.ToLower(hostName)
		hostName = strings.Map(func(r rune) rune {
			switch {
			case 'a' <= r && r <= 'z':
				return r
			case '0' <= r && r <= '9':
				return r
			case r == '-', r == '.':
				return r
			default:
				return '_'
			}
		}, hostName)

		config.InstanceID = hostName
	}

	return nil
}

func ensureGitRootIsSet(config *types.Config) error {
	if config.Git.Root == "" {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		config.Git.Root = filepath.Join(homedir, ".gitness")
	}

	return nil
}

// PackageConfigsWireSet contains providers that generate configs required for sub packages.
var PackageConfigsWireSet = wire.NewSet(
	ProvideGitRPCServerConfig,
	ProvideGitRPCClientConfig,
	ProvideEventsConfig,
	ProvideWebhookConfig,
)

func ProvideGitRPCServerConfig(config *types.Config) server.Config {
	return server.Config{
		Bind:          config.Server.GRPC.Bind,
		GitRoot:       config.Git.Root,
		ReposTempPath: config.Git.ReposTempPath,
	}
}

func ProvideGitRPCClientConfig(config *types.Config) gitrpc.Config {
	return gitrpc.Config{
		Bind: config.Server.GRPC.Bind,
	}
}

func ProvideEventsConfig(config *types.Config) events.Config {
	return events.Config{
		Mode:            events.Mode(config.Events.Mode),
		Namespace:       config.Events.Namespace,
		MaxStreamLength: config.Events.MaxStreamLength,
	}
}

func ProvideWebhookConfig(config *types.Config) webhook.Config {
	return webhook.Config{
		// Use instanceID as readerName as every instance should be one reader
		EventReaderName:     config.InstanceID,
		Concurrency:         config.Webhook.Concurrency,
		MaxRetryCount:       config.Webhook.MaxRetryCount,
		AllowLoopback:       config.Webhook.AllowLoopback,
		AllowPrivateNetwork: config.Webhook.AllowPrivateNetwork,
	}
}
