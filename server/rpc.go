package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	oldcontext "golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/cncd/logging"
	"github.com/cncd/pipeline/pipeline/rpc"
	"github.com/cncd/pipeline/pipeline/rpc/proto"
	"github.com/cncd/pubsub"
	"github.com/cncd/queue"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/store"
)

// This file is a complete disaster because I'm trying to wedge in some
// experimental code. Please pardon our appearance during renovations.

// Config is an evil global configuration that will be used as we transition /
// refactor the codebase to move away from storing these values in the Context.
var Config = struct {
	Services struct {
		Pubsub     pubsub.Publisher
		Queue      queue.Queue
		Logs       logging.Log
		Senders    model.SenderService
		Secrets    model.SecretService
		Registries model.RegistryService
		Environ    model.EnvironService
	}
	Storage struct {
		// Users  model.UserStore
		// Repos  model.RepoStore
		// Builds model.BuildStore
		// Logs   model.LogStore
		Config model.ConfigStore
		Files  model.FileStore
		Procs  model.ProcStore
		// Registries model.RegistryStore
		// Secrets model.SecretStore
	}
	Server struct {
		Key  string
		Cert string
		Host string
		Port string
		Pass string
		// Open bool
		// Orgs map[string]struct{}
		// Admins map[string]struct{}
	}
	Pipeline struct {
		Limits     model.ResourceLimit
		Volumes    []string
		Networks   []string
		Privileged []string
	}
}{}

type RPC struct {
	remote remote.Remote
	queue  queue.Queue
	pubsub pubsub.Publisher
	logger logging.Log
	store  store.Store
	host   string
}

// Next implements the rpc.Next function
func (s *RPC) Next(c context.Context, filter rpc.Filter) (*rpc.Pipeline, error) {
	fn := func(task *queue.Task) bool {
		for k, v := range filter.Labels {
			if task.Labels[k] != v {
				return false
			}
		}
		return true
	}
	task, err := s.queue.Poll(c, fn)
	if err != nil {
		return nil, err
	} else if task == nil {
		return nil, nil
	}
	pipeline := new(rpc.Pipeline)

	// check if the process was previously cancelled
	// cancelled, _ := s.checkCancelled(pipeline)
	// if cancelled {
	// 	logrus.Debugf("ignore pid %v: cancelled by user", pipeline.ID)
	// 	if derr := s.queue.Done(c, pipeline.ID); derr != nil {
	// 		logrus.Errorf("error: done: cannot ack proc_id %v: %s", pipeline.ID, err)
	// 	}
	// 	return nil, nil
	// }

	err = json.Unmarshal(task.Data, pipeline)
	return pipeline, err
}

// Wait implements the rpc.Wait function
func (s *RPC) Wait(c context.Context, id string) error {
	return s.queue.Wait(c, id)
}

// Extend implements the rpc.Extend function
func (s *RPC) Extend(c context.Context, id string) error {
	return s.queue.Extend(c, id)
}

// Update implements the rpc.Update function
func (s *RPC) Update(c context.Context, id string, state rpc.State) error {
	procID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}

	pproc, err := s.store.ProcLoad(procID)
	if err != nil {
		log.Printf("error: rpc.update: cannot find pproc with id %d: %s", procID, err)
		return err
	}

	build, err := s.store.GetBuild(pproc.BuildID)
	if err != nil {
		log.Printf("error: cannot find build with id %d: %s", pproc.BuildID, err)
		return err
	}

	proc, err := s.store.ProcChild(build, pproc.PID, state.Proc)
	if err != nil {
		log.Printf("error: cannot find proc with name %s: %s", state.Proc, err)
		return err
	}

	repo, err := s.store.GetRepo(build.RepoID)
	if err != nil {
		log.Printf("error: cannot find repo with id %d: %s", build.RepoID, err)
		return err
	}

	if state.Exited {
		proc.Stopped = state.Finished
		proc.ExitCode = state.ExitCode
		proc.Error = state.Error
		proc.State = model.StatusSuccess
		if state.ExitCode != 0 || state.Error != "" {
			proc.State = model.StatusFailure
		}
	} else {
		proc.Started = state.Started
		proc.State = model.StatusRunning
	}

	if err := s.store.ProcUpdate(proc); err != nil {
		log.Printf("error: rpc.update: cannot update proc: %s", err)
	}

	build.Procs, _ = s.store.ProcList(build)
	build.Procs = model.Tree(build.Procs)
	message := pubsub.Message{
		Labels: map[string]string{
			"repo":    repo.FullName,
			"private": strconv.FormatBool(repo.IsPrivate),
		},
	}
	message.Data, _ = json.Marshal(model.Event{
		Repo:  *repo,
		Build: *build,
	})
	s.pubsub.Publish(c, "topic/events", message)

	return nil
}

// Upload implements the rpc.Upload function
func (s *RPC) Upload(c context.Context, id string, file *rpc.File) error {
	procID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}

	pproc, err := s.store.ProcLoad(procID)
	if err != nil {
		log.Printf("error: cannot find parent proc with id %d: %s", procID, err)
		return err
	}

	build, err := s.store.GetBuild(pproc.BuildID)
	if err != nil {
		log.Printf("error: cannot find build with id %d: %s", pproc.BuildID, err)
		return err
	}

	proc, err := s.store.ProcChild(build, pproc.PID, file.Proc)
	if err != nil {
		log.Printf("error: cannot find child proc with name %s: %s", file.Proc, err)
		return err
	}

	if file.Mime == "application/json+logs" {
		return s.store.LogSave(
			proc,
			bytes.NewBuffer(file.Data),
		)
	}

	return Config.Storage.Files.FileCreate(&model.File{
		BuildID: proc.BuildID,
		ProcID:  proc.ID,
		Mime:    file.Mime,
		Name:    file.Name,
		Size:    file.Size,
		Time:    file.Time,
	},
		bytes.NewBuffer(file.Data),
	)
}

// Init implements the rpc.Init function
func (s *RPC) Init(c context.Context, id string, state rpc.State) error {
	procID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}

	proc, err := s.store.ProcLoad(procID)
	if err != nil {
		log.Printf("error: cannot find proc with id %d: %s", procID, err)
		return err
	}

	build, err := s.store.GetBuild(proc.BuildID)
	if err != nil {
		log.Printf("error: cannot find build with id %d: %s", proc.BuildID, err)
		return err
	}

	repo, err := s.store.GetRepo(build.RepoID)
	if err != nil {
		log.Printf("error: cannot find repo with id %d: %s", build.RepoID, err)
		return err
	}

	if build.Status == model.StatusPending {
		build.Status = model.StatusRunning
		build.Started = state.Started
		if err := s.store.UpdateBuild(build); err != nil {
			log.Printf("error: init: cannot update build_id %d state: %s", build.ID, err)
		}
	}

	defer func() {
		build.Procs, _ = s.store.ProcList(build)
		message := pubsub.Message{
			Labels: map[string]string{
				"repo":    repo.FullName,
				"private": strconv.FormatBool(repo.IsPrivate),
			},
		}
		message.Data, _ = json.Marshal(model.Event{
			Repo:  *repo,
			Build: *build,
		})
		s.pubsub.Publish(c, "topic/events", message)
	}()

	proc.Started = state.Started
	proc.State = model.StatusRunning
	return s.store.ProcUpdate(proc)
}

// Done implements the rpc.Done function
func (s *RPC) Done(c context.Context, id string, state rpc.State) error {
	procID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}

	proc, err := s.store.ProcLoad(procID)
	if err != nil {
		log.Printf("error: cannot find proc with id %d: %s", procID, err)
		return err
	}

	build, err := s.store.GetBuild(proc.BuildID)
	if err != nil {
		log.Printf("error: cannot find build with id %d: %s", proc.BuildID, err)
		return err
	}

	repo, err := s.store.GetRepo(build.RepoID)
	if err != nil {
		log.Printf("error: cannot find repo with id %d: %s", build.RepoID, err)
		return err
	}

	proc.Stopped = state.Finished
	proc.Error = state.Error
	proc.ExitCode = state.ExitCode
	proc.State = model.StatusSuccess
	if proc.ExitCode != 0 || proc.Error != "" {
		proc.State = model.StatusFailure
	}
	if err := s.store.ProcUpdate(proc); err != nil {
		log.Printf("error: done: cannot update proc_id %d state: %s", procID, err)
	}

	if err := s.queue.Done(c, id); err != nil {
		log.Printf("error: done: cannot ack proc_id %d: %s", procID, err)
	}

	// TODO handle this error
	procs, _ := s.store.ProcList(build)
	for _, p := range procs {
		if p.Running() && p.PPID == proc.PID {
			p.State = model.StatusSkipped
			if p.Started != 0 {
				p.State = model.StatusSuccess // for deamons that are killed
				p.Stopped = proc.Stopped
			}
			if err := s.store.ProcUpdate(p); err != nil {
				log.Printf("error: done: cannot update proc_id %d child state: %s", p.ID, err)
			}
		}
	}

	running := false
	status := model.StatusSuccess
	for _, p := range procs {
		if p.PPID == 0 {
			if p.Running() {
				running = true
			}
			if p.Failing() {
				status = p.State
			}
		}
	}
	if !running {
		build.Status = status
		build.Finished = proc.Stopped
		if err := s.store.UpdateBuild(build); err != nil {
			log.Printf("error: done: cannot update build_id %d final state: %s", build.ID, err)
		}

		// update the status
		user, err := s.store.GetUser(repo.UserID)
		if err == nil {
			if refresher, ok := s.remote.(remote.Refresher); ok {
				ok, _ := refresher.Refresh(user)
				if ok {
					s.store.UpdateUser(user)
				}
			}
			uri := fmt.Sprintf("%s/%s/%d", s.host, repo.FullName, build.Number)
			err = s.remote.Status(user, repo, build, uri)
			if err != nil {
				logrus.Errorf("error setting commit status for %s/%d", repo.FullName, build.Number)
			}
		}
	}

	if err := s.logger.Close(c, id); err != nil {
		log.Printf("error: done: cannot close build_id %d logger: %s", proc.ID, err)
	}

	build.Procs = model.Tree(procs)
	message := pubsub.Message{
		Labels: map[string]string{
			"repo":    repo.FullName,
			"private": strconv.FormatBool(repo.IsPrivate),
		},
	}
	message.Data, _ = json.Marshal(model.Event{
		Repo:  *repo,
		Build: *build,
	})
	s.pubsub.Publish(c, "topic/events", message)

	return nil
}

// Log implements the rpc.Log function
func (s *RPC) Log(c context.Context, id string, line *rpc.Line) error {
	entry := new(logging.Entry)
	entry.Data, _ = json.Marshal(line)
	s.logger.Write(c, id, entry)
	return nil
}

func (s *RPC) checkCancelled(pipeline *rpc.Pipeline) (bool, error) {
	pid, err := strconv.ParseInt(pipeline.ID, 10, 64)
	if err != nil {
		return false, err
	}
	proc, err := s.store.ProcLoad(pid)
	if err != nil {
		return false, err
	}
	if proc.State == model.StatusKilled {
		return true, nil
	}
	return false, err
}

//
//
//

// DroneServer is a grpc server implementation.
type DroneServer struct {
	Remote remote.Remote
	Queue  queue.Queue
	Pubsub pubsub.Publisher
	Logger logging.Log
	Store  store.Store
	Host   string
}

func (s *DroneServer) Next(c oldcontext.Context, req *proto.NextRequest) (*proto.NextReply, error) {
	peer := RPC{
		remote: s.Remote,
		store:  s.Store,
		queue:  s.Queue,
		pubsub: s.Pubsub,
		logger: s.Logger,
		host:   s.Host,
	}
	filter := rpc.Filter{
		Labels: req.GetFilter().GetLabels(),
	}

	res := new(proto.NextReply)
	pipeline, err := peer.Next(c, filter)
	if err != nil {
		return res, err
	}
	if pipeline == nil {
		return res, err
	}

	res.Pipeline = new(proto.Pipeline)
	res.Pipeline.Id = pipeline.ID
	res.Pipeline.Timeout = pipeline.Timeout
	res.Pipeline.Payload, _ = json.Marshal(pipeline.Config)

	return res, err

	// fn := func(task *queue.Task) bool {
	// 	for k, v := range req.GetFilter().Labels {
	// 		if task.Labels[k] != v {
	// 			return false
	// 		}
	// 	}
	// 	return true
	// }
	// task, err := s.Queue.Poll(c, fn)
	// if err != nil {
	// 	return nil, err
	// } else if task == nil {
	// 	return nil, nil
	// }
	//
	// pipeline := new(rpc.Pipeline)
	// json.Unmarshal(task.Data, pipeline)
	//
	// res := new(proto.NextReply)
	// res.Pipeline = new(proto.Pipeline)
	// res.Pipeline.Id = pipeline.ID
	// res.Pipeline.Timeout = pipeline.Timeout
	// res.Pipeline.Payload, _ = json.Marshal(pipeline.Config)
	//
	// // check if the process was previously cancelled
	// // cancelled, _ := s.checkCancelled(pipeline)
	// // if cancelled {
	// // 	logrus.Debugf("ignore pid %v: cancelled by user", pipeline.ID)
	// // 	if derr := s.queue.Done(c, pipeline.ID); derr != nil {
	// // 		logrus.Errorf("error: done: cannot ack proc_id %v: %s", pipeline.ID, err)
	// // 	}
	// // 	return nil, nil
	// // }
	//
	// return res, nil
}

func (s *DroneServer) Init(c oldcontext.Context, req *proto.InitRequest) (*proto.Empty, error) {
	peer := RPC{
		remote: s.Remote,
		store:  s.Store,
		queue:  s.Queue,
		pubsub: s.Pubsub,
		logger: s.Logger,
		host:   s.Host,
	}
	state := rpc.State{
		Error:    req.GetState().GetError(),
		ExitCode: int(req.GetState().GetExitCode()),
		Finished: req.GetState().GetFinished(),
		Started:  req.GetState().GetStarted(),
		Proc:     req.GetState().GetName(),
		Exited:   req.GetState().GetExited(),
	}
	res := new(proto.Empty)
	err := peer.Init(c, req.GetId(), state)
	return res, err
}

func (s *DroneServer) Update(c oldcontext.Context, req *proto.UpdateRequest) (*proto.Empty, error) {
	peer := RPC{
		remote: s.Remote,
		store:  s.Store,
		queue:  s.Queue,
		pubsub: s.Pubsub,
		logger: s.Logger,
		host:   s.Host,
	}
	state := rpc.State{
		Error:    req.GetState().GetError(),
		ExitCode: int(req.GetState().GetExitCode()),
		Finished: req.GetState().GetFinished(),
		Started:  req.GetState().GetStarted(),
		Proc:     req.GetState().GetName(),
		Exited:   req.GetState().GetExited(),
	}
	res := new(proto.Empty)
	err := peer.Update(c, req.GetId(), state)
	return res, err
}

func (s *DroneServer) Upload(c oldcontext.Context, req *proto.UploadRequest) (*proto.Empty, error) {
	peer := RPC{
		remote: s.Remote,
		store:  s.Store,
		queue:  s.Queue,
		pubsub: s.Pubsub,
		logger: s.Logger,
		host:   s.Host,
	}
	file := &rpc.File{
		Data: req.GetFile().GetData(),
		Mime: req.GetFile().GetMime(),
		Name: req.GetFile().GetName(),
		Proc: req.GetFile().GetProc(),
		Size: int(req.GetFile().GetSize()),
		Time: req.GetFile().GetTime(),
	}
	res := new(proto.Empty)
	err := peer.Upload(c, req.GetId(), file)
	return res, err
}

func (s *DroneServer) Done(c oldcontext.Context, req *proto.DoneRequest) (*proto.Empty, error) {
	peer := RPC{
		remote: s.Remote,
		store:  s.Store,
		queue:  s.Queue,
		pubsub: s.Pubsub,
		logger: s.Logger,
		host:   s.Host,
	}
	state := rpc.State{
		Error:    req.GetState().GetError(),
		ExitCode: int(req.GetState().GetExitCode()),
		Finished: req.GetState().GetFinished(),
		Started:  req.GetState().GetStarted(),
		Proc:     req.GetState().GetName(),
		Exited:   req.GetState().GetExited(),
	}
	res := new(proto.Empty)
	err := peer.Done(c, req.GetId(), state)
	return res, err
}

func (s *DroneServer) Wait(c oldcontext.Context, req *proto.WaitRequest) (*proto.Empty, error) {
	peer := RPC{
		remote: s.Remote,
		store:  s.Store,
		queue:  s.Queue,
		pubsub: s.Pubsub,
		logger: s.Logger,
		host:   s.Host,
	}
	res := new(proto.Empty)
	err := peer.Wait(c, req.GetId())
	return res, err
}

func (s *DroneServer) Extend(c oldcontext.Context, req *proto.ExtendRequest) (*proto.Empty, error) {
	peer := RPC{
		remote: s.Remote,
		store:  s.Store,
		queue:  s.Queue,
		pubsub: s.Pubsub,
		logger: s.Logger,
		host:   s.Host,
	}
	res := new(proto.Empty)
	err := peer.Extend(c, req.GetId())
	return res, err
}

func (s *DroneServer) Log(c oldcontext.Context, req *proto.LogRequest) (*proto.Empty, error) {
	peer := RPC{
		remote: s.Remote,
		store:  s.Store,
		queue:  s.Queue,
		pubsub: s.Pubsub,
		logger: s.Logger,
		host:   s.Host,
	}
	line := &rpc.Line{
		Out:  req.GetLine().GetOut(),
		Pos:  int(req.GetLine().GetPos()),
		Time: req.GetLine().GetTime(),
		Proc: req.GetLine().GetProc(),
	}
	res := new(proto.Empty)
	err := peer.Log(c, req.GetId(), line)
	return res, err
}
