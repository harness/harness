package git

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
)

const (
	//FilterTags only tags, binary 01
	FilterTags = 1
	//FilterHeads only heads, binary 10
	FilterHeads = 2
	//FilterAll tags and heads, binary 11
	FilterAll = 3
)

const (
	//GitSshWrapper GIT_SSH
	GitSshWrapper = "git_ssh_wrapper"
	//GitSshWrapperScript GIT_SSH script
	GitSshWrapperScript = `#!/bin/sh

ssh -F %s -i %s $@`
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
	//ErrBadCommit invalid commit
	ErrBadCommit = errors.New("bad commit")
	//ErrBadKey bad private key
	ErrBadKey = errors.New("bad key")
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
	Key    string
}

//NewClient create new client
func NewClient(workspace, dir, uri, branch, key string) (*Client, error) {
	if len(uri) == 0 {
		return nil, ErrBadURI
	}

	client := &Client{
		URI:    uri,
		Branch: branch,
		Key:    key,
	}

	if err := client.initRepo(workspace, dir); err != nil {
		return nil, err
	}

	if err := client.initPrivateKey(); err != nil {
		return nil, err
	}

	return client, nil
}

//String impl
func (client *Client) String() string {
	return fmt.Sprintf("uri: %s, branch: %s", client.URI, client.Branch)
}

//initRepo init empty repo
func (client *Client) initRepo(workspace string, name string) error {
	if len(name) == 0 {
		return ErrBadName
	}
	if len(workspace) == 0 {
		workspace = "/var/lib/drone/workspace/"
	}
	if !filepath.IsAbs(workspace) {
		log.Errorln("bad workspace", workspace)
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
		log.Infoln("existing .git")
	}
	return nil
}

func gitCmd(path string, args ...string) (*exec.Cmd, error) {
	if len(path) == 0 {
		log.Errorln("bad workspace", path)
		return nil, ErrBadWorkspace
	}
	if len(args) < 1 {
		return nil, ErrBadCmd
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "GIT_SSH="+filepath.Join(sshPath(path), GitSshWrapper))
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

//ShowTimestamps show commits timestamps
func (client *Client) ShowTimestamps(commits ...string) ([]int, error) {
	if len(commits) < 1 {
		return nil, ErrBadCommit
	}

	for _, commit := range commits {
		if len(commit) != 40 {
			return nil, ErrBadCommit
		}
	}

	cmd, err := gitCmd(client.Path, append([]string{"show", "-s", "--format=%ct"}, commits...)...)
	if err != nil {
		return nil, err
	}

	cmd.Stdout = nil
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseTimestamps(commits, out)
}

func parseTimestamps(commits []string, out []byte) ([]int, error) {
	if len(out) < 10 {
		return nil, ErrShow
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	timestamps := []int{}
	for scanner.Scan() {
		lineStr := scanner.Text()
		timestamp, err := strconv.Atoi(lineStr)
		if err != nil {
			return nil, err
		}
		timestamps = append(timestamps, timestamp)
	}
	if len(commits) != len(timestamps) {
		log.Errorf("timstamps err %q -> %q", commits, timestamps)
		return nil, ErrShow
	}
	return nil, ErrShow
}

//LsRemote git ls-remote
func (client *Client) LsRemote(filter uint8, refs string) (push, tag *Reference, err error) {
	if filter < FilterTags || filter > FilterAll {
		return nil, nil, ErrBadFilter
	}

	log.Debugln("LsRemote filter", filter, "refs", refs)
	if filter&FilterTags > 0 {
		tag, err = client.lsRemoteRefs("-t", refs)
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
		push, err = client.lsRemoteRefs("-h", refs)
		if err != nil && err != ErrNoRefs {
			return nil, nil, err
		}
	}

	if push == nil && tag == nil {
		return nil, nil, ErrNoRefs
	}

	return push, tag, nil
}

func (client *Client) lsRemoteRefs(filter string, refs string) (*Reference, error) {
	args := []string{
		"ls-remote",
	}
	if len(filter) > 0 {
		args = append(args, filter)
	}
	args = append(args, client.URI)
	if len(refs) > 0 {
		args = append(args, refs)
	}
	log.Debugf("ls-remote command %q\n", args)

	cmd, err := gitCmd(client.Path, args...)
	if err != nil {
		return nil, err
	}
	cmd.Stdout = nil
	out, err := cmd.Output()
	if err != nil {
		log.Debugln("ls-remote failed:", err.Error())
		return nil, err
	}

	log.Debugln("ls-remote: %s\n", out)
	if len(out) == 0 {
		return nil, ErrNoRefs
	}

	reference, commit, err := parseLsRemote(out)
	if err != nil {
		return nil, err
	}

	return &Reference{string(reference), string(commit)}, nil
}

func parseLsRemote(text []byte) ([]byte, []byte, error) {
	if len(text) < 1 {
		return nil, nil, ErrNoRefs
	}

	entries := map[string]string{}
	references := []string{}
	scanner := bufio.NewScanner(bytes.NewReader(text))
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Split(line, "\t")
		log.Debugf("tokens %q\n", tokens)

		if len(tokens) == 2 && sha1Pattern.Match([]byte(tokens[0])) {
			reference := strings.TrimSuffix(tokens[1], "^{}")
			entries[reference] = tokens[0]
			references = append(references, reference)
		}
	}

	log.Debugf("entries %q\n", entries)
	log.Debugf("references %q\n", references)
	if len(references) > 0 {
		sort.Strings(references)
		latestRef := references[len(references)-1]
		latestCommit := entries[latestRef]

		return []byte(latestRef), []byte(latestCommit), nil
	}

	return nil, nil, ErrNoRefs
}

//initPrivateKey write private key and set GIT_SSH
func (client *Client) initPrivateKey() error {
	if len(client.Key) == 0 {
		return ErrBadKey
	}

	sshpath := sshPath(client.Path)
	if err := os.MkdirAll(sshpath, 0700); err != nil {
		return err
	}
	confpath := filepath.Join(sshpath, "config")
	privpath := filepath.Join(sshpath, "id_rsa")
	wrapperpath := filepath.Join(sshpath, GitSshWrapper)
	log.Debugln("conf", confpath, "private", privpath, "wrapper", wrapperpath)

	err := ioutil.WriteFile(confpath, []byte("StrictHostKeyChecking no\n"), 0700)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(privpath, []byte(client.Key), 0600)
	if err != nil {
		return err
	}

	wrapperScript := fmt.Sprintf(GitSshWrapperScript, confpath, privpath)
	err = ioutil.WriteFile(wrapperpath, []byte(wrapperScript), 0755)
	if err != nil {
		return err
	}

	return nil
}

func sshPath(path string) string {
	return filepath.Join(path, ".ssh")
}

// Trace writes each command to standard error (preceded by a ‘$ ’) before it
// is executed. Used for debugging your build.
func trace(cmd *exec.Cmd) {
	log.Debugln("$env", strings.Join(cmd.Env, " "))
	log.Infoln("$", cmd.Dir, strings.Join(cmd.Args, " "))
}
