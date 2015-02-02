package repo

type Repo struct {
	// The name of the Repository. This should be the
	// canonical name, for example, github.com/drone/drone.
	Name string

	// The type of source control management.
	Scm string

	// The path of the Repoisotry. This could be
	// the remote path of a Git repository or the path of
	// of the repository on the local file system.
	//
	// A remote path must start with http://, https://,
	// git://, ssh:// or git@. Otherwise we'll assume
	// the repository is located on the local filesystem.
	Path string

	// (optional) Specific Branch that we should checkout
	// when the Repository is cloned. If no value is
	// provided we'll assume the default, master branch.
	Branch string

	// (optional) Specific Commit Hash that we should
	// checkout when the Repository is cloned. If no
	// value is provided we'll assume HEAD.
	Commit string

	// (optional) Pull Request number that we should
	// checkout when the Repository is cloned.
	PR string

	// (optional) The filesystem path that the repository
	// will be cloned into (or copied to) inside the
	// host system (Docker Container).
	Dir string

	// (optional) The depth of the `git clone` command.
	Depth int
}

func (r *Repo) IsRemote() bool {
	if r.Scm == "local" {
		return false
	} else {
		return true
	}
}

func (r *Repo) IsLocal() bool {
	if r.Scm == "local" {
		return true
	} else {
		return false
	}
}
