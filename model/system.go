package types

// System provides important information about the Drone
// server to the plugin.
type System struct {
	Version string   `json:"version"`
	Link    string   `json:"link_url"`
	Plugins []string `json:"plugins"`
	Globals []string `json:"globals"`
}

// Workspace defines the build's workspace inside the
// container. This helps the plugin locate the source
// code directory.
type Workspace struct {
	Root string `json:"root"`
	Path string `json:"path"`

	Netrc *Netrc   `json:"netrc"`
	Keys  *Keypair `json:"keys"`
}

type Netrc struct {
	Machine  string `json:"machine"`
	Login    string `json:"login"`
	Password string `json:"user"`
}
