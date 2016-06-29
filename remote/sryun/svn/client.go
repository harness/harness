package svn

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
)

type Client struct {
	URI    string
	Branch string
	Path   string
	User   string
	Passwd string
}

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
)

func NewClient(workspace, dir, uri, branch string) (*Client, error) {

	if len(uri) == 0 {
		return nil, ErrBadURI
	}
	client := &Client{
		URI:    uri,
		Branch: branch,
	}

	if err := client.initRepo(workspace, dir); err != nil {
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

//ShowFile get file by svn cat
func (client *Client) ShowFile(version string, file string) ([]byte, error) {
	if len(rev) == 0 {
		return nil, ErrBadRev
	}
	if len(file) == 0 {
		return nil, ErrBadFile
	}
	cmd, err := svnCmd(client.Path, "cat", "--revision", version, fmt.Sprintf("%s/%s/%s", client.URI, client.Branch, file))
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
	trace(cmd)

	return cmd, nil
}

// Trace writes each command to standard error (preceded by a ‘$ ’) before it
// is executed. Used for debugging your build.
func trace(cmd *exec.Cmd) {
	log.Debugln("$env", strings.Join(cmd.Env, " "))
	log.Infoln("$", cmd.Dir, strings.Join(cmd.Args, " "))
}
