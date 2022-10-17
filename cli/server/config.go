// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/harness/gitness/types"

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

	err = ensureGitRootIsSet(config)
	if err != nil {
		return nil, fmt.Errorf("unable to ensure that git root is set in config: %w", err)
	}

	return config, nil
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
