package server

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

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

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GetQueueInfo(c *gin.Context) {
	c.IndentedJSON(200,
		Config.Services.Queue.Info(c),
	)
}

func PostHook(c *gin.Context) {
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
	// the build.
	if refresher, ok := remote_.(remote.Refresher); ok {
		ok, _ := refresher.Refresh(user)
		if ok {
			store.UpdateUser(c, user)
		}
	}

	// fetch the build file from the database
	confb, err := remote_.File(user, repo, build, repo.Config)
	if err != nil {
		logrus.Errorf("failure to get build config for %s. %s", repo.FullName, err)
		c.AbortWithError(404, err)
		return
	}
	sha := shasum(confb)
	conf, err := Config.Storage.Config.ConfigFind(repo, sha)
	if err != nil {
		conf = &model.Config{
			RepoID: repo.ID,
			Data:   string(confb),
			Hash:   sha,
		}
		err = Config.Storage.Config.ConfigCreate(conf)
		if err != nil {
			logrus.Errorf("failure to persist config for %s. %s", repo.FullName, err)
			c.AbortWithError(500, err)
			return
		}
	}
	build.ConfigID = conf.ID

	netrc, err := remote_.Netrc(user, repo)
	if err != nil {
		c.String(500, "Failed to generate netrc file. %s", err)
		return
	}

	// verify the branches can be built vs skipped
	branches, err := yaml.ParseString(conf.Data)
	if err == nil {
		if !branches.Branches.Match(build.Branch) && build.Event != model.EventTag && build.Event != model.EventDeploy {
			c.String(200, "Branch does not match restrictions defined in yaml")
			return
		}
	}

	// update some build fields
	build.RepoID = repo.ID
	build.Verified = true
	build.Status = model.StatusPending

	if repo.IsGated {
		allowed, _ := Config.Services.Senders.SenderAllowed(user, repo, build, conf)
		if !allowed {
			build.Status = model.StatusBlocked
		}
	}

	build.Trim()
	err = store.CreateBuild(c, build, build.Procs...)
	if err != nil {
		logrus.Errorf("failure to save commit for %s. %s", repo.FullName, err)
		c.AbortWithError(500, err)
		return
	}

	c.JSON(200, build)

	if build.Status == model.StatusBlocked {
		return
	}

	envs := map[string]string{}
	if Config.Services.Environ != nil {
		globals, _ := Config.Services.Environ.EnvironList(repo)
		for _, global := range globals {
			envs[global.Name] = global.Value
		}
	}

	secs, err := Config.Services.Secrets.SecretListBuild(repo, build)
	if err != nil {
		logrus.Debugf("Error getting secrets for %s#%d. %s", repo.FullName, build.Number, err)
	}

	regs, err := Config.Services.Registries.RegistryList(repo)
	if err != nil {
		logrus.Debugf("Error getting registry credentials for %s#%d. %s", repo.FullName, build.Number, err)
	}

	// get the previous build so that we can send
	// on status change notifications
	last, _ := store.GetBuildLastBefore(c, repo, build.Branch, build.ID)

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
		Regs:  regs,
		Envs:  envs,
		Link:  httputil.GetURL(c.Request),
		Yaml:  conf.Data,
	}
	items, err := b.Build()
	if err != nil {
		build.Status = model.StatusError
		build.Started = time.Now().Unix()
		build.Finished = build.Started
		build.Error = err.Error()
		store.UpdateBuild(c, build)
		return
	}

	var pcounter = len(items)

	for _, item := range items {
		build.Procs = append(build.Procs, item.Proc)
		item.Proc.BuildID = build.ID

		for _, stage := range item.Config.Stages {
			var gid int
			for _, step := range stage.Steps {
				pcounter++
				if gid == 0 {
					gid = pcounter
				}
				proc := &model.Proc{
					BuildID: build.ID,
					Name:    step.Alias,
					PID:     pcounter,
					PPID:    item.Proc.PID,
					PGID:    gid,
					State:   model.StatusPending,
				}
				build.Procs = append(build.Procs, proc)
			}
		}
	}
	err = store.FromContext(c).ProcCreate(build.Procs)
	if err != nil {
		logrus.Errorf("error persisting procs %s/%d: %s", repo.FullName, build.Number, err)
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
	buildCopy := *build
	buildCopy.Procs = model.Tree(buildCopy.Procs)
	message.Data, _ = json.Marshal(model.Event{
		Type:  model.Enqueued,
		Repo:  *repo,
		Build: buildCopy,
	})
	// TODO remove global reference
	Config.Services.Pubsub.Publish(c, "topic/events", message)
	//
	// end publish topic
	//

	for _, item := range items {
		task := new(queue.Task)
		task.ID = fmt.Sprint(item.Proc.ID)
		task.Labels = map[string]string{}
		task.Labels["platform"] = item.Platform
		for k, v := range item.Labels {
			task.Labels[k] = v
		}

		task.Data, _ = json.Marshal(rpc.Pipeline{
			ID:      fmt.Sprint(item.Proc.ID),
			Config:  item.Config,
			Timeout: b.Repo.Timeout,
		})

		Config.Services.Logs.Open(context.Background(), task.ID)
		Config.Services.Queue.Push(context.Background(), task)
	}
}

// return the metadata from the cli context.
func metadataFromStruct(repo *model.Repo, build, last *model.Build, proc *model.Proc, link string) frontend.Metadata {
	return frontend.Metadata{
		Repo: frontend.Repo{
			Name:    repo.FullName,
			Link:    repo.Link,
			Remote:  repo.Clone,
			Private: repo.IsPrivate,
		},
		Curr: frontend.Build{
			Number:   build.Number,
			Parent:   build.Parent,
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
			Number: proc.PID,
			Matrix: proc.Environ,
		},
		Sys: frontend.System{
			Name: "drone",
			Link: link,
			Arch: "linux/amd64",
		},
	}
}

type builder struct {
	Repo  *model.Repo
	Curr  *model.Build
	Last  *model.Build
	Netrc *model.Netrc
	Secs  []*model.Secret
	Regs  []*model.Registry
	Link  string
	Yaml  string
	Envs  map[string]string
}

type buildItem struct {
	Proc     *model.Proc
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
		proc := &model.Proc{
			BuildID: b.Curr.ID,
			PID:     i + 1,
			PGID:    i + 1,
			State:   model.StatusPending,
			Environ: axis,
		}

		metadata := metadataFromStruct(b.Repo, b.Curr, b.Last, proc, b.Link)
		environ := metadata.Environ()
		for k, v := range metadata.EnvironDrone() {
			environ[k] = v
		}
		for k, v := range axis {
			environ[k] = v
		}

		var secrets []compiler.Secret
		for _, sec := range b.Secs {
			if !sec.Match(b.Curr.Event) {
				continue
			}
			secrets = append(secrets, compiler.Secret{
				Name:  sec.Name,
				Value: sec.Value,
				Match: sec.Images,
			})
		}

		y := b.Yaml
		s, err := envsubst.Eval(y, func(name string) string {
			return environ[name]
		})
		if err != nil {
			return nil, err
		}
		y = s

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
			return nil, lerr
		}

		var registries []compiler.Registry
		for _, reg := range b.Regs {
			registries = append(registries, compiler.Registry{
				Hostname: reg.Address,
				Username: reg.Username,
				Password: reg.Password,
				Email:    reg.Email,
			})
		}

		ir := compiler.New(
			compiler.WithEnviron(environ),
			compiler.WithEnviron(b.Envs),
			compiler.WithEscalated(Config.Pipeline.Privileged...),
			compiler.WithResourceLimit(Config.Pipeline.Limits.MemSwapLimit, Config.Pipeline.Limits.MemLimit, Config.Pipeline.Limits.ShmSize, Config.Pipeline.Limits.CPUQuota, Config.Pipeline.Limits.CPUShares, Config.Pipeline.Limits.CPUSet),
			compiler.WithVolumes(Config.Pipeline.Volumes...),
			compiler.WithNetworks(Config.Pipeline.Networks...),
			compiler.WithLocal(false),
			compiler.WithOption(
				compiler.WithNetrc(
					b.Netrc.Login,
					b.Netrc.Password,
					b.Netrc.Machine,
				),
				b.Repo.IsPrivate,
			),
			compiler.WithRegistry(registries...),
			compiler.WithSecret(secrets...),
			compiler.WithPrefix(
				fmt.Sprintf(
					"%d_%d",
					proc.ID,
					rand.Int(),
				),
			),
			compiler.WithEnviron(proc.Environ),
			compiler.WithProxy(),
			compiler.WithWorkspaceFromURL("/drone", b.Curr.Link),
			compiler.WithMetadata(metadata),
		).Compile(parsed)

		// for _, sec := range b.Secs {
		// 	if !sec.MatchEvent(b.Curr.Event) {
		// 		continue
		// 	}
		// 	if b.Curr.Verified || sec.SkipVerify {
		// 		ir.Secrets = append(ir.Secrets, &backend.Secret{
		// 			Mask:  sec.Conceal,
		// 			Name:  sec.Name,
		// 			Value: sec.Value,
		// 		})
		// 	}
		// }

		item := &buildItem{
			Proc:     proc,
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

func shasum(raw []byte) string {
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum)
}
