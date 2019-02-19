// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package registry

import (
	"context"
	"strings"

	"github.com/drone/drone/core"
	"github.com/drone/drone/plugin/registry/auths"
)

// Static returns a new static credentials controller.
func Static(secrets []*core.Secret) core.RegistryService {
	return &staticController{secrets: secrets}
}

type staticController struct {
	secrets []*core.Secret
}

func (c *staticController) List(ctx context.Context, in *core.RegistryArgs) ([]*core.Registry, error) {
	for _, secret := range c.secrets {
		if !isRegistrySecret(secret.Name) {
			continue
		}
		// The secret can be restricted to non-pull request
		// events. If the secret is restricted, return
		// empty results.
		if secret.PullRequest == false &&
			in.Build.Event == core.EventPullRequest {
			continue
		}
		return auths.ParseString(secret.Data)
	}
	return nil, nil
}

func isRegistrySecret(name string) bool {
	return strings.EqualFold(name, "docker_auth_config") ||
		strings.EqualFold(name, ".dockerconfig") ||
		strings.EqualFold(name, ".dockerconfigjson") ||
		strings.EqualFold(name, "_docker")
}
