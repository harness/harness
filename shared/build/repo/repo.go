package repo

import (
	"fmt"
	"strings"
)

const (
	Git       = "git"
	Mercurial = "mercurial"
)

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

// IsRemote returns true if the Repository is located
// on a remote server (ie Github, Bitbucket)
func (r *Repo) IsRemote() bool {
	switch {
	case strings.HasPrefix(r.Path, "git://"):
		return true
	case strings.HasPrefix(r.Path, "git@"):
		return true
	case strings.HasPrefix(r.Path, "gitlab@"):
		return true
	case strings.HasPrefix(r.Path, "http://"):
		return true
	case strings.HasPrefix(r.Path, "https://"):
		return true
	case strings.HasPrefix(r.Path, "ssh://"):
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
//
// TODO we should also enable SVN projects
func (r *Repo) Commands() []string {
	// get the branch. default to master or default if no branch exists.
	branch := r.Branch
	if len(branch) == 0 {
		switch {
		case r.Scm == Mercurial:
			branch = "default"
		case r.Scm == Git:
		default:
			branch = "master"
		}
	}

	cmds := []string{}
	switch {
	case r.Scm == Mercurial:
		cmds = append(cmds, fmt.Sprintf("hg clone --branch %s %s %s", branch, r.Path, r.Dir))
		if len(r.Commit) > 0 {
			cmds = append(cmds, fmt.Sprintf("hg update %s", r.Commit))
		}
	case r.Scm == Git:
	default:
		if len(r.PR) > 0 {
			// If a specific PR is provided then we need to clone it.
			cmds = append(cmds, fmt.Sprintf("git clone --depth=%d --recursive %s %s", r.Depth, r.Path, r.Dir))
			cmds = append(cmds, fmt.Sprintf("git fetch origin +refs/pull/%s/head:refs/remotes/origin/pr/%s", r.PR, r.PR))
			cmds = append(cmds, fmt.Sprintf("git checkout -qf -b pr/%s origin/pr/%s", r.PR, r.PR))
		} else {
			// Otherwise just clone the branch.
			cmds = append(cmds, fmt.Sprintf("git clone --depth=%d --recursive --branch=%s %s %s", r.Depth, branch, r.Path, r.Dir))
			// If a specific commit is provided then we'll need to check it out.
			if len(r.Commit) > 0 {
				cmds = append(cmds, fmt.Sprintf("git checkout -qf %s", r.Commit))
			}
		}
	}

	return cmds
}
