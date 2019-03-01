// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package registry

import (
	"context"

	"github.com/drone/drone-go/plugin/registry"
	"github.com/drone/drone/core"
	"github.com/drone/drone/logger"
)

// EndpointSource returns a registry credential provider
// that sources registry credentials from an http endpoint.
func EndpointSource(endpoint, secret string, skipVerify bool) core.RegistryService {
	return &service{
		endpoint:   endpoint,
		secret:     secret,
		skipVerify: skipVerify,
	}
}

type service struct {
	endpoint   string
	secret     string
	skipVerify bool
}

func (c *service) List(ctx context.Context, in *core.RegistryArgs) ([]*core.Registry, error) {
	if c.endpoint == "" {
		return nil, nil
	}
	logger := logger.FromContext(ctx)
	logger.Trace("registry: plugin: get credentials")

	req := &registry.Request{
		Repo:  toRepo(in.Repo),
		Build: toBuild(in.Build),
	}
	client := registry.Client(c.endpoint, c.secret, c.skipVerify)
	res, err := client.List(ctx, req)
	if err != nil {
		logger.WithError(err).Warn("registry: plugin: cannot get credentials")
		return nil, err
	}

	var registries []*core.Registry
	for _, registry := range res {
		registries = append(registries, &core.Registry{
			Address:  registry.Address,
			Username: registry.Username,
			Password: registry.Password,
		})
		logger.WithField("address", registry.Address).
			Trace("registry: plugin: found credentials")
	}
	return registries, nil
}
