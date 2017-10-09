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
