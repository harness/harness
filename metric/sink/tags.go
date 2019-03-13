// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package sink

import "fmt"

func createTags(config Config) []string {
	var tags []string
	switch {
	case config.EnableBitbucket:
		tags = append(tags, "remote:bitbucket:cloud")
	case config.EnableStash:
		tags = append(tags, "remote:bitbucket:server")
	case config.EnableGithubEnt:
		tags = append(tags, "remote:github:enterprise")
	case config.EnableGithub:
		tags = append(tags, "remote:github:cloud")
	case config.EnableGitlab:
		tags = append(tags, "remote:gitlab")
	case config.EnableGogs:
		tags = append(tags, "remote:gogs")
	case config.EnableGitea:
		tags = append(tags, "remote:gitea")
	default:
		tags = append(tags, "remote:undefined")
	}

	switch {
	case config.EnableAgents:
		tags = append(tags, "scheduler:internal:agents")
	case config.EnableKubernetes:
		tags = append(tags, "scheduler:kubernetes")
	case config.EnableGithub:
		tags = append(tags, "scheduler:nomad")
	default:
		tags = append(tags, "scheduler:internal:local")
	}

	if config.Subscription != "" {
		tag := fmt.Sprintf("license:%s:%s:%s",
			config.License,
			config.Licensor,
			config.Subscription,
		)
		tags = append(tags, tag)
	} else if config.Licensor != "" {
		tag := fmt.Sprintf("license:%s:%s",
			config.License,
			config.Licensor,
		)
		tags = append(tags, tag)
	} else {
		tag := fmt.Sprintf("license:%s", config.License)
		tags = append(tags, tag)
	}
	return tags
}
