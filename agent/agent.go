package agent

import (
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/drone/drone/build"
	"github.com/drone/drone/model"
	"github.com/drone/drone/queue"
	"github.com/drone/drone/version"
	"github.com/drone/drone/yaml"
	"github.com/drone/drone/yaml/expander"
	"github.com/drone/drone/yaml/transform"
)

type Logger interface {
	Write(*build.Line)
}

type Agent struct {
	Update    UpdateFunc
	Logger    LoggerFunc
	Engine    build.Engine
	Timeout   time.Duration
	Platform  string
	Namespace string
	Disable   []string
	Escalate  []string
	Netrc     []string
	Local     string
	Pull      bool
}

func (a *Agent) Poll() error {

	// logrus.Infof("Starting build %s/%s#%d.%d",
	// 	payload.Repo.Owner, payload.Repo.Name, payload.Build.Number, payload.Job.Number)
	//
	//
	// 	logrus.Infof("Finished build %s/%s#%d.%d",
	// 		payload.Repo.Owner, payload.Repo.Name, payload.Build.Number, payload.Job.Number)

	return nil
}

func (a *Agent) Run(payload *queue.Work, cancel <-chan bool) error {

	payload.Job.Status = model.StatusRunning
	payload.Job.Started = time.Now().Unix()

	spec, err := a.prep(payload)
	if err != nil {
		payload.Job.Error = err.Error()
		payload.Job.ExitCode = 255
		payload.Job.Finished = payload.Job.Started
		payload.Job.Status = model.StatusError
		a.Update(payload)
		return err
	}
	a.Update(payload)
	err = a.exec(spec, payload, cancel)

	if err != nil {
		payload.Job.ExitCode = 255
		payload.Job.Error = err.Error()
	}
	if exitErr, ok := err.(*build.ExitError); ok {
		payload.Job.ExitCode = exitErr.Code
		payload.Job.Error = "" // exit errors are already written to the log
	}

	payload.Job.Finished = time.Now().Unix()

	switch payload.Job.ExitCode {
	case 128, 130, 137:
		payload.Job.Status = model.StatusKilled
	case 0:
		payload.Job.Status = model.StatusSuccess
	default:
		payload.Job.Status = model.StatusFailure
	}

	a.Update(payload)

	return err
}

func (a *Agent) prep(w *queue.Work) (*yaml.Config, error) {

	envs := toEnv(w)
	w.Yaml = expander.ExpandString(w.Yaml, envs)

	// inject the netrc file into the clone plugin if the repository is
	// private and requires authentication.
	var secrets []*model.Secret
	if w.Verified {
		secrets = append(secrets, w.Secrets...)
	}

	if w.Repo.IsPrivate {
		secrets = append(secrets, &model.Secret{
			Name:   "DRONE_NETRC_USERNAME",
			Value:  w.Netrc.Login,
			Images: []string{"*"},
			Events: []string{"*"},
		})
		secrets = append(secrets, &model.Secret{
			Name:   "DRONE_NETRC_PASSWORD",
			Value:  w.Netrc.Password,
			Images: []string{"*"},
			Events: []string{"*"},
		})
		secrets = append(secrets, &model.Secret{
			Name:   "DRONE_NETRC_MACHINE",
			Value:  w.Netrc.Machine,
			Images: []string{"*"},
			Events: []string{"*"},
		})
	}

	conf, err := yaml.ParseString(w.Yaml)
	if err != nil {
		return nil, err
	}

	src := "src"
	if url, _ := url.Parse(w.Repo.Link); url != nil {
		host, _, err := net.SplitHostPort(url.Host)
		if err == nil {
			url.Host = host
		}
		src = filepath.Join(src, url.Host, url.Path)
	}

	transform.Clone(conf, w.Repo.Kind)
	transform.Environ(conf, envs)
	transform.DefaultFilter(conf)
	if w.BuildLast != nil {
		transform.ChangeFilter(conf, w.BuildLast.Status)
	}

	transform.ImageSecrets(conf, secrets, w.Build.Event)
	transform.Identifier(conf)
	transform.WorkspaceTransform(conf, "/drone", src)

	if err := transform.Check(conf, w.Repo.IsTrusted); err != nil {
		return nil, err
	}

	transform.CommandTransform(conf)
	transform.ImagePull(conf, a.Pull)
	transform.ImageTag(conf)
	transform.ImageName(conf)
	transform.ImageNamespace(conf, a.Namespace)
	if err := transform.ImageEscalate(conf, a.Escalate); err != nil {
		return nil, err
	}
	transform.PluginParams(conf)

	if a.Local != "" {
		transform.PluginDisable(conf, a.Disable)
		transform.ImageVolume(conf, []string{a.Local + ":" + conf.Workspace.Path})
	}

	transform.Pod(conf)

	return conf, nil
}

func (a *Agent) exec(spec *yaml.Config, payload *queue.Work, cancel <-chan bool) error {

	conf := build.Config{
		Engine: a.Engine,
		Buffer: 500,
	}

	pipeline := conf.Pipeline(spec)
	defer pipeline.Teardown()

	// setup the build environment
	if err := pipeline.Setup(); err != nil {
		return err
	}

	timeout := time.After(time.Duration(payload.Repo.Timeout) * time.Minute)

	for {
		select {
		case <-pipeline.Done():
			return pipeline.Err()
		case <-cancel:
			pipeline.Stop()
			return fmt.Errorf("termination request received, build cancelled")
		case <-timeout:
			pipeline.Stop()
			return fmt.Errorf("maximum time limit exceeded, build cancelled")
		case <-time.After(a.Timeout):
			pipeline.Stop()
			return fmt.Errorf("terminal inactive for %v, build cancelled", a.Timeout)
		case <-pipeline.Next():

			// TODO(bradrydzewski) this entire block of code should probably get
			// encapsulated in the pipeline.
			status := model.StatusSuccess
			if pipeline.Err() != nil {
				status = model.StatusFailure
			}
			// updates the build status passed into each container. I realize this is
			// a bit out of place and will work to resolve.
			pipeline.Head().Environment["DRONE_BUILD_STATUS"] = status

			if !pipeline.Head().Constraints.Match(
				a.Platform,
				payload.Build.Deploy,
				payload.Build.Event,
				payload.Build.Branch,
				status, payload.Job.Environment) { // TODO: fix this whole section

				pipeline.Skip()
			} else {
				pipeline.Exec()
			}
		case line := <-pipeline.Pipe():
			a.Logger(line)
		}
	}
}

func toEnv(w *queue.Work) map[string]string {
	envs := map[string]string{
		"CI":                         "drone",
		"DRONE":                      "true",
		"DRONE_ARCH":                 "linux/amd64",
		"DRONE_REPO":                 w.Repo.FullName,
		"DRONE_REPO_SCM":             w.Repo.Kind,
		"DRONE_REPO_OWNER":           w.Repo.Owner,
		"DRONE_REPO_NAME":            w.Repo.Name,
		"DRONE_REPO_LINK":            w.Repo.Link,
		"DRONE_REPO_AVATAR":          w.Repo.Avatar,
		"DRONE_REPO_BRANCH":          w.Repo.Branch,
		"DRONE_REPO_PRIVATE":         fmt.Sprintf("%v", w.Repo.IsPrivate),
		"DRONE_REPO_TRUSTED":         fmt.Sprintf("%v", w.Repo.IsTrusted),
		"DRONE_REMOTE_URL":           w.Repo.Clone,
		"DRONE_COMMIT_SHA":           w.Build.Commit,
		"DRONE_COMMIT_REF":           w.Build.Ref,
		"DRONE_COMMIT_BRANCH":        w.Build.Branch,
		"DRONE_COMMIT_LINK":          w.Build.Link,
		"DRONE_COMMIT_MESSAGE":       w.Build.Message,
		"DRONE_COMMIT_AUTHOR":        w.Build.Author,
		"DRONE_COMMIT_AUTHOR_EMAIL":  w.Build.Email,
		"DRONE_COMMIT_AUTHOR_AVATAR": w.Build.Avatar,
		"DRONE_BUILD_NUMBER":         fmt.Sprintf("%d", w.Build.Number),
		"DRONE_BUILD_EVENT":          w.Build.Event,
		"DRONE_BUILD_STATUS":         w.Build.Status,
		"DRONE_BUILD_LINK":           fmt.Sprintf("%s/%s/%d", w.System.Link, w.Repo.FullName, w.Build.Number),
		"DRONE_BUILD_CREATED":        fmt.Sprintf("%d", w.Build.Created),
		"DRONE_BUILD_STARTED":        fmt.Sprintf("%d", w.Build.Started),
		"DRONE_BUILD_FINISHED":       fmt.Sprintf("%d", w.Build.Finished),
		"DRONE_JOB_NUMBER":           fmt.Sprintf("%d", w.Job.Number),
		"DRONE_JOB_STATUS":           w.Job.Status,
		"DRONE_JOB_ERROR":            w.Job.Error,
		"DRONE_JOB_EXIT_CODE":        fmt.Sprintf("%d", w.Job.ExitCode),
		"DRONE_JOB_STARTED":          fmt.Sprintf("%d", w.Job.Started),
		"DRONE_JOB_FINISHED":         fmt.Sprintf("%d", w.Job.Finished),
		"DRONE_YAML_VERIFIED":        fmt.Sprintf("%v", w.Verified),
		"DRONE_YAML_SIGNED":          fmt.Sprintf("%v", w.Signed),
		"DRONE_BRANCH":               w.Build.Branch,
		"DRONE_COMMIT":               w.Build.Commit,
		"DRONE_VERSION":              version.Version,
	}

	if w.Build.Event == model.EventTag {
		envs["DRONE_TAG"] = strings.TrimPrefix(w.Build.Ref, "refs/tags/")
	}
	if w.Build.Event == model.EventPull {
		envs["DRONE_PULL_REQUEST"] = pullRegexp.FindString(w.Build.Ref)
	}
	if w.Build.Event == model.EventDeploy {
		envs["DRONE_DEPLOY_TO"] = w.Build.Deploy
	}

	if w.BuildLast != nil {
		envs["DRONE_PREV_BUILD_STATUS"] = w.BuildLast.Status
		envs["DRONE_PREV_BUILD_NUMBER"] = fmt.Sprintf("%v", w.BuildLast.Number)
		envs["DRONE_PREV_COMMIT_SHA"] = w.BuildLast.Commit
	}

	// inject matrix values as environment variables
	for key, val := range w.Job.Environment {
		envs[key] = val
	}
	return envs
}

var pullRegexp = regexp.MustCompile("\\d+")
