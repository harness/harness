package types

// Agent represents a worker that has connected
// to the system in order to perform work
type Agent struct {
	ID        int64  `meddler:"agent_id,pk"    json:"id,omitempty"`
	Kind      string `meddler:"agent_kind"     json:"kind,omitempty"`
	Addr      string `meddler:"agent_addr"     json:"address"`
	Token     string `meddler:"agent_token"    json:"token"`
	Cert      string `meddler:"agent_cert"     json:"-"`
	Key       string `meddler:"agent_key"      json:"-"`
	Active    bool   `meddler:"agent_active"   json:"is_active"`
	IsHealthy bool   `meddler:"-"              json:"is_healthy,omitempty"`
}
