package common

// Agent represents a worker that has connected
// to the system in order to perform work
type Agent struct {
	Name      string `json:"name"`
	Addr      string `json:"addr"`
	IsHealthy bool   `json:"is_healthy"`
}
