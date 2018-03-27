// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"time"

	"github.com/cncd/queue"
	"github.com/dimfeld/httptreemux"
	"github.com/drone/drone/model"
	"github.com/drone/drone/plugins/registry"
	"github.com/drone/drone/plugins/secrets"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/remote/bitbucket"
	"github.com/drone/drone/remote/bitbucketserver"
	"github.com/drone/drone/remote/coding"
	"github.com/drone/drone/remote/gitea"
	"github.com/drone/drone/remote/github"
	"github.com/drone/drone/remote/gitlab"
	"github.com/drone/drone/remote/gitlab3"
	"github.com/drone/drone/remote/gogs"
	"github.com/drone/drone/server/web"
	"github.com/drone/drone/store"
	"github.com/drone/drone/store/datastore"

	"github.com/urfave/cli"
)

func setupStore(c *cli.Context) store.Store {
	return datastore.New(
		c.String("driver"),
		c.String("datasource"),
	)
}

func setupQueue(c *cli.Context, s store.Store) queue.Queue {
	return model.WithTaskStore(queue.New(), s)
}

func setupSecretService(c *cli.Context, s store.Store) model.SecretService {
	return secrets.New(s)
}

func setupRegistryService(c *cli.Context, s store.Store) model.RegistryService {
	return registry.New(s)
}

func setupEnvironService(c *cli.Context, s store.Store) model.EnvironService {
	return nil
}

func setupLimiter(c *cli.Context, s store.Store) model.Limiter {
	return new(model.NoLimit)
}

func setupPubsub(c *cli.Context)        {}
func setupStream(c *cli.Context)        {}
func setupGatingService(c *cli.Context) {}

// helper function to setup the remote from the CLI arguments.
func SetupRemote(c *cli.Context) (remote.Remote, error) {
	switch {
	case c.Bool("github"):
		return setupGithub(c)
	case c.Bool("gitlab"):
		return setupGitlab(c)
	case c.Bool("bitbucket"):
		return setupBitbucket(c)
	case c.Bool("stash"):
		return setupStash(c)
	case c.Bool("gogs"):
		return setupGogs(c)
	case c.Bool("gitea"):
		return setupGitea(c)
	case c.Bool("coding"):
		return setupCoding(c)
	default:
		return nil, fmt.Errorf("version control system not configured")
	}
}

// helper function to setup the Bitbucket remote from the CLI arguments.
func setupBitbucket(c *cli.Context) (remote.Remote, error) {
	return bitbucket.New(
		c.String("bitbucket-client"),
		c.String("bitbucket-secret"),
	), nil
}

// helper function to setup the Gogs remote from the CLI arguments.
func setupGogs(c *cli.Context) (remote.Remote, error) {
	return gogs.New(gogs.Opts{
		URL:         c.String("gogs-server"),
		Username:    c.String("gogs-git-username"),
		Password:    c.String("gogs-git-password"),
		PrivateMode: c.Bool("gogs-private-mode"),
		SkipVerify:  c.Bool("gogs-skip-verify"),
	})
}

// helper function to setup the Gitea remote from the CLI arguments.
func setupGitea(c *cli.Context) (remote.Remote, error) {
	return gitea.New(gitea.Opts{
		URL:         c.String("gitea-server"),
		Username:    c.String("gitea-git-username"),
		Password:    c.String("gitea-git-password"),
		PrivateMode: c.Bool("gitea-private-mode"),
		SkipVerify:  c.Bool("gitea-skip-verify"),
	})
}

// helper function to setup the Stash remote from the CLI arguments.
func setupStash(c *cli.Context) (remote.Remote, error) {
	return bitbucketserver.New(bitbucketserver.Opts{
		URL:               c.String("stash-server"),
		Username:          c.String("stash-git-username"),
		Password:          c.String("stash-git-password"),
		ConsumerKey:       c.String("stash-consumer-key"),
		ConsumerRSA:       c.String("stash-consumer-rsa"),
		ConsumerRSAString: c.String("stash-consumer-rsa-string"),
		SkipVerify:        c.Bool("stash-skip-verify"),
	})
}

// helper function to setup the Gitlab remote from the CLI arguments.
func setupGitlab(c *cli.Context) (remote.Remote, error) {
	if c.Bool("gitlab-v3-api") {
		return gitlab3.New(gitlab3.Opts{
			URL:         c.String("gitlab-server"),
			Client:      c.String("gitlab-client"),
			Secret:      c.String("gitlab-secret"),
			Username:    c.String("gitlab-git-username"),
			Password:    c.String("gitlab-git-password"),
			PrivateMode: c.Bool("gitlab-private-mode"),
			SkipVerify:  c.Bool("gitlab-skip-verify"),
		})
	}
	return gitlab.New(gitlab.Opts{
		URL:         c.String("gitlab-server"),
		Client:      c.String("gitlab-client"),
		Secret:      c.String("gitlab-secret"),
		Username:    c.String("gitlab-git-username"),
		Password:    c.String("gitlab-git-password"),
		PrivateMode: c.Bool("gitlab-private-mode"),
		SkipVerify:  c.Bool("gitlab-skip-verify"),
	})
}

// helper function to setup the GitHub remote from the CLI arguments.
func setupGithub(c *cli.Context) (remote.Remote, error) {
	return github.New(github.Opts{
		URL:         c.String("github-server"),
		Context:     c.String("github-context"),
		Client:      c.String("github-client"),
		Secret:      c.String("github-secret"),
		Scopes:      c.StringSlice("github-scope"),
		Username:    c.String("github-git-username"),
		Password:    c.String("github-git-password"),
		PrivateMode: c.Bool("github-private-mode"),
		SkipVerify:  c.Bool("github-skip-verify"),
		MergeRef:    c.BoolT("github-merge-ref"),
	})
}

// helper function to setup the Coding remote from the CLI arguments.
func setupCoding(c *cli.Context) (remote.Remote, error) {
	return coding.New(coding.Opts{
		URL:        c.String("coding-server"),
		Client:     c.String("coding-client"),
		Secret:     c.String("coding-secret"),
		Scopes:     c.StringSlice("coding-scope"),
		Machine:    c.String("coding-git-machine"),
		Username:   c.String("coding-git-username"),
		Password:   c.String("coding-git-password"),
		SkipVerify: c.Bool("coding-skip-verify"),
	})
}

func setupTree(c *cli.Context) *httptreemux.ContextMux {
	tree := httptreemux.NewContextMux()
	web.New(
		web.WithDir(c.String("www")),
		web.WithSync(time.Hour*72),
	).Register(tree)
	return tree
}

func before(c *cli.Context) error { return nil }
