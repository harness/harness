// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package secret

import (
	"context"
	"strings"

	"github.com/drone/drone/core"
)

// Combine combines the secret services, allowing the system
// to get pipeline secrets from multiple sources.
func Combine(services ...core.SecretService) core.SecretService {
	return &combined{services}
}

type combined struct {
	sources []core.SecretService
}

func (c *combined) Find(ctx context.Context, in *core.SecretArgs) (*core.Secret, error) {
	// Ignore any requests for the .docker/config.json file.
	// This file is reserved for internal use only, and is
	// never exposed to the build environment.
	if isDockerConfig(in.Name) {
		return nil, nil
	}

	for _, source := range c.sources {
		secret, err := source.Find(ctx, in)
		if err != nil {
			return nil, err
		}
		if secret == nil {
			continue
		}
		// if the secret object is not nil, but is empty
		// we should assume the secret service returned a
		// 204 no content, and proceed to the next service
		// in the chain.
		if secret.Data == "" {
			continue
		}
		return secret, nil
	}
	return nil, nil
}

// helper function returns true if the build event matches the
// docker_auth_config variable name.
func isDockerConfig(name string) bool {
	return strings.EqualFold(name, "docker_auth_config") ||
		strings.EqualFold(name, ".dockerconfigjson") ||
		strings.EqualFold(name, ".dockerconfig")
}
