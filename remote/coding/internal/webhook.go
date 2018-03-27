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
	"net/url"
)

type Webhook struct {
	Id      int    `json:"id"`
	HookURL string `json:"hook_url"`
}

func (c *Client) GetWebhooks(globalKey, projectName string) ([]*Webhook, error) {
	u := fmt.Sprintf("/user/%s/project/%s/git/hooks", globalKey, projectName)
	resp, err := c.Get(u, nil)
	if err != nil {
		return nil, err
	}
	webhooks := make([]*Webhook, 0)
	err = json.Unmarshal(resp, &webhooks)
	if err != nil {
		return nil, APIClientErr{"fail to parse webhooks data", u, err}
	}
	return webhooks, nil
}

func (c *Client) AddWebhook(globalKey, projectName, link string) error {
	webhooks, err := c.GetWebhooks(globalKey, projectName)
	if err != nil {
		return err
	}
	webhook := matchingHooks(webhooks, link)
	if webhook != nil {
		u := fmt.Sprintf("/user/%s/project/%s/git/hook/%d", globalKey, projectName, webhook.Id)
		params := url.Values{}
		params.Set("hook_url", link)
		params.Set("type_pust", "true")
		params.Set("type_mr_pr", "true")

		_, err := c.Do("PUT", u, params)
		if err != nil {
			return APIClientErr{"fail to edit webhook", u, err}
		}
		return nil
	}

	u := fmt.Sprintf("/user/%s/project/%s/git/hook", globalKey, projectName)
	params := url.Values{}
	params.Set("hook_url", link)
	params.Set("type_push", "true")
	params.Set("type_mr_pr", "true")

	_, err = c.Do("POST", u, params)
	if err != nil {
		return APIClientErr{"fail to add webhook", u, err}
	}
	return nil
}

func (c *Client) RemoveWebhook(globalKey, projectName, link string) error {
	webhooks, err := c.GetWebhooks(globalKey, projectName)
	if err != nil {
		return err
	}
	webhook := matchingHooks(webhooks, link)
	if webhook == nil {
		return nil
	}

	u := fmt.Sprintf("/user/%s/project/%s/git/hook/%d", globalKey, projectName, webhook.Id)
	_, err = c.Do("DELETE", u, nil)
	if err != nil {
		return APIClientErr{"fail to remove webhook", u, err}
	}
	return nil
}

// helper function to return matching hook.
func matchingHooks(hooks []*Webhook, rawurl string) *Webhook {
	link, err := url.Parse(rawurl)
	if err != nil {
		return nil
	}
	for _, hook := range hooks {
		hookurl, err := url.Parse(hook.HookURL)
		if err == nil && hookurl.Host == link.Host {
			return hook
		}
	}
	return nil
}
