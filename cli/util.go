package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func parseRepo(str string) (host, owner, repo string) {
	var parts = strings.Split(str, "/")
	if len(parts) != 3 {
		return
	}
	host = parts[0]
	owner = parts[1]
	repo = parts[2]
	return
}

// getGoPath checks the source codes absolute path
// in reference to the host operating system's GOPATH
// to correctly determine the code's package path. This
// is Go-specific, since Go code must exist in
// $GOPATH/src/github.com/{owner}/{name}
func getGoPath(dir string) (string, bool) {
	path := os.Getenv("GOPATH")
	if len(path) == 0 {
		return "", false
	}
	// append src to the GOPATH, since
	// the code will be stored in the src dir
	path = filepath.Join(path, "src")
	if !filepath.HasPrefix(dir, path) {
		return "", false
	}

	// remove the prefix from the directory
	// this should leave us with the go package name
	return dir[len(path):], true
}

var gopathExp = regexp.MustCompile("./src/(github.com/[^/]+/[^/]+|bitbucket.org/[^/]+/[^/]+|code.google.com/[^/]+/[^/]+)")

// getRepoPath checks the source codes absolute path
// on the host operating system in an attempt
// to correctly determine the code's package path. This
// is Go-specific, since Go code must exist in
// $GOPATH/src/github.com/{owner}/{name}
func getRepoPath(dir string) (path string, ok bool) {
	// let's get the package directory based
	// on the path in the host OS
	indexes := gopathExp.FindStringIndex(dir)
	if len(indexes) == 0 {
		return
	}

	index := indexes[len(indexes)-1]

	// if the dir is /home/ubuntu/go/src/github.com/foo/bar
	// the index will start at /src/github.com/foo/bar.
	// We'll need to strip "/src/" which is where the
	// magic number 5 comes from.
	index = strings.LastIndex(dir, "/src/")
	return dir[index+5:], true
}

// getGitOrigin checks the .git origin in an attempt
// to correctly determine the code's package path. This
// is Go-specific, since Go code must exist in
// $GOPATH/src/github.com/{owner}/{name}
func getGitOrigin(dir string) (path string, ok bool) {
	// TODO
	return
}

// prints the time as a human readable string
func humanizeDuration(d time.Duration) string {
	if seconds := int(d.Seconds()); seconds < 1 {
		return "Less than a second"
	} else if seconds < 60 {
		return fmt.Sprintf("%d seconds", seconds)
	} else if minutes := int(d.Minutes()); minutes == 1 {
		return "About a minute"
	} else if minutes < 60 {
		return fmt.Sprintf("%d minutes", minutes)
	} else if hours := int(d.Hours()); hours == 1 {
		return "About an hour"
	} else if hours < 48 {
		return fmt.Sprintf("%d hours", hours)
	} else if hours < 24*7*2 {
		return fmt.Sprintf("%d days", hours/24)
	} else if hours < 24*30*3 {
		return fmt.Sprintf("%d weeks", hours/24/7)
	} else if hours < 24*365*2 {
		return fmt.Sprintf("%d months", hours/24/30)
	}
	return fmt.Sprintf("%f years", d.Hours()/24/365)
}


// getExternalVariables retrieves all environment variables with a specific
// prefix and puts them into a map. The keys will have the prefix stripped
// away.
func getExternalVariables(prefix string, dest map[string]string) {
	// pull in environment variables for the drone command
	for _, kv := range os.Environ() {
		envvar := strings.SplitN(kv, "=", 2)
		if len(envvar) != 2 {
			continue
		}

		key := envvar[0]
		value := envvar[1]
		if strings.HasPrefix(key, prefix) {
			dest[strings.TrimPrefix(key, prefix)] = value
		}
	}
}
