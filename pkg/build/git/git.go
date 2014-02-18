package git

const (
	DefaultGitDepth = 50
)

// Git stores the configuration details for
// executing Git commands.
type Git struct {
	// Depth options instructs git to create a shallow
	// clone with a history truncated to the specified
	// number of revisions.
	Depth *int `yaml:"depth,omitempty"`

	// The name of a directory to clone into.
	// TODO this still needs to be implemented. this field is
	//      critical for forked Go projects, that need to clone
	//      to a specific repository.
	Path string `yaml:"path,omitempty"`
}

// GitDepth returns GitDefaultDepth
// when Git.Depth is empty.
// GitDepth returns Git.Depth
// when it is not empty.
func GitDepth(g *Git) int {
	if g == nil || g.Depth == nil {
		return DefaultGitDepth
	}
	return *g.Depth
}
