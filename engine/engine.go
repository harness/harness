package engine

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"runtime"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/drone/drone/model"
	"github.com/drone/drone/shared/docker"
	"github.com/drone/drone/shared/envconfig"
	"github.com/drone/drone/store"
	"github.com/samalba/dockerclient"
	"golang.org/x/net/context"
)

type Engine interface {
	Schedule(context.Context, *Task)
	Cancel(int64, int64, *model.Node) error
	Stream(int64, int64, *model.Node) (io.ReadCloser, error)
	Deallocate(*model.Node)
	Allocate(*model.Node) error
	Subscribe(chan *Event)
	Unsubscribe(chan *Event)
}

var (
	// options to fetch the stdout and stderr logs
	logOpts = &dockerclient.LogOptions{
		Stdout: true,
		Stderr: true,
	}

	// options to fetch the stdout and stderr logs
	// by tailing the output.
	logOptsTail = &dockerclient.LogOptions{
		Follow: true,
		Stdout: true,
		Stderr: true,
	}

	// error when the system cannot find logs
	errLogging = errors.New("Logs not available")
)

type engine struct {
	bus     *eventbus
	updater *updater
	pool    *pool
	envs    []string
}

// Load creates a new build engine, loaded with registered nodes from the
// database. The registered nodes are added to the pool of nodes to immediately
// start accepting workloads.
func Load(env envconfig.Env, s store.Store) Engine {
	engine := &engine{}
	engine.bus = newEventbus()
	engine.pool = newPool()
	engine.updater = &updater{engine.bus}

	// quick fix to propagate HTTP_PROXY variables
	// throughout the build environment.
	var proxyVars = []string{"HTTP_PROXY", "http_proxy", "HTTPS_PROXY", "https_proxy", "NO_PROXY", "no_proxy"}
	for _, proxyVar := range proxyVars {
		proxyVal := env.Get(proxyVar)
		if len(proxyVal) != 0 {
			engine.envs = append(engine.envs, proxyVar+"="+proxyVal)
		}
	}

	nodes, err := s.GetNodeList()
	if err != nil {
		log.Fatalf("failed to get nodes from database. %s", err)
	}
	for _, node := range nodes {
		engine.pool.allocate(node)
		log.Infof("registered docker daemon %s", node.Addr)
	}

	return engine
}

// Cancel cancels the job running on the specified Node.
func (e *engine) Cancel(build, job int64, node *model.Node) error {
	client, err := newDockerClient(node.Addr, node.Cert, node.Key, node.CA)
	if err != nil {
		return err
	}

	id := fmt.Sprintf("drone_build_%d_job_%d", build, job)
	return client.StopContainer(id, 30)
}

// Stream streams the job output from the specified Node.
func (e *engine) Stream(build, job int64, node *model.Node) (io.ReadCloser, error) {
	client, err := newDockerClient(node.Addr, node.Cert, node.Key, node.CA)
	if err != nil {
		log.Errorf("cannot create Docker client for node %s", node.Addr)
		return nil, err
	}

	id := fmt.Sprintf("drone_build_%d_job_%d", build, job)
	log.Debugf("streaming container logs %s", id)
	return client.ContainerLogs(id, logOptsTail)
}

// Subscribe subscribes the channel to all build events.
func (e *engine) Subscribe(c chan *Event) {
	e.bus.subscribe(c)
}

// Unsubscribe unsubscribes the channel from all build events.
func (e *engine) Unsubscribe(c chan *Event) {
	e.bus.unsubscribe(c)
}

func (e *engine) Allocate(node *model.Node) error {

	// run the full build!
	client, err := newDockerClient(node.Addr, node.Cert, node.Key, node.CA)
	if err != nil {
		log.Errorf("error creating docker client %s. %s.", node.Addr, err)
		return err
	}
	version, err := client.Version()
	if err != nil {
		log.Errorf("error connecting to docker daemon %s. %s.", node.Addr, err)
		return err
	}

	log.Infof("registered docker daemon %s running version %s", node.Addr, version.Version)
	e.pool.allocate(node)
	return nil
}

func (e *engine) Deallocate(n *model.Node) {
	nodes := e.pool.list()
	for _, node := range nodes {
		if node.ID == n.ID {
			log.Infof("un-registered docker daemon %s", node.Addr)
			e.pool.deallocate(node)
			break
		}
	}
}

func (e *engine) Schedule(c context.Context, req *Task) {
	node := <-e.pool.reserve()

	// since we are probably running in a go-routine
	// make sure we recover from any panics so that
	// a bug doesn't crash the whole system.
	defer func() {
		if err := recover(); err != nil {

			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			log.Errorf("panic running build: %v\n%s", err, string(buf))
		}
		e.pool.release(node)
	}()

	// update the node that was allocated to each job
	func(id int64) {
		for _, job := range req.Jobs {
			job.NodeID = id
			store.UpdateJob(c, job)
		}
	}(node.ID)

	// run the full build!
	client, err := newDockerClient(node.Addr, node.Cert, node.Key, node.CA)
	if err != nil {
		log.Errorln("error creating docker client", err)
	}

	// update the build state if any of the sub-tasks
	// had a non-success status
	req.Build.Started = time.Now().UTC().Unix()
	req.Build.Status = model.StatusRunning
	e.updater.SetBuild(c, req)

	// run all bulid jobs
	for _, job := range req.Jobs {
		req.Job = job
		e.runJob(c, req, e.updater, client)
	}

	// update overall status based on each job
	req.Build.Status = model.StatusSuccess
	for _, job := range req.Jobs {
		if job.Status != model.StatusSuccess {
			req.Build.Status = job.Status
			break
		}
	}
	req.Build.Finished = time.Now().UTC().Unix()
	err = e.updater.SetBuild(c, req)
	if err != nil {
		log.Errorf("error updating build completion status. %s", err)
	}

	// run notifications
	err = e.runJobNotify(req, client)
	if err != nil {
		log.Errorf("error executing notification step. %s", err)
	}
}

func newDockerClient(addr, cert, key, ca string) (dockerclient.Client, error) {
	var tlc *tls.Config

	// create the Docket client TLS config
	if len(cert) != 0 {
		pem, err := tls.X509KeyPair([]byte(cert), []byte(key))
		if err != nil {
			log.Errorf("error loading X509 key pair. %s.", err)
			return dockerclient.NewDockerClient(addr, nil)
		}

		// create the TLS configuration for secure
		// docker communications.
		tlc = &tls.Config{}
		tlc.Certificates = []tls.Certificate{pem}

		// use the certificate authority if provided.
		// else don't use a certificate authority and set
		// skip verify to true
		if len(ca) != 0 {
			log.Infof("creating docker client %s with CA", addr)
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM([]byte(ca))
			tlc.RootCAs = pool

		} else {
			log.Infof("creating docker client %s WITHOUT CA", addr)
			tlc.InsecureSkipVerify = true
		}
	}

	// create the Docker client. In this version of Drone (alpha)
	// we do not spread builds across clients, but this can and
	// (probably) will change in the future.
	return dockerclient.NewDockerClient(addr, tlc)
}

func (e *engine) runJob(c context.Context, r *Task, updater *updater, client dockerclient.Client) error {

	name := fmt.Sprintf("drone_build_%d_job_%d", r.Build.ID, r.Job.ID)

	defer func() {
		if r.Job.Status == model.StatusRunning {
			r.Job.Status = model.StatusError
			r.Job.Finished = time.Now().UTC().Unix()
			r.Job.ExitCode = 255
		}
		if r.Job.Status == model.StatusPending {
			r.Job.Status = model.StatusError
			r.Job.Started = time.Now().UTC().Unix()
			r.Job.Finished = time.Now().UTC().Unix()
			r.Job.ExitCode = 255
		}
		updater.SetJob(c, r)

		client.KillContainer(name, "9")
		client.RemoveContainer(name, true, true)
	}()

	// marks the task as running
	r.Job.Status = model.StatusRunning
	r.Job.Started = time.Now().UTC().Unix()

	// encode the build payload to write to stdin
	// when launching the build container
	in, err := encodeToLegacyFormat(r)
	if err != nil {
		log.Errorf("failure to marshal work. %s", err)
		return err
	}

	// CREATE AND START BUILD
	args := DefaultBuildArgs
	if r.Build.Event == model.EventPull {
		args = DefaultPullRequestArgs
	}
	args = append(args, "--")
	args = append(args, string(in))

	conf := &dockerclient.ContainerConfig{
		Image:      DefaultAgent,
		Entrypoint: DefaultEntrypoint,
		Cmd:        args,
		Env:        e.envs,
		HostConfig: dockerclient.HostConfig{
			Binds:            []string{"/var/run/docker.sock:/var/run/docker.sock"},
			MemorySwappiness: -1,
		},
		Volumes: map[string]struct{}{
			"/var/run/docker.sock": {},
		},
	}

	log.Infof("preparing container %s", name)
	client.PullImage(conf.Image, nil)

	_, err = docker.RunDaemon(client, conf, name)
	if err != nil {
		log.Errorf("error starting build container. %s", err)
		return err
	}

	// UPDATE STATUS

	err = updater.SetJob(c, r)
	if err != nil {
		log.Errorf("error updating job status as running. %s", err)
		return err
	}

	// WAIT FOR OUTPUT
	info, builderr := docker.Wait(client, name)

	switch {
	case info.State.Running:
		// A build unblocked before actually being completed.
		log.Errorf("incomplete build: %s", name)
		r.Job.ExitCode = 1
		r.Job.Status = model.StatusError
	case info.State.ExitCode == 128:
		r.Job.ExitCode = info.State.ExitCode
		r.Job.Status = model.StatusKilled
	case info.State.ExitCode == 130:
		r.Job.ExitCode = info.State.ExitCode
		r.Job.Status = model.StatusKilled
	case builderr != nil:
		r.Job.Status = model.StatusError
	case info.State.ExitCode != 0:
		r.Job.ExitCode = info.State.ExitCode
		r.Job.Status = model.StatusFailure
	default:
		r.Job.Status = model.StatusSuccess
	}

	// send the logs to the datastore
	var buf bytes.Buffer
	rc, err := client.ContainerLogs(name, docker.LogOpts)
	if err != nil && builderr != nil {
		buf.WriteString("Error launching build")
		buf.WriteString(builderr.Error())
	} else if err != nil {
		buf.WriteString("Error launching build")
		buf.WriteString(err.Error())
		log.Errorf("error opening connection to logs. %s", err)
		return err
	} else {
		defer rc.Close()
		stdcopy.StdCopy(&buf, &buf, io.LimitReader(rc, 5000000))
	}

	// update the task in the datastore
	r.Job.Finished = time.Now().UTC().Unix()
	err = updater.SetJob(c, r)
	if err != nil {
		log.Errorf("error updating job after completion. %s", err)
		return err
	}

	err = updater.SetLogs(c, r, ioutil.NopCloser(&buf))
	if err != nil {
		log.Errorf("error updating logs. %s", err)
		return err
	}

	log.Debugf("completed job %d with status %s.", r.Job.ID, r.Job.Status)
	return nil
}

func (e *engine) runJobNotify(r *Task, client dockerclient.Client) error {

	name := fmt.Sprintf("drone_build_%d_notify", r.Build.ID)

	defer func() {
		client.KillContainer(name, "9")
		client.RemoveContainer(name, true, true)
	}()

	// encode the build payload to write to stdin
	// when launching the build container
	in, err := encodeToLegacyFormat(r)
	if err != nil {
		log.Errorf("failure to marshal work. %s", err)
		return err
	}

	args := DefaultNotifyArgs
	args = append(args, "--")
	args = append(args, string(in))

	conf := &dockerclient.ContainerConfig{
		Image:      DefaultAgent,
		Entrypoint: DefaultEntrypoint,
		Cmd:        args,
		Env:        e.envs,
		HostConfig: dockerclient.HostConfig{
			Binds:            []string{"/var/run/docker.sock:/var/run/docker.sock"},
			MemorySwappiness: -1,
		},
		Volumes: map[string]struct{}{
			"/var/run/docker.sock": {},
		},
	}

	log.Infof("preparing container %s", name)
	info, err := docker.Run(client, conf, name)
	if err != nil {
		log.Errorf("Error starting notification container %s. %s", name, err)
	}

	// for debugging purposes we print a failed notification executions
	// output to the logs. Otherwise we have no way to troubleshoot failed
	// notifications. This is temporary code until I've come up with
	// a better solution.
	if info != nil && info.State.ExitCode != 0 && log.GetLevel() >= log.InfoLevel {
		var buf bytes.Buffer
		rc, err := client.ContainerLogs(name, docker.LogOpts)
		if err == nil {
			defer rc.Close()
			stdcopy.StdCopy(&buf, &buf, io.LimitReader(rc, 50000))
		}
		log.Infof("Notification container %s exited with %d", name, info.State.ExitCode)
		log.Infoln(buf.String())
	}

	return err
}
