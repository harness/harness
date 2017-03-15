package server

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/square/go-jose"

	"github.com/Sirupsen/logrus"
	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/token"
	"github.com/drone/drone/store"
	"github.com/drone/envsubst"

	"github.com/cncd/pipeline/pipeline/backend"
	"github.com/cncd/pipeline/pipeline/frontend"
	"github.com/cncd/pipeline/pipeline/frontend/yaml"
	"github.com/cncd/pipeline/pipeline/frontend/yaml/compiler"
	"github.com/cncd/pipeline/pipeline/frontend/yaml/linter"
	"github.com/cncd/pipeline/pipeline/frontend/yaml/matrix"
	"github.com/cncd/pipeline/pipeline/rpc"
	"github.com/cncd/pubsub"
	"github.com/cncd/queue"
)

//
// CANARY IMPLEMENTATION
//
// This file is a complete disaster because I'm trying to wedge in some
// experimental code. Please pardon our appearance during renovations.
//

var skipRe = regexp.MustCompile(`\[(?i:ci *skip|skip *ci)\]`)

func GetQueueInfo(c *gin.Context) {
	c.IndentedJSON(200,
		config.queue.Info(c),
	)
}

// func PostHookOld(c *gin.Context) {
// 	remote_ := remote.FromContext(c)
//
// 	tmprepo, build, err := remote_.Hook(c.Request)
// 	if err != nil {
// 		logrus.Errorf("failure to parse hook. %s", err)
// 		c.AbortWithError(400, err)
// 		return
// 	}
// 	if build == nil {
// 		c.Writer.WriteHeader(200)
// 		return
// 	}
// 	if tmprepo == nil {
// 		logrus.Errorf("failure to ascertain repo from hook.")
// 		c.Writer.WriteHeader(400)
// 		return
// 	}
//
// 	// skip the build if any case-insensitive combination of the words "skip" and "ci"
// 	// wrapped in square brackets appear in the commit message
// 	skipMatch := skipRe.FindString(build.Message)
// 	if len(skipMatch) > 0 {
// 		logrus.Infof("ignoring hook. %s found in %s", skipMatch, build.Commit)
// 		c.Writer.WriteHeader(204)
// 		return
// 	}
//
// 	repo, err := store.GetRepoOwnerName(c, tmprepo.Owner, tmprepo.Name)
// 	if err != nil {
// 		logrus.Errorf("failure to find repo %s/%s from hook. %s", tmprepo.Owner, tmprepo.Name, err)
// 		c.AbortWithError(404, err)
// 		return
// 	}
//
// 	// get the token and verify the hook is authorized
// 	parsed, err := token.ParseRequest(c.Request, func(t *token.Token) (string, error) {
// 		return repo.Hash, nil
// 	})
// 	if err != nil {
// 		logrus.Errorf("failure to parse token from hook for %s. %s", repo.FullName, err)
// 		c.AbortWithError(400, err)
// 		return
// 	}
// 	if parsed.Text != repo.FullName {
// 		logrus.Errorf("failure to verify token from hook. Expected %s, got %s", repo.FullName, parsed.Text)
// 		c.AbortWithStatus(403)
// 		return
// 	}
//
// 	if repo.UserID == 0 {
// 		logrus.Warnf("ignoring hook. repo %s has no owner.", repo.FullName)
// 		c.Writer.WriteHeader(204)
// 		return
// 	}
// 	var skipped = true
// 	if (build.Event == model.EventPush && repo.AllowPush) ||
// 		(build.Event == model.EventPull && repo.AllowPull) ||
// 		(build.Event == model.EventDeploy && repo.AllowDeploy) ||
// 		(build.Event == model.EventTag && repo.AllowTag) {
// 		skipped = false
// 	}
//
// 	if skipped {
// 		logrus.Infof("ignoring hook. repo %s is disabled for %s events.", repo.FullName, build.Event)
// 		c.Writer.WriteHeader(204)
// 		return
// 	}
//
// 	user, err := store.GetUser(c, repo.UserID)
// 	if err != nil {
// 		logrus.Errorf("failure to find repo owner %s. %s", repo.FullName, err)
// 		c.AbortWithError(500, err)
// 		return
// 	}
//
// 	// if the remote has a refresh token, the current access token
// 	// may be stale. Therefore, we should refresh prior to dispatching
// 	// the job.
// 	if refresher, ok := remote_.(remote.Refresher); ok {
// 		ok, _ := refresher.Refresh(user)
// 		if ok {
// 			store.UpdateUser(c, user)
// 		}
// 	}
//
// 	// fetch the build file from the database
// 	cfg := ToConfig(c)
// 	raw, err := remote_.File(user, repo, build, cfg.Yaml)
// 	if err != nil {
// 		logrus.Errorf("failure to get build config for %s. %s", repo.FullName, err)
// 		c.AbortWithError(404, err)
// 		return
// 	}
// 	sec, err := remote_.File(user, repo, build, cfg.Shasum)
// 	if err != nil {
// 		logrus.Debugf("cannot find yaml signature for %s. %s", repo.FullName, err)
// 		// NOTE we don't exit on failure. The sec file is optional
// 	}
//
// 	axes, err := matrix.Parse(raw)
// 	if err != nil {
// 		c.String(500, "Failed to parse yaml file or calculate matrix. %s", err)
// 		return
// 	}
// 	if len(axes) == 0 {
// 		axes = append(axes, matrix.Axis{})
// 	}
//
// 	netrc, err := remote_.Netrc(user, repo)
// 	if err != nil {
// 		c.String(500, "Failed to generate netrc file. %s", err)
// 		return
// 	}
//
// 	// verify the branches can be built vs skipped
// 	branches, err := yaml.ParseBytes(raw)
// 	if err != nil {
// 		c.String(500, "Failed to parse yaml file. %s", err)
// 		return
// 	}
// 	if !branches.Branches.Match(build.Branch) && build.Event != model.EventTag && build.Event != model.EventDeploy {
// 		c.String(200, "Branch does not match restrictions defined in yaml")
// 		return
// 	}
//
// 	signature, err := jose.ParseSigned(string(sec))
// 	if err != nil {
// 		logrus.Debugf("cannot parse .drone.yml.sig file. %s", err)
// 	} else if len(sec) == 0 {
// 		logrus.Debugf("cannot parse .drone.yml.sig file. empty file")
// 	} else {
// 		build.Signed = true
// 		output, verr := signature.Verify([]byte(repo.Hash))
// 		if verr != nil {
// 			logrus.Debugf("cannot verify .drone.yml.sig file. %s", verr)
// 		} else if string(output) != string(raw) {
// 			logrus.Debugf("cannot verify .drone.yml.sig file. no match")
// 		} else {
// 			build.Verified = true
// 		}
// 	}
//
// 	// update some build fields
// 	build.Status = model.StatusPending
// 	build.RepoID = repo.ID
//
// 	// and use a transaction
// 	var jobs []*model.Job
// 	for num, axis := range axes {
// 		jobs = append(jobs, &model.Job{
// 			BuildID:     build.ID,
// 			Number:      num + 1,
// 			Status:      model.StatusPending,
// 			Environment: axis,
// 		})
// 	}
// 	err = store.CreateBuild(c, build, jobs...)
// 	if err != nil {
// 		logrus.Errorf("failure to save commit for %s. %s", repo.FullName, err)
// 		c.AbortWithError(500, err)
// 		return
// 	}
//
// 	c.JSON(200, build)
//
// 	uri := fmt.Sprintf("%s/%s/%d", httputil.GetURL(c.Request), repo.FullName, build.Number)
// 	err = remote_.Status(user, repo, build, uri)
// 	if err != nil {
// 		logrus.Errorf("error setting commit status for %s/%d", repo.FullName, build.Number)
// 	}
//
// 	// get the previous build so that we can send
// 	// on status change notifications
// 	last, _ := store.GetBuildLastBefore(c, repo, build.Branch, build.ID)
// 	secs, err := store.GetMergedSecretList(c, repo)
// 	if err != nil {
// 		logrus.Debugf("Error getting secrets for %s#%d. %s", repo.FullName, build.Number, err)
// 	}
//
// 	//
// 	// BELOW: NEW
// 	//
//
// 	b := builder{
// 		Repo:  repo,
// 		Curr:  build,
// 		Last:  last,
// 		Netrc: netrc,
// 		Secs:  secs,
// 		Link:  httputil.GetURL(c.Request),
// 		Yaml:  string(raw),
// 	}
// 	items, err := b.Build()
// 	if err != nil {
// 		build.Status = model.StatusError
// 		build.Started = time.Now().Unix()
// 		build.Finished = build.Started
// 		build.Error = err.Error()
// 		store.CreateBuild(c, build, build.Jobs...)
// 		return
// 	}
//
// 	for _, item := range items {
// 		build.Jobs = append(build.Jobs, item.Job)
// 	}
//
// 	if err := store.CreateBuild(c, build, build.Jobs...); err != nil {
// 		logrus.Errorf("failure to save commit for %s. %s", repo.FullName, err)
// 		c.AbortWithError(500, err)
// 		return
// 	}
//
// 	for _, item := range items {
//
// 		task := new(queue.Task)
// 		task.ID = fmt.Sprint(item.Job.ID)
// 		task.Labels = map[string]string{}
// 		task.Labels["platform"] = item.Platform
// 		for k, v := range item.Labels {
// 			task.Labels[k] = v
// 		}
//
// 		task.Data, _ = json.Marshal(rpc.Pipeline{
// 			ID:      fmt.Sprint(item.Job.ID),
// 			Config:  item.Config,
// 			Timeout: b.Repo.Timeout,
// 		})
//
// 		config.logger.Open(context.Background(), task.ID)
// 		config.queue.Push(context.Background(), task)
// 	}
//
// 	//
// 	// new code here
// 	//
//
// 	message := pubsub.Message{
// 		Labels: map[string]string{
// 			"repo":    repo.FullName,
// 			"private": strconv.FormatBool(repo.IsPrivate),
// 		},
// 	}
// 	message.Data, _ = json.Marshal(model.Event{
// 		Type:  model.Enqueued,
// 		Repo:  *repo,
// 		Build: *build,
// 	})
// 	// TODO remove global reference
// 	config.pubsub.Publish(c, "topic/events", message)
//
// 	//
// 	// workspace
// 	//
//
// 	for _, job := range jobs {
//
// 		metadata := metadataFromStruct(repo, build, last, job, httputil.GetURL(c.Request))
// 		environ := metadata.Environ()
//
// 		secrets := map[string]string{}
// 		for _, sec := range secs {
// 			if !sec.MatchEvent(build.Event) {
// 				continue
// 			}
// 			if build.Verified || sec.SkipVerify {
// 				secrets[sec.Name] = sec.Value
// 			}
// 		}
// 		sub := func(name string) string {
// 			if v, ok := environ[name]; ok {
// 				return v
// 			}
// 			return secrets[name]
// 		}
// 		if s, err := envsubst.Eval(string(raw), sub); err != nil {
// 			raw = []byte(s)
// 		}
// 		parsed, err := yaml.ParseBytes(raw)
// 		if err != nil {
// 			job.ExitCode = 255
// 			job.Enqueued = time.Now().Unix()
// 			job.Started = time.Now().Unix()
// 			job.Finished = time.Now().Unix()
// 			job.Error = err.Error()
// 			store.UpdateBuildJob(c, build, job)
// 			continue
// 		}
//
// 		lerr := linter.New(
// 			linter.WithTrusted(repo.IsTrusted),
// 		).Lint(parsed)
// 		if lerr != nil {
// 			job.ExitCode = 255
// 			job.Enqueued = time.Now().Unix()
// 			job.Started = time.Now().Unix()
// 			job.Finished = time.Now().Unix()
// 			job.Error = lerr.Error()
// 			store.UpdateBuildJob(c, build, job)
// 			continue
// 		}
//
// 		ir := compiler.New(
// 			compiler.WithEnviron(environ),
// 			// TODO ability to customize the escalated plugins
// 			compiler.WithEscalated("plugins/docker", "plugins/gcr", "plugins/ecr"),
// 			compiler.WithLocal(false),
// 			compiler.WithNetrc(netrc.Login, netrc.Password, netrc.Machine),
// 			compiler.WithPrefix(
// 				fmt.Sprintf(
// 					"%d_%d",
// 					job.ID,
// 					time.Now().Unix(),
// 				),
// 			),
// 			compiler.WithEnviron(job.Environment),
// 			compiler.WithProxy(),
// 			// TODO ability to set global volumes for things like certs
// 			compiler.WithVolumes(),
// 			compiler.WithWorkspaceFromURL("/drone", repo.Link),
// 		).Compile(parsed)
//
// 		// TODO there is a chicken and egg problem here because
// 		// the compiled yaml has a platform environment variable
// 		// that is not correctly set, because we are just about
// 		// to set it ....
// 		// TODO maybe we remove platform from metadata and let
// 		// the compiler set the value from the yaml itself.
// 		if parsed.Platform == "" {
// 			parsed.Platform = "linux/amd64"
// 		}
//
// 		for _, sec := range secs {
// 			if !sec.MatchEvent(build.Event) {
// 				continue
// 			}
// 			if build.Verified || sec.SkipVerify {
// 				ir.Secrets = append(ir.Secrets, &backend.Secret{
// 					Mask:  sec.Conceal,
// 					Name:  sec.Name,
// 					Value: sec.Value,
// 				})
// 			}
// 		}
//
// 		task := new(queue.Task)
// 		task.ID = fmt.Sprint(job.ID)
// 		task.Labels = map[string]string{}
// 		task.Labels["platform"] = parsed.Platform
// 		if parsed.Labels != nil {
// 			for k, v := range parsed.Labels {
// 				task.Labels[k] = v
// 			}
// 		}
//
// 		task.Data, _ = json.Marshal(rpc.Pipeline{
// 			ID:      fmt.Sprint(job.ID),
// 			Config:  ir,
// 			Timeout: repo.Timeout,
// 		})
//
// 		config.logger.Open(context.Background(), task.ID)
// 		config.queue.Push(context.Background(), task)
// 	}
//
// }

// return the metadata from the cli context.
func metadataFromStruct(repo *model.Repo, build, last *model.Build, job *model.Job, link string) frontend.Metadata {
	return frontend.Metadata{
		Repo: frontend.Repo{
			Name:    repo.Name,
			Link:    repo.Link,
			Remote:  repo.Clone,
			Private: repo.IsPrivate,
		},
		Curr: frontend.Build{
			Number:   build.Number,
			Created:  build.Created,
			Started:  build.Started,
			Finished: build.Finished,
			Status:   build.Status,
			Event:    build.Event,
			Link:     build.Link,
			Target:   build.Deploy,
			Commit: frontend.Commit{
				Sha:     build.Commit,
				Ref:     build.Ref,
				Refspec: build.Refspec,
				Branch:  build.Branch,
				Message: build.Message,
				Author: frontend.Author{
					Name:   build.Author,
					Email:  build.Email,
					Avatar: build.Avatar,
				},
			},
		},
		Prev: frontend.Build{
			Number:   last.Number,
			Created:  last.Created,
			Started:  last.Started,
			Finished: last.Finished,
			Status:   last.Status,
			Event:    last.Event,
			Link:     last.Link,
			Target:   last.Deploy,
			Commit: frontend.Commit{
				Sha:     last.Commit,
				Ref:     last.Ref,
				Refspec: last.Refspec,
				Branch:  last.Branch,
				Message: last.Message,
				Author: frontend.Author{
					Name:   last.Author,
					Email:  last.Email,
					Avatar: last.Avatar,
				},
			},
		},
		Job: frontend.Job{
			Number: job.Number,
			Matrix: job.Environment,
		},
		Sys: frontend.System{
			Name: "drone",
			Link: link,
			Arch: "linux/amd64",
		},
	}
}

// use helper funciton to return ([]backend.Config, error)

// 1. fetch everything from github
// 2. create and persist the build object
//
// 3. generate the build jobs [Launcher?]
//       a. parse yaml
//       b. lint yaml
//       c. compile yaml
//
// 4. persist the build jobs (... what if I already have jobs, via re-start)
// 5. update github status
// 6. send to queue
// 7. trigger pubsub

type builder struct {
	Repo  *model.Repo
	Curr  *model.Build
	Last  *model.Build
	Netrc *model.Netrc
	Secs  []*model.Secret
	Link  string
	Yaml  string
}

type buildItem struct {
	Job      *model.Job
	Platform string
	Labels   map[string]string
	Config   *backend.Config
}

func (b *builder) Build() ([]*buildItem, error) {

	axes, err := matrix.ParseString(b.Yaml)
	if err != nil {
		return nil, err
	}
	if len(axes) == 0 {
		axes = append(axes, matrix.Axis{})
	}

	var items []*buildItem
	for i, axis := range axes {
		job := &model.Job{
			BuildID:     b.Curr.ID,
			Number:      i + 1,
			Status:      model.StatusPending,
			Environment: axis,
			Enqueued:    b.Curr.Created,
		}

		metadata := metadataFromStruct(b.Repo, b.Curr, b.Last, job, b.Link)
		environ := metadata.Environ()
		for k, v := range metadata.EnvironDrone() {
			environ[k] = v
		}

		secrets := map[string]string{}
		for _, sec := range b.Secs {
			if !sec.MatchEvent(b.Curr.Event) {
				continue
			}
			if b.Curr.Verified || sec.SkipVerify {
				secrets[sec.Name] = sec.Value
			}
		}
		sub := func(name string) string {
			if v, ok := environ[name]; ok {
				return v
			}
			return secrets[name]
		}

		y := b.Yaml
		if s, err := envsubst.Eval(y, sub); err != nil {
			y = s
		}

		parsed, err := yaml.ParseString(y)
		if err != nil {
			return nil, err
		}
		metadata.Sys.Arch = parsed.Platform
		if metadata.Sys.Arch == "" {
			metadata.Sys.Arch = "linux/amd64"
		}

		lerr := linter.New(
			linter.WithTrusted(b.Repo.IsTrusted),
		).Lint(parsed)
		if lerr != nil {
			return nil, err
		}

		ir := compiler.New(
			compiler.WithEnviron(environ),
			// TODO ability to customize the escalated plugins
			compiler.WithEscalated("plugins/docker", "plugins/gcr", "plugins/ecr"),
			compiler.WithLocal(false),
			compiler.WithNetrc(b.Netrc.Login, b.Netrc.Password, b.Netrc.Machine),
			compiler.WithPrefix(
				fmt.Sprintf(
					"%d_%d",
					job.ID,
					time.Now().Unix(),
				),
			),
			compiler.WithEnviron(job.Environment),
			compiler.WithProxy(),
			// TODO ability to set global volumes for things like certs
			compiler.WithVolumes(),
			compiler.WithWorkspaceFromURL("/drone", b.Curr.Link),
		).Compile(parsed)

		for _, sec := range b.Secs {
			if !sec.MatchEvent(b.Curr.Event) {
				continue
			}
			if b.Curr.Verified || sec.SkipVerify {
				ir.Secrets = append(ir.Secrets, &backend.Secret{
					Mask:  sec.Conceal,
					Name:  sec.Name,
					Value: sec.Value,
				})
			}
		}

		item := &buildItem{
			Job:      job,
			Config:   ir,
			Labels:   parsed.Labels,
			Platform: metadata.Sys.Arch,
		}
		if item.Labels == nil {
			item.Labels = map[string]string{}
		}
		items = append(items, item)
	}

	return items, nil
}

//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//

func PostHook2(c *gin.Context) {
	remote_ := remote.FromContext(c)

	tmprepo, build, err := remote_.Hook(c.Request)
	if err != nil {
		logrus.Errorf("failure to parse hook. %s", err)
		c.AbortWithError(400, err)
		return
	}
	if build == nil {
		c.Writer.WriteHeader(200)
		return
	}
	if tmprepo == nil {
		logrus.Errorf("failure to ascertain repo from hook.")
		c.Writer.WriteHeader(400)
		return
	}

	// skip the build if any case-insensitive combination of the words "skip" and "ci"
	// wrapped in square brackets appear in the commit message
	skipMatch := skipRe.FindString(build.Message)
	if len(skipMatch) > 0 {
		logrus.Infof("ignoring hook. %s found in %s", skipMatch, build.Commit)
		c.Writer.WriteHeader(204)
		return
	}

	repo, err := store.GetRepoOwnerName(c, tmprepo.Owner, tmprepo.Name)
	if err != nil {
		logrus.Errorf("failure to find repo %s/%s from hook. %s", tmprepo.Owner, tmprepo.Name, err)
		c.AbortWithError(404, err)
		return
	}

	// get the token and verify the hook is authorized
	parsed, err := token.ParseRequest(c.Request, func(t *token.Token) (string, error) {
		return repo.Hash, nil
	})
	if err != nil {
		logrus.Errorf("failure to parse token from hook for %s. %s", repo.FullName, err)
		c.AbortWithError(400, err)
		return
	}
	if parsed.Text != repo.FullName {
		logrus.Errorf("failure to verify token from hook. Expected %s, got %s", repo.FullName, parsed.Text)
		c.AbortWithStatus(403)
		return
	}

	if repo.UserID == 0 {
		logrus.Warnf("ignoring hook. repo %s has no owner.", repo.FullName)
		c.Writer.WriteHeader(204)
		return
	}
	var skipped = true
	if (build.Event == model.EventPush && repo.AllowPush) ||
		(build.Event == model.EventPull && repo.AllowPull) ||
		(build.Event == model.EventDeploy && repo.AllowDeploy) ||
		(build.Event == model.EventTag && repo.AllowTag) {
		skipped = false
	}

	if skipped {
		logrus.Infof("ignoring hook. repo %s is disabled for %s events.", repo.FullName, build.Event)
		c.Writer.WriteHeader(204)
		return
	}

	user, err := store.GetUser(c, repo.UserID)
	if err != nil {
		logrus.Errorf("failure to find repo owner %s. %s", repo.FullName, err)
		c.AbortWithError(500, err)
		return
	}

	// if the remote has a refresh token, the current access token
	// may be stale. Therefore, we should refresh prior to dispatching
	// the job.
	if refresher, ok := remote_.(remote.Refresher); ok {
		ok, _ := refresher.Refresh(user)
		if ok {
			store.UpdateUser(c, user)
		}
	}

	// fetch the build file from the database
	cfg := ToConfig(c)
	raw, err := remote_.File(user, repo, build, cfg.Yaml)
	if err != nil {
		logrus.Errorf("failure to get build config for %s. %s", repo.FullName, err)
		c.AbortWithError(404, err)
		return
	}
	sec, err := remote_.File(user, repo, build, cfg.Shasum)
	if err != nil {
		logrus.Debugf("cannot find yaml signature for %s. %s", repo.FullName, err)
	}

	netrc, err := remote_.Netrc(user, repo)
	if err != nil {
		c.String(500, "Failed to generate netrc file. %s", err)
		return
	}

	// verify the branches can be built vs skipped
	branches, err := yaml.ParseBytes(raw)
	if err != nil {
		c.String(500, "Failed to parse yaml file. %s", err)
		return
	}
	if !branches.Branches.Match(build.Branch) && build.Event != model.EventTag && build.Event != model.EventDeploy {
		c.String(200, "Branch does not match restrictions defined in yaml")
		return
	}

	signature, err := jose.ParseSigned(string(sec))
	if err != nil {
		logrus.Debugf("cannot parse .drone.yml.sig file. %s", err)
	} else if len(sec) == 0 {
		logrus.Debugf("cannot parse .drone.yml.sig file. empty file")
	} else {
		build.Signed = true
		output, verr := signature.Verify([]byte(repo.Hash))
		if verr != nil {
			logrus.Debugf("cannot verify .drone.yml.sig file. %s", verr)
		} else if string(output) != string(raw) {
			logrus.Debugf("cannot verify .drone.yml.sig file. no match")
		} else {
			build.Verified = true
		}
	}

	// update some build fields
	build.Status = model.StatusPending
	build.RepoID = repo.ID

	if err := store.CreateBuild(c, build, build.Jobs...); err != nil {
		logrus.Errorf("failure to save commit for %s. %s", repo.FullName, err)
		c.AbortWithError(500, err)
		return
	}

	c.JSON(200, build)

	// get the previous build so that we can send
	// on status change notifications
	last, _ := store.GetBuildLastBefore(c, repo, build.Branch, build.ID)
	secs, err := store.GetMergedSecretList(c, repo)
	if err != nil {
		logrus.Debugf("Error getting secrets for %s#%d. %s", repo.FullName, build.Number, err)
	}

	//
	// BELOW: NEW
	//

	defer func() {
		uri := fmt.Sprintf("%s/%s/%d", httputil.GetURL(c.Request), repo.FullName, build.Number)
		err = remote_.Status(user, repo, build, uri)
		if err != nil {
			logrus.Errorf("error setting commit status for %s/%d", repo.FullName, build.Number)
		}
	}()

	b := builder{
		Repo:  repo,
		Curr:  build,
		Last:  last,
		Netrc: netrc,
		Secs:  secs,
		Link:  httputil.GetURL(c.Request),
		Yaml:  string(raw),
	}
	items, err := b.Build()
	if err != nil {
		build.Status = model.StatusError
		build.Started = time.Now().Unix()
		build.Finished = build.Started
		build.Error = err.Error()
		return
	}

	for _, item := range items {
		build.Jobs = append(build.Jobs, item.Job)
		store.CreateJob(c, item.Job)
		// TODO err
	}

	//
	// publish topic
	//
	message := pubsub.Message{
		Labels: map[string]string{
			"repo":    repo.FullName,
			"private": strconv.FormatBool(repo.IsPrivate),
		},
	}
	message.Data, _ = json.Marshal(model.Event{
		Type:  model.Enqueued,
		Repo:  *repo,
		Build: *build,
	})
	// TODO remove global reference
	config.pubsub.Publish(c, "topic/events", message)
	//
	// end publish topic
	//

	for _, item := range items {
		task := new(queue.Task)
		task.ID = fmt.Sprint(item.Job.ID)
		task.Labels = map[string]string{}
		task.Labels["platform"] = item.Platform
		for k, v := range item.Labels {
			task.Labels[k] = v
		}

		task.Data, _ = json.Marshal(rpc.Pipeline{
			ID:      fmt.Sprint(item.Job.ID),
			Config:  item.Config,
			Timeout: b.Repo.Timeout,
		})

		config.logger.Open(context.Background(), task.ID)
		config.queue.Push(context.Background(), task)
	}
}
