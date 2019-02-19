// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package secret

import (
	"context"
	"strings"

	"github.com/drone/drone/core"
)

// Static returns a new static Secret controller.
func Static(secrets []*core.Secret) core.SecretService {
	return &staticController{secrets: secrets}
}

type staticController struct {
	secrets []*core.Secret
}

func (c *staticController) Find(ctx context.Context, in *core.SecretArgs) (*core.Secret, error) {
	for _, secret := range c.secrets {
		if !strings.EqualFold(secret.Name, in.Name) {
			continue
		}
		// The secret can be restricted to non-pull request
		// events. If the secret is restricted, return
		// empty results.
		if secret.PullRequest == false &&
			in.Build.Event == core.EventPullRequest {
			continue
		}
		return secret, nil
	}
	return nil, nil
}
