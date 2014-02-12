package git

const (
	DefaultGitDepth = 50
)

// Git stores the configuration details for
// executing Git commands.
type Git struct {
	Depth *int `yaml:"depth,omitempty"`
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
