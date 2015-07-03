package git

const (
	DefaultGitDepth     = 50
	DefaultGitRecursive = true
)

// Git stores the configuration details for
// executing Git commands.
type Git struct {
	// Depth options instructs git to create a shallow
	// clone with a history truncated to the specified
	// number of revisions.
	Depth *int `yaml:"depth,omitempty"`

	// Recursive option instructs git to do a recursive
	// clone of all submodules.
	Recursive *bool `yaml:"recursive,omitempty"`

	// The name of a directory to clone into.
	Path *string `yaml:"path,omitempty"`
}

// GitDepth returns DefaultGitDepth
// when Git.Depth is empty.
// GitDepth returns Git.Depth
// when it is not empty.
func GitDepth(g *Git) int {
	if g == nil || g.Depth == nil {
		return DefaultGitDepth
	}
	return *g.Depth
}

// GitRecursive returns DefaultGitRecursive
// when Git.Recursive is empty.
// GitRecursive returns Git.Recursive
// when it is not empty.
func GitRecursive(g *Git) bool {
	if g == nil || g.Recursive == nil {
		return DefaultGitRecursive
	}
	return *g.Recursive
}

// GitPath returns the given default path
// when Git.Path is empty.
// GitPath returns Git.Path
// when it is not empty.
func GitPath(g *Git, defaultPath string) string {
	if g == nil || g.Path == nil {
		return defaultPath
	}
	return *g.Path
}
