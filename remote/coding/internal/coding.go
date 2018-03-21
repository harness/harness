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

package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	baseURL string
	apiPath string
	token   string
	agent   string
	client  *http.Client
}

type GenericAPIResponse struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data,omitempty"`
}

func NewClient(baseURL, apiPath, token, agent string, client *http.Client) *Client {
	return &Client{
		baseURL: baseURL,
		apiPath: apiPath,
		token:   token,
		agent:   agent,
		client:  client,
	}
}

// Generic GET for requesting Coding OAuth API
func (c *Client) Get(u string, params url.Values) ([]byte, error) {
	return c.Do(http.MethodGet, u, params)
}

// Generic method for requesting Coding OAuth API
func (c *Client) Do(method, u string, params url.Values) ([]byte, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("access_token", c.token)

	rawURL := c.baseURL + c.apiPath + u

	var req *http.Request
	var err error
	if method != "GET" {
		req, err = http.NewRequest(method, rawURL+"?access_token="+c.token, strings.NewReader(params.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	} else {
		req, err = http.NewRequest("GET", rawURL+"?"+params.Encode(), nil)
	}
	if err != nil {
		return nil, fmt.Errorf("fail to create request for url %q: %v", rawURL, err)
	}
	req.Header.Set("User-Agent", c.agent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fail to request %s %s: %v", req.Method, req.URL, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s %s respond %d", req.Method, req.URL, resp.StatusCode)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fail to read response from %s %s: %v", req.Method, req.URL.String(), err)
	}

	apiResp := &GenericAPIResponse{}
	err = json.Unmarshal(body, apiResp)
	if err != nil {
		return nil, fmt.Errorf("fail to parse response from %s %s: %v", req.Method, req.URL.String(), err)
	}
	if apiResp.Code != 0 {
		return nil, fmt.Errorf("Coding OAuth API respond error: %s", string(body))
	}
	return apiResp.Data, nil
}
