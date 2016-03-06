package git

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	//FilterTags only tags, binary 01
	FilterTags = 1
	//FilterHeads only heads, binary 10
	FilterHeads = 2
	//FilterAll tags and heads, binary 11
	FilterAll = 3
)

var (
	//ErrNoRefs not found refs
	ErrNoRefs = errors.New("no refs")
	//ErrBadURI bad git uri
	ErrBadURI = errors.New("bad uri")
	//ErrBadFilter bad filter, please refers to filters const
	ErrBadFilter = errors.New("bad filter")
	//ErrBadName bad repo name
	ErrBadName = errors.New("bad repo name")
	//ErrBadWorkspace invalid workspace dir
	ErrBadWorkspace = errors.New("bad workspace")
	//ErrInit git init failed
	ErrInit = errors.New("git init failed")
	//ErrBadCmd bad git subcommand
	ErrBadCmd = errors.New("bad git subcommand")
	//ErrRemoteAdd git remote add failed
	ErrRemoteAdd = errors.New("git remote add failed")
	//ErrBadRef invalid ref
	ErrBadRef = errors.New("bad ref")
	//ErrBadFile invalid filepath
	ErrBadFile = errors.New("bad file")
	//ErrShow git show failed
	ErrShow = errors.New("git show failed")
	//ErrBadRev invalid revision
	ErrBadRev = errors.New("bad rev")
)

var (
	sha1Pattern = regexp.MustCompile(`\b[0-9a-f]{40}\b`)
)

//Reference git reference includes ref name and commit
type Reference struct {
	Ref    string
	Commit string
}

//String impl
func (ref *Reference) String() string {
	return fmt.Sprintf("ref: %s, commit: %s", ref.Ref, ref.Commit)
}

//Client git client for executing git commands
type Client struct {
	URI    string
	Branch string
	Path   string
}

//NewClient create new client
func NewClient(uri string, branch string) (*Client, error) {
	if len(uri) == 0 {
		return nil, ErrBadURI
	}

	return &Client{
		URI:    uri,
		Branch: branch,
	}, nil
}

//String impl
func (client *Client) String() string {
	return fmt.Sprintf("uri: %s, branch: %s", client.URI, client.Branch)
}

//InitRepo init empty repo
func (client *Client) InitRepo(workspace string, name string) error {
	if len(name) == 0 {
		return ErrBadName
	}
	if len(workspace) == 0 {
		workspace = "/var/lib/drone/workspace/"
	}
	if !filepath.IsAbs(workspace) {
		return ErrBadWorkspace
	}
	path := filepath.Join(workspace, name)
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}
	if err := gitInit(path); err != nil {
		return err
	}
	gitRemoteAdd(path, client.URI)
	client.Path = path

	return nil
}

func gitRemoteAdd(path string, remote string) error {
	if len(remote) == 0 {
		return ErrBadURI
	}
	cmd, err := gitCmd(path, "remote", "add", "origin", remote)
	if err != nil {
		return err
	}
	err = cmd.Run()
	if err != nil {
		return ErrRemoteAdd
	}
	return nil
}

func gitInit(path string) error {
	if _, err := os.Stat(filepath.Join(path, ".git")); os.IsNotExist(err) {
		cmd, err := gitCmd(path, "init")
		if err != nil {
			return err
		}
		err = cmd.Run()
		if err != nil {
			return ErrInit
		}
	} else {
		log.Println("existing .git")
	}
	return nil
}

func gitCmd(path string, args ...string) (*exec.Cmd, error) {
	if len(path) == 0 {
		return nil, ErrBadWorkspace
	}
	if len(args) < 1 {
		return nil, ErrBadCmd
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	trace(cmd)

	return cmd, nil
}

//FetchRef fetch a specified ref frome remote repo
func (client *Client) FetchRef(ref string) error {
	if len(ref) == 0 {
		return ErrBadRef
	}
	cmd, err := gitCmd(client.Path, "fetch", "--depth=1", "origin", ref)
	if err != nil {
		return err
	}
	err = cmd.Run()
	if err != nil {
		return err
	}
	cmd, err = gitCmd(client.Path, "reset", "--hard", "FETCH_HEAD")
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

//ShowFile get file by git show
func (client *Client) ShowFile(rev string, file string) ([]byte, error) {
	if len(rev) == 0 {
		return nil, ErrBadRev
	}
	if len(file) == 0 {
		return nil, ErrBadFile
	}
	cmd, err := gitCmd(client.Path, "show", fmt.Sprintf("%s:%s", rev, file))
	if err != nil {
		return nil, err
	}
	cmd.Stdout = nil
	out, err := cmd.Output()
	if err != nil {
		return nil, ErrShow
	}
	return out, nil
}

//LsRemote git ls-remote
func (client *Client) LsRemote(filter uint8, refs string) (push, tag *Reference, err error) {
	if filter < FilterTags || filter > FilterAll {
		return nil, nil, ErrBadFilter
	}

	log.Println("LsRemote filter", filter, "refs", refs)

	if filter&FilterTags > 0 {
		tag, err = lsRemoteRefs(client.URI, "-t", refs)
		if err != nil && err != ErrNoRefs {
			return nil, nil, err
		}
	}

	if filter&FilterHeads > 0 {
		if len(refs) == 0 {
			if len(client.Branch) != 0 {
				refs = client.Branch
			} else {
				refs = "master"
			}
		}
		push, err = lsRemoteRefs(client.URI, "-h", refs)
		if err != nil && err != ErrNoRefs {
			return nil, nil, err
		}
	}

	if push == nil && tag == nil {
		return nil, nil, ErrNoRefs
	}

	return push, tag, nil
}

func lsRemoteRefs(uri string, filter string, refs string) (*Reference, error) {
	args := []string{
		"ls-remote",
	}
	if len(filter) > 0 {
		args = append(args, filter)
	}
	args = append(args, uri)
	if len(refs) > 0 {
		args = append(args, refs)
	}
	log.Printf("ls-remote command %q\n", args)

	cmd := exec.Command(
		"git",
		args...,
	)
	cmd.Dir = os.TempDir()

	out, err := cmd.Output()
	if err != nil {
		log.Println("ls-remote failed:", err.Error())
		return nil, err
	}

	log.Printf("ls-remote: %s\n", out)
	if len(out) == 0 {
		return nil, ErrNoRefs
	}

	reference, commit, err := parseLsRemote(string(out))
	if err != nil {
		return nil, err
	}

	return &Reference{string(reference), string(commit)}, nil
}

func parseLsRemote(text string) ([]byte, []byte, error) {
	lines := strings.Split(string(text), "\n")
	log.Printf("lines %q\n", lines)

	entries := map[string]string{}
	references := []string{}
	if len(lines) > 0 {
		for _, line := range lines {
			tokens := strings.Split(line, "\t")
			log.Printf("tokens %q\n", tokens)

			if len(tokens) == 2 && sha1Pattern.Match([]byte(tokens[0])) {
				reference := strings.TrimSuffix(tokens[1], "^{}")
				entries[reference] = tokens[0]
				references = append(references, reference)
			}
		}
	}

	log.Printf("entries %q\n", entries)
	log.Printf("references %q\n", references)
	if len(references) > 0 {
		sort.Strings(references)
		latestRef := references[len(references)-1]
		latestCommit := entries[latestRef]

		return []byte(latestRef), []byte(latestCommit), nil
	}

	return nil, nil, ErrNoRefs
}

// Trace writes each command to standard error (preceded by a ‘$ ’) before it
// is executed. Used for debugging your build.
func trace(cmd *exec.Cmd) {
	log.Println("$", cmd.Dir, strings.Join(cmd.Args, " "))
}
