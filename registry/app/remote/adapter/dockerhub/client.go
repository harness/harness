// Source: https://github.com/goharbor/harbor

// Copyright 2016 Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dockerhub

import (
	"context"
	"fmt"
	"io"
	"net/http"

	commonhttp "github.com/harness/gitness/registry/app/common/http"
	"github.com/harness/gitness/registry/types"

	"github.com/rs/zerolog/log"
)

// Client is a client to interact with DockerHub.
type Client struct {
	client *http.Client
	token  string
	host   string
	// credential LoginCredential
}

// NewClient creates a new DockerHub client.
func NewClient(_ types.UpstreamProxy) (*Client, error) {
	client := &Client{
		host: registryURL,
		client: &http.Client{
			Transport: commonhttp.GetHTTPTransport(commonhttp.WithInsecure(true)),
		},
	}

	return client, nil
}

// Do performs http request to DockerHub, it will set token automatically.
func (c *Client) Do(method, path string, body io.Reader) (*http.Response, error) {
	url := baseURL + path
	log.Info().Msgf("%s %s", method, url)
	req, err := http.NewRequestWithContext(context.TODO(), method, url, body)
	if err != nil {
		return nil, err
	}
	if body != nil || method == http.MethodPost || method == http.MethodPut {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", "Bearer", c.token))
	return c.client.Do(req)
}
