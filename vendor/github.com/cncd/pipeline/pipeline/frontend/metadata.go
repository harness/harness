package frontend

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Event types corresponding to scm hooks.
const (
	EventPush   = "push"
	EventPull   = "pull_request"
	EventTag    = "tag"
	EventDeploy = "deployment"
)

type (
	// Metadata defines runtime m.
	Metadata struct {
		ID   string `json:"id,omitempty"`
		Repo Repo   `json:"repo,omitempty"`
		Curr Build  `json:"curr,omitempty"`
		Prev Build  `json:"prev,omitempty"`
		Job  Job    `json:"job,omitempty"`
		Sys  System `json:"sys,omitempty"`
	}

	// Repo defines runtime metadata for a repository.
	Repo struct {
		Name    string   `json:"name,omitempty"`
		Link    string   `json:"link,omitempty"`
		Remote  string   `json:"remote,omitempty"`
		Private bool     `json:"private,omitempty"`
		Secrets []Secret `json:"secrets,omitempty"`
	}

	// Build defines runtime metadata for a build.
	Build struct {
		Number   int    `json:"number,omitempty"`
		Created  int64  `json:"created,omitempty"`
		Started  int64  `json:"started,omitempty"`
		Finished int64  `json:"finished,omitempty"`
		Timeout  int64  `json:"timeout,omitempty"`
		Status   string `json:"status,omitempty"`
		Event    string `json:"event,omitempty"`
		Link     string `json:"link,omitempty"`
		Target   string `json:"target,omitempty"`
		Trusted  bool   `json:"trusted,omitempty"`
		Commit   Commit `json:"commit,omitempty"`
		Parent   int    `json:"parent,omitempty"`
	}

	// Commit defines runtime metadata for a commit.
	Commit struct {
		Sha     string `json:"sha,omitempty"`
		Ref     string `json:"ref,omitempty"`
		Refspec string `json:"refspec,omitempty"`
		Branch  string `json:"branch,omitempty"`
		Message string `json:"message,omitempty"`
		Author  Author `json:"author,omitempty"`
	}

	// Author defines runtime metadata for a commit author.
	Author struct {
		Name   string `json:"name,omitempty"`
		Email  string `json:"email,omitempty"`
		Avatar string `json:"avatar,omitempty"`
	}

	// Job defines runtime metadata for a job.
	Job struct {
		Number int               `json:"number,omitempty"`
		Matrix map[string]string `json:"matrix,omitempty"`
	}

	// Secret defines a runtime secret
	Secret struct {
		Name  string `json:"name,omitempty"`
		Value string `json:"value,omitempty"`
		Mount string `json:"mount,omitempty"`
		Mask  bool   `json:"mask,omitempty"`
	}

	// System defines runtime metadata for a ci/cd system.
	System struct {
		Name    string `json:"name,omitempty"`
		Host    string `json:"host,omitempty"`
		Link    string `json:"link,omitempty"`
		Arch    string `json:"arch,omitempty"`
		Version string `json:"version,omitempty"`
	}
)

// Environ returns the metadata as a map of environment variables.
func (m *Metadata) Environ() map[string]string {
	params := map[string]string{
		"CI_REPO":                      m.Repo.Name,
		"CI_REPO_NAME":                 m.Repo.Name,
		"CI_REPO_LINK":                 m.Repo.Link,
		"CI_REPO_REMOTE":               m.Repo.Remote,
		"CI_REMOTE_URL":                m.Repo.Remote,
		"CI_REPO_PRIVATE":              strconv.FormatBool(m.Repo.Private),
		"CI_BUILD_NUMBER":              strconv.Itoa(m.Curr.Number),
		"CI_PARENT_BUILD_NUMBER":       strconv.Itoa(m.Curr.Parent),
		"CI_BUILD_CREATED":             strconv.FormatInt(m.Curr.Created, 10),
		"CI_BUILD_STARTED":             strconv.FormatInt(m.Curr.Started, 10),
		"CI_BUILD_FINISHED":            strconv.FormatInt(m.Curr.Finished, 10),
		"CI_BUILD_STATUS":              m.Curr.Status,
		"CI_BUILD_EVENT":               m.Curr.Event,
		"CI_BUILD_LINK":                m.Curr.Link,
		"CI_BUILD_TARGET":              m.Curr.Target,
		"CI_COMMIT_SHA":                m.Curr.Commit.Sha,
		"CI_COMMIT_REF":                m.Curr.Commit.Ref,
		"CI_COMMIT_REFSPEC":            m.Curr.Commit.Refspec,
		"CI_COMMIT_BRANCH":             m.Curr.Commit.Branch,
		"CI_COMMIT_MESSAGE":            m.Curr.Commit.Message,
		"CI_COMMIT_AUTHOR":             m.Curr.Commit.Author.Name,
		"CI_COMMIT_AUTHOR_NAME":        m.Curr.Commit.Author.Name,
		"CI_COMMIT_AUTHOR_EMAIL":       m.Curr.Commit.Author.Email,
		"CI_COMMIT_AUTHOR_AVATAR":      m.Curr.Commit.Author.Avatar,
		"CI_PREV_BUILD_NUMBER":         strconv.Itoa(m.Prev.Number),
		"CI_PREV_BUILD_CREATED":        strconv.FormatInt(m.Prev.Created, 10),
		"CI_PREV_BUILD_STARTED":        strconv.FormatInt(m.Prev.Started, 10),
		"CI_PREV_BUILD_FINISHED":       strconv.FormatInt(m.Prev.Finished, 10),
		"CI_PREV_BUILD_STATUS":         m.Prev.Status,
		"CI_PREV_BUILD_EVENT":          m.Prev.Event,
		"CI_PREV_BUILD_LINK":           m.Prev.Link,
		"CI_PREV_COMMIT_SHA":           m.Prev.Commit.Sha,
		"CI_PREV_COMMIT_REF":           m.Prev.Commit.Ref,
		"CI_PREV_COMMIT_REFSPEC":       m.Prev.Commit.Refspec,
		"CI_PREV_COMMIT_BRANCH":        m.Prev.Commit.Branch,
		"CI_PREV_COMMIT_MESSAGE":       m.Prev.Commit.Message,
		"CI_PREV_COMMIT_AUTHOR":        m.Prev.Commit.Author.Name,
		"CI_PREV_COMMIT_AUTHOR_NAME":   m.Prev.Commit.Author.Name,
		"CI_PREV_COMMIT_AUTHOR_EMAIL":  m.Prev.Commit.Author.Email,
		"CI_PREV_COMMIT_AUTHOR_AVATAR": m.Prev.Commit.Author.Avatar,
		"CI_JOB_NUMBER":                strconv.Itoa(m.Job.Number),
		"CI_SYSTEM":                    m.Sys.Name,
		"CI_SYSTEM_NAME":               m.Sys.Name,
		"CI_SYSTEM_LINK":               m.Sys.Link,
		"CI_SYSTEM_HOST":               m.Sys.Host,
		"CI_SYSTEM_ARCH":               m.Sys.Arch,
		"CI_SYSTEM_VERSION":            m.Sys.Version,
		"CI":                           m.Sys.Name,
	}
	if m.Curr.Event == EventTag {
		params["CI_TAG"] = strings.TrimPrefix(m.Curr.Commit.Ref, "refs/tags/")
	}
	if m.Curr.Event == EventPull {
		params["CI_PULL_REQUEST"] = pullRegexp.FindString(m.Curr.Commit.Ref)
	}
	return params
}

// EnvironDrone returns metadata as a map of DRONE_ environment variables.
// TODO: This is here for backward compatibility and will eventually be removed.
func (m *Metadata) EnvironDrone() map[string]string {
	// MISSING PARAMETERS
	// * DRONE_REPO_TRUSTED
	// * DRONE_YAML_VERIFIED
	// * DRONE_YAML_VERIFIED
	var (
		owner string
		name  string

		parts = strings.Split(m.Repo.Name, "/")
	)
	if len(parts) == 2 {
		owner = strings.Split(m.Repo.Name, "/")[0]
		name = strings.Split(m.Repo.Name, "/")[1]
	} else {
		name = m.Repo.Name
	}
	params := map[string]string{
		"CI":                         "drone",
		"DRONE":                      "true",
		"DRONE_ARCH":                 "linux/amd64",
		"DRONE_REPO":                 m.Repo.Name,
		"DRONE_REPO_SCM":             "git",
		"DRONE_REPO_OWNER":           owner,
		"DRONE_REPO_NAME":            name,
		"DRONE_REPO_LINK":            m.Repo.Link,
		"DRONE_REPO_BRANCH":          m.Curr.Commit.Branch,
		"DRONE_REPO_PRIVATE":         fmt.Sprintf("%v", m.Repo.Private),
		"DRONE_REPO_TRUSTED":         "false", // TODO should this be added?
		"DRONE_REMOTE_URL":           m.Repo.Remote,
		"DRONE_COMMIT_SHA":           m.Curr.Commit.Sha,
		"DRONE_COMMIT_REF":           m.Curr.Commit.Ref,
		"DRONE_COMMIT_REFSPEC":       m.Curr.Commit.Refspec,
		"DRONE_COMMIT_BRANCH":        m.Curr.Commit.Branch,
		"DRONE_COMMIT_LINK":          m.Curr.Link,
		"DRONE_COMMIT_MESSAGE":       m.Curr.Commit.Message,
		"DRONE_COMMIT_AUTHOR":        m.Curr.Commit.Author.Name,
		"DRONE_COMMIT_AUTHOR_EMAIL":  m.Curr.Commit.Author.Email,
		"DRONE_COMMIT_AUTHOR_AVATAR": m.Curr.Commit.Author.Avatar,
		"DRONE_BUILD_NUMBER":         fmt.Sprintf("%d", m.Curr.Number),
		"DRONE_PARENT_BUILD_NUMBER":  fmt.Sprintf("%d", m.Curr.Parent),
		"DRONE_BUILD_EVENT":          m.Curr.Event,
		"DRONE_BUILD_LINK":           fmt.Sprintf("%s/%s/%d", m.Sys.Link, m.Repo.Name, m.Curr.Number),
		"DRONE_BUILD_CREATED":        fmt.Sprintf("%d", m.Curr.Created),
		"DRONE_BUILD_STARTED":        fmt.Sprintf("%d", m.Curr.Started),
		"DRONE_BUILD_FINISHED":       fmt.Sprintf("%d", m.Curr.Finished),
		"DRONE_JOB_NUMBER":           fmt.Sprintf("%d", m.Job.Number),
		"DRONE_JOB_STARTED":          fmt.Sprintf("%d", m.Curr.Started), // ISSUE: no job started
		"DRONE_BRANCH":               m.Curr.Commit.Branch,
		"DRONE_COMMIT":               m.Curr.Commit.Sha,
		"DRONE_VERSION":              m.Sys.Version,
		"DRONE_DEPLOY_TO":            m.Curr.Target,
		"DRONE_PREV_BUILD_STATUS":    m.Prev.Status,
		"DRONE_PREV_BUILD_NUMBER":    fmt.Sprintf("%v", m.Prev.Number),
		"DRONE_PREV_COMMIT_SHA":      m.Prev.Commit.Sha,
	}
	if m.Curr.Event == EventTag {
		params["DRONE_TAG"] = strings.TrimPrefix(m.Curr.Commit.Ref, "refs/tags/")
	}
	if m.Curr.Event == EventPull {
		params["DRONE_PULL_REQUEST"] = pullRegexp.FindString(m.Curr.Commit.Ref)
	}
	return params
}

var pullRegexp = regexp.MustCompile("\\d+")
