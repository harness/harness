package svn

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Client struct {
	URI    string
	Branch string
	Path   string
	Key    string
	User   string
	Passwd string
}

const (
	//SvnSshWrapper SVN_SSH
	SvnSshWrapper = "git_ssh_wrapper"
	//GitSshWrapperScript SVN_SSH script
	SvnSshWrapperScript = `#!/bin/sh

					ssh -F %s -i %s $@`
)

var (

	//ErrBadURI bad svn uri
	ErrBadURI = errors.New("bad uri")
	//ErrBadName bad repo name
	ErrBadName = errors.New("bad repo name")
	//ErrBadWorkspace invalid workspace dir
	ErrBadWorkspace = errors.New("bad workspace")
	//ErrBadCmd bad svn subcommand
	ErrBadCmd = errors.New("bad svn subcommand")
	//ErrShow svn show failed
	ErrShow = errors.New("svn info failed")
	//ErrBadFile invalid filepath
	ErrBadFile = errors.New("bad file")
	//ErrBadRev invalid revision
	ErrBadRev = errors.New("bad rev")
	//ErrBadKey bad private key
	ErrBadKey = errors.New("bad key")
)

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

	client.Path = path

	return nil
}

//initPrivateKey write private key and set SVN_SSH
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
	wrapperpath := filepath.Join(sshpath, SvnSshWrapper)
	log.Debugln("conf", confpath, "private", privpath, "wrapper", wrapperpath)

	err := ioutil.WriteFile(confpath, []byte("StrictHostKeyChecking no\n"), 0700)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(privpath, []byte(client.Key), 0600)
	if err != nil {
		return err
	}

	wrapperScript := fmt.Sprintf(SvnSshWrapperScript, confpath, privpath)
	err = ioutil.WriteFile(wrapperpath, []byte(wrapperScript), 0755)
	if err != nil {
		return err
	}

	return nil
}

//ShowFile get file by svn cat
func (client *Client) ShowFile(rev string, file string) ([]byte, error) {
	if len(rev) == 0 {
		return nil, ErrBadRev
	}
	if len(file) == 0 {
		return nil, ErrBadFile
	}
	cmd, err := svnCmd(client.Path, "cat", "--revision", rev, fmt.Sprintf("%s/%s/%s", client.URI, client.Branch, file))
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

func (client *Client) RemoteVersion() (string, error) {

	cmd, err := svnCmd(client.Path, "info", fmt.Sprintf("%s/%s", client.URI, client.Branch))
	if err != nil {
		return "", err
	}

	cmd.Stdout = nil
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return parseChangedVersion(out)
}

func parseChangedVersion(out []byte) (string, error) {

	if len(out) < 10 {
		return "", ErrShow
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))

	for scanner.Scan() {
		lineStr := scanner.Text()
		lineStrs := strings.Split(lineStr, ":")
		if len(lineStrs) == 2 && lineStrs[0] == "Last Changed Rev" {
			return strings.TrimSpace(lineStrs[1]), nil
		}
	}

	return "", ErrShow
}

func svnCmd(path string, args ...string) (*exec.Cmd, error) {
	if len(path) == 0 {
		log.Errorln("bad workspace", path)
		return nil, ErrBadWorkspace
	}
	if len(args) < 1 {
		return nil, ErrBadCmd
	}

	cmd := exec.Command("svn", args...)
	cmd.Dir = path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	//	svnSSH := fmt.Sprintf("ssh -i %s", filepath.Join(sshPath(path), "id_rsa"))

	//	cmd.Env = append(os.Environ(), "SVN_SSH="+svnSSH)
	cmd.Env = append(os.Environ(), "SVN_SSH="+filepath.Join(sshPath(path), SvnSshWrapper))
	trace(cmd)

	return cmd, nil
}

// Trace writes each command to standard error (preceded by a ‘$ ’) before it
// is executed. Used for debugging your build.
func trace(cmd *exec.Cmd) {
	log.Debugln("$env", strings.Join(cmd.Env, " "))
	log.Infoln("$", cmd.Dir, strings.Join(cmd.Args, " "))
}

func sshPath(path string) string {
	return filepath.Join(path, ".ssh")
}
