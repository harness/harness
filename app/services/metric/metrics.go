// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metric

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/version"
)

const jobType = "metric-collector"

type metricData struct {
	IP         string `json:"ip"`
	Hostname   string `json:"hostname"`
	Installer  string `json:"installed_by"`
	Installed  string `json:"installed_at"`
	Version    string `json:"version"`
	Users      int64  `json:"user_count"`
	Repos      int64  `json:"repo_count"`
	Pipelines  int64  `json:"pipeline_count"`
	Executions int64  `json:"execution_count"`
	Gitspaces  int64  `json:"gitspace_count"`
}

type Collector struct {
	hostname            string
	enabled             bool
	endpoint            string
	token               string
	userStore           store.PrincipalStore
	repoStore           store.RepoStore
	pipelineStore       store.PipelineStore
	executionStore      store.ExecutionStore
	scheduler           *job.Scheduler
	gitspaceConfigStore store.GitspaceConfigStore
}

func (c *Collector) Register(ctx context.Context) error {
	if !c.enabled {
		return nil
	}
	err := c.scheduler.AddRecurring(ctx, jobType, jobType, "0 0 * * *", time.Minute)
	if err != nil {
		return fmt.Errorf("failed to register recurring job for collector: %w", err)
	}

	return nil
}

func (c *Collector) Handle(ctx context.Context, _ string, _ job.ProgressReporter) (string, error) {
	if !c.enabled {
		return "", nil
	}

	// get first available user
	users, err := c.userStore.ListUsers(ctx, &types.UserFilter{
		Page: 1,
		Size: 1,
	})
	if err != nil {
		return "", err
	}
	if len(users) == 0 {
		return "", nil
	}

	// total users in the system
	totalUsers, err := c.userStore.CountUsers(ctx, &types.UserFilter{})
	if err != nil {
		return "", fmt.Errorf("failed to get users total count: %w", err)
	}

	// total repos in the system
	totalRepos, err := c.repoStore.Count(ctx, 0, &types.RepoFilter{})
	if err != nil {
		return "", fmt.Errorf("failed to get repositories total count: %w", err)
	}

	// total pipelines in the system
	totalPipelines, err := c.pipelineStore.Count(ctx, 0, types.ListQueryFilter{})
	if err != nil {
		return "", fmt.Errorf("failed to get pipelines total count: %w", err)
	}

	// total executions in the system
	totalExecutions, err := c.executionStore.Count(ctx, 0)
	if err != nil {
		return "", fmt.Errorf("failed to get executions total count: %w", err)
	}

	// total gitspaces (configs) in the system
	totalGitspaces, err := c.gitspaceConfigStore.Count(ctx, &types.GitspaceFilter{IncludeDeleted: true})
	if err != nil {
		return "", fmt.Errorf("failed to get gitspace total count: %w", err)
	}

	data := metricData{
		Hostname:   c.hostname,
		Installer:  users[0].Email,
		Installed:  time.UnixMilli(users[0].Created).Format("2006-01-02 15:04:05"),
		Version:    version.Version.String(),
		Users:      totalUsers,
		Repos:      totalRepos,
		Pipelines:  totalPipelines,
		Executions: totalExecutions,
		Gitspaces:  totalGitspaces,
	}

	buf := new(bytes.Buffer)
	err = json.NewEncoder(buf).Encode(data)
	if err != nil {
		return "", fmt.Errorf("failed to encode metric data: %w", err)
	}

	endpoint := fmt.Sprintf("%s?api_key=%s", c.endpoint, c.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, buf)
	if err != nil {
		return "", fmt.Errorf("failed to create a request for metric data to endpoint %s: %w", endpoint, err)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send metric data to endpoint %s: %w", endpoint, err)
	}

	res.Body.Close()

	return res.Status, nil
}

// httpClient should be used for HTTP requests. It
// is configured with a timeout for reliability.
var httpClient = &http.Client{
	Transport: &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		TLSHandshakeTimeout: 30 * time.Second,
		DisableKeepAlives:   true,
	},
	Timeout: 1 * time.Minute,
}
