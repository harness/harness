package types

type System struct {
	URL     string            // System URL
	Env     map[string]string // Global environment variables
	Builder string            // Name of build container (default drone/drone-build)
	Plugins string            // Name of approved plugin containers (default plugins/*)
}
