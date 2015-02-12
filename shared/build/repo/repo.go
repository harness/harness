package repo

import (
	"github.com/drone/drone/plugin/remote"
	"github.com/drone/drone/shared/model"
)

type Repo struct {
	// The name of the Repository. This should be the
	// canonical name, for example, github.com/drone/drone.
	Name string

	// Local path of the Repository
	Path string

	// Remote repository
	Repo *model.Repo

	// Commit to build, empty if local repository
	Commit *model.Commit

	// (optional) The filesystem path that the repository
	// will be cloned into (or copied to) inside the
	// host system (Docker Container).
	Dir string

	// (optional) The depth of the `git clone` command.
	Depth int
}

// IsRemote returns true if the Repository is located
// on a remote server (ie Github, Bitbucket)
func (r *Repo) IsRemote() bool {
	if r.Repo != nil {
		return true
	}
	return false
}

// IsLocal returns true if the Repository is located
// on the local filesystem.
func (r *Repo) IsLocal() bool {
	return !r.IsRemote()
}

// returns commands that can be used in a Dockerfile
// to clone the repository.
func (r *Repo) Commands() []string {
	var remote = remote.Lookup(r.Repo.Remote)
	if remote == nil {
		return nil
	}

	cmds := remote.Commands(r.Repo, r.Commit, r.Dir, r.Depth)

	return cmds
}
