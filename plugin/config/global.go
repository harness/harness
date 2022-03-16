// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

//go:build !oss
// +build !oss

package config

import (
	"context"
	"time"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/config"

	"github.com/drone/drone/core"
)

// Global returns a configuration service that fetches the yaml
// configuration from a remote endpoint.
func Global(endpoint, signer string, skipVerify bool, timeout time.Duration) core.ConfigService {
	if endpoint == "" {
		return new(global)
	}
	return &global{
		client: config.Client(
			endpoint,
			signer,
			skipVerify,
		),
		timeout: timeout,
	}
}

type global struct {
	client  config.Plugin
	timeout time.Duration
}

func (g *global) Find(ctx context.Context, in *core.ConfigArgs) (*core.Config, error) {
	if g.client == nil {
		return nil, nil
	}
	// include a timeout to prevent an API call from
	// hanging the build process indefinitely. The
	// external service must return a response within
	// the configured timeout (default 1m).
	ctx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	req := &config.Request{
		Repo:  toRepo(in.Repo),
		Build: toBuild(in.Build),
		Token: drone.Token{
			Access:  in.User.Token,
			Refresh: in.User.Refresh,
		},
	}

	res, err := g.client.Find(ctx, req)
	if err != nil {
		return nil, err
	}

	// if no error is returned and the secret is empty,
	// this indicates the client returned No Content,
	// and we should exit with no secret, but no error.
	if res.Data == "" {
		return nil, nil
	}

	return &core.Config{
		Kind: res.Kind,
		Data: res.Data,
	}, nil
}

func toRepo(from *core.Repository) drone.Repo {
	return drone.Repo{
		ID:         from.ID,
		UID:        from.UID,
		UserID:     from.UserID,
		Namespace:  from.Namespace,
		Name:       from.Name,
		Slug:       from.Slug,
		SCM:        from.SCM,
		HTTPURL:    from.HTTPURL,
		SSHURL:     from.SSHURL,
		Link:       from.Link,
		Branch:     from.Branch,
		Private:    from.Private,
		Visibility: from.Visibility,
		Active:     from.Active,
		Config:     from.Config,
		Trusted:    from.Trusted,
		Protected:  from.Protected,
		Timeout:    from.Timeout,
	}
}

func toBuild(from *core.Build) drone.Build {
	return drone.Build{
		ID:           from.ID,
		RepoID:       from.RepoID,
		Trigger:      from.Trigger,
		Number:       from.Number,
		Parent:       from.Parent,
		Status:       from.Status,
		Error:        from.Error,
		Event:        from.Event,
		Action:       from.Action,
		Link:         from.Link,
		Timestamp:    from.Timestamp,
		Title:        from.Title,
		Message:      from.Message,
		Before:       from.Before,
		After:        from.After,
		Ref:          from.Ref,
		Fork:         from.Fork,
		Source:       from.Source,
		Target:       from.Target,
		Author:       from.Author,
		AuthorName:   from.AuthorName,
		AuthorEmail:  from.AuthorEmail,
		AuthorAvatar: from.AuthorAvatar,
		Sender:       from.Sender,
		Params:       from.Params,
		Deploy:       from.Deploy,
		Started:      from.Started,
		Finished:     from.Finished,
		Created:      from.Created,
		Updated:      from.Updated,
		Version:      from.Version,
	}
}
