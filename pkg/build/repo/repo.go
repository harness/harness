package repo

import (
	"fmt"
	"strings"
)

type Repo struct {
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
}

// IsRemote returns true if the Repository is located
// on a remote server (ie Github, Bitbucket)
func (r *Repo) IsRemote() bool {
	switch {
	case strings.HasPrefix(r.Path, "git://"):
		return true
	case strings.HasPrefix(r.Path, "git@"):
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

// IsGit returns true if the Repository is
// a Git repoisitory.
func (r *Repo) IsGit() bool {
	switch {
	case strings.HasPrefix(r.Path, "git://"):
		return true
	case strings.HasPrefix(r.Path, "git@"):
		return true
	case strings.HasPrefix(r.Path, "ssh://git@"):
		return true
	case strings.HasPrefix(r.Path, "https://github.com/"):
		return true
	case strings.HasPrefix(r.Path, "http://github.com"):
		return true
	case strings.HasSuffix(r.Path, ".git"):
		return true
	}

	// we could also ping the repository to check

	return false
}

// returns commands that can be used in a Dockerfile
// to clone the repository.
//
// TODO we should also enable Mercurial projects and SVN projects
func (r *Repo) Commands() []string {

	// get the branch. default to master
	// if no branch exists.
	branch := r.Branch
	if len(branch) == 0 {
		branch = "master"
	}

	cmds := []string{}
	cmds = append(cmds, fmt.Sprintf("git clone --recursive --branch=%s %s %s", branch, r.Path, r.Dir))

	switch {
	// if a specific commit is provided then we'll
	// need to clone it.
	case len(r.PR) > 0:

		cmds = append(cmds, fmt.Sprintf("git fetch origin +refs/pull/%s/head:refs/remotes/origin/pr/%s", r.PR, r.PR))
		cmds = append(cmds, fmt.Sprintf("git checkout -qf pr/%s", r.PR))
		//cmds = append(cmds, fmt.Sprintf("git fetch origin +refs/pull/%s/merge:", r.PR))
		//cmds = append(cmds, fmt.Sprintf("git checkout -qf %s", "FETCH_HEAD"))
	// if a specific commit is provided then we'll
	// need to clone it.
	case len(r.Commit) > 0:
		cmds = append(cmds, fmt.Sprintf("git checkout -qf %s", r.Commit))
	}

	return cmds
}
