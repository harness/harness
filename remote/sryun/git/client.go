package git

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
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
