package gogitlab

const (
	drone_service_url = "/projects/:id/services/drone-ci"
)

func (g *Gitlab) AddDroneService(id string, params map[string]string) error {
	url, opaque := g.ResourceUrlQueryRaw(drone_service_url, map[string]string{":id": id}, params)

	_, err := g.buildAndExecRequestRaw("PUT", url, opaque, nil)
	return err
}

func (g *Gitlab) DeleteDroneService(id string) error {
	url, opaque := g.ResourceUrlQueryRaw(drone_service_url, map[string]string{":id": id}, nil)

	_, err := g.buildAndExecRequestRaw("DELETE", url, opaque, nil)
	return err
}
