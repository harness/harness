package common

// Agent represents a worker that has connected
// to the system in order to perform work
type Agent struct {
	Name      string
	Addr      string
	IsHealthy bool
}
