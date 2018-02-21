// Copyright 2018 Drone.IO Inc
// Use of this software is governed by the Drone Enterpise License
// that can be found in the LICENSE file.

package secrets

import (
	"github.com/drone/drone/model"
	"github.com/drone/drone/store"
)

// NewDefault returns the Store wrapped as a Service.
func NewDefault(store store.Store) model.SecretService {
	return New(store)
}

// Plugin defines the required interface for implementing a remote
// secret plugin and sourcing secrets from an external source.
type Plugin interface {
	SecretListBuild(*model.Repo, *model.Build) ([]*model.Secret, error)
}

// Extend exetends the base secret service with the plugin.
func Extend(base model.SecretService, with Plugin) model.SecretService {
	return &extender{base, with}
}

type extender struct {
	model.SecretService
	plugin Plugin
}

// extends the base secret service and combines the secret list with the
// secret list returned by the plugin.
func (e *extender) SecretListBuild(repo *model.Repo, build *model.Build) ([]*model.Secret, error) {
	base, err := e.SecretService.SecretListBuild(repo, build)
	if err != nil {
		return nil, err
	}
	with, err := e.plugin.SecretListBuild(repo, build)
	if err != nil {
		return nil, err
	}
	for _, secret := range base {
		with = append(with, secret)
	}
	return with, nil
}
