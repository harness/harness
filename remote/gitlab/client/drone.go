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

package client

const (
	droneServiceUrl = "/projects/:id/services/drone-ci"
)

func (c *Client) AddDroneService(id string, params QMap) error {
	url, opaque := c.ResourceUrl(
		droneServiceUrl,
		QMap{":id": id},
		params,
	)

	_, err := c.Do("PUT", url, opaque, nil)
	return err
}

func (c *Client) DeleteDroneService(id string) error {
	url, opaque := c.ResourceUrl(
		droneServiceUrl,
		QMap{":id": id},
		nil,
	)

	_, err := c.Do("DELETE", url, opaque, nil)
	return err
}
