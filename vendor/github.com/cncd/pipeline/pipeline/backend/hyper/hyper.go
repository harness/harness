package hyper

import (
	"context"
	"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/cncd/pipeline/pipeline/backend"

	"github.com/docker/docker/pkg/stdcopy"

	"github.com/hyperhq/hyper-api/types"
	"github.com/hyperhq/hyper-api/client"
)

type engine struct {
	client client.APIClient
}

// New returns a new Docker Engine using the given client.
func New(cli client.APIClient) backend.Engine {
	return &engine{
		client: cli,
	}
}

// NewEnv returns a new Docker Engine using the client connection
// environment variables.
func NewEnv(accessKey string, secretKey string) (backend.Engine, error) {
	var (
		host          = "tcp://us-west-1.hyper.sh:443"
		customHeaders = map[string]string{}
		verStr        = "v1.23"
	)

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	cli, err := client.NewClient(host, verStr, httpClient, customHeaders, accessKey, secretKey)
	if err != nil {
		return nil, err
	}

	return New(cli), nil
}

func (e *engine) Setup(conf *backend.Config) error {
	for _, vol := range conf.Volumes {
		// HYPER.SH IMPLEMENTATION
		vol.Driver = "hyper"
		vol.Name = strings.Replace(vol.Name, "_", "-", -1)

		_, err := e.client.VolumeCreate(noContext, types.VolumeCreateRequest{
			Name:       vol.Name,
			Driver:     vol.Driver,
			DriverOpts: vol.DriverOpts,
		})
		if err != nil {
			return err
		}
	}

	// HYPER.SH IMPLEMENTATION
	// Prepare shared volume name
	sharedVolumeName := conf.Volumes[0].Name + "-share"
	// Prepare intemediary configurations
	ctx := context.Background()
	step := *(conf.Stages[0].Steps[0])
	step.Labels = make(map[string]string)
	step.Labels["sh_hyper_instancetype"] = "s4"
	step.Image = "hyperhq/nfs-server"
	// Generate config & hostConfig
	config := toConfig(&step)
	hostConfig := toHostConfig(&step)
	// Prepare latest image and start container
	e.client.ImagePull(ctx, config.Image, types.ImagePullOptions{})
	e.client.ContainerCreate(ctx, config, hostConfig, nil, sharedVolumeName)
	e.client.ContainerStart(ctx, sharedVolumeName, "")

	return nil
}

func (e *engine) Exec(proc *backend.Step) error {
	// HYPER.SH IMPLEMENTATION
	// Fix names to comply with hyper.sh restrictions
	proc.Name = strings.Replace(proc.Name, "_", "-", -1)
	proc.Labels = make(map[string]string)
	// Create labels if not created
	if proc.Labels == nil {
		proc.Labels = make(map[string]string)
	}
	// If size defined, set default as size else "s4"
	if proc.Size != "" {
		proc.Labels["sh_hyper_instancetype"] = proc.Size
	} else {
		proc.Labels["sh_hyper_instancetype"] = "s4"
	}
	// If name is `plugins/docker`, replace with `plugins/hyper`
	if strings.Contains(proc.Image, "plugins/docker") {
		// proc.Image = "plugins/hyper"
		proc.Image = "lflare/drone-hyper"
	}

	// Get context
	ctx := context.Background()
	
	// Convert
	config := toConfig(proc)
	hostConfig := toHostConfig(proc)

	// HYPER.SH IMPLEMENTATION
	sharedVolumeName := strings.Split(proc.Volumes[0], ":")[0] + "-share"
	hostConfig.VolumesFrom = append(hostConfig.VolumesFrom, sharedVolumeName)
	config.Volumes = nil
	config.Hostname = proc.Alias

	// Create pull options with encoded auth credentials
	pullopts := types.ImagePullOptions{}
	if proc.AuthConfig.Username != "" && proc.AuthConfig.Password != "" {
		pullopts.RegistryAuth, _ = encodeAuthToBase64(proc.AuthConfig)
	}

	// If configured, pull latest image
	if proc.Pull {
		rc, perr := e.client.ImagePull(ctx, config.Image, pullopts)
		if perr == nil {
			io.Copy(ioutil.Discard, rc)
			rc.Close()
		}
		// fix for drone/drone#1917
		if perr != nil && proc.AuthConfig.Password != "" {
			return perr
		}
	}

	_, err := e.client.ContainerCreate(ctx, config, hostConfig, nil, proc.Name)

	// If image not found, pull and reattempt to create container
	if client.IsErrImageNotFound(err) {
		rc, perr := e.client.ImagePull(ctx, config.Image, pullopts)

		if perr != nil {
			return perr
		}

		io.Copy(ioutil.Discard, rc)
		rc.Close()

		_, err = e.client.ContainerCreate(ctx, config, hostConfig, nil, proc.Name)
	}
	if err != nil {
		return err
	}

	return e.client.ContainerStart(ctx, proc.Name, "")
}

func (e *engine) Kill(proc *backend.Step) error {
	return e.client.ContainerKill(noContext, proc.Name, "9")
}

func (e *engine) Wait(proc *backend.Step) (*backend.State, error) {
	_, err := e.client.ContainerWait(noContext, proc.Name)
	info, err := e.client.ContainerInspect(noContext, proc.Name)

	if err != nil {
		return nil, err
	}

	return &backend.State{
		Exited:    true,
		ExitCode:  info.State.ExitCode,
		OOMKilled: info.State.OOMKilled,
	}, nil
}

func (e *engine) Tail(proc *backend.Step) (io.ReadCloser, error) {
	logs, err := e.client.ContainerLogs(noContext, proc.Name, logsOpts)

	if err != nil {
		return nil, err
	}

	rc, wc := io.Pipe()

	go func() {
		stdcopy.StdCopy(wc, wc, logs)
		logs.Close()
		wc.Close()
		rc.Close()
	}()

	return rc, nil
}

func (e *engine) Destroy(conf *backend.Config) error {
	// For each container in each step in each stage, remove
	for _, stage := range conf.Stages {
		for _, step := range stage.Steps {
			e.client.ContainerRemove(noContext, step.Name, removeOpts)
		}
	}

	// HYPER.SH IMPLEMENTATION
	// Remove shared volume container
	sharedVolumeName := conf.Volumes[0].Name + "-share"
	e.client.ContainerRemove(noContext, sharedVolumeName, removeOpts)

	// For each configured volumes, remove
	for _, volume := range conf.Volumes {
		e.client.VolumeRemove(noContext, volume.Name)
	}

	return nil
}

var (
	noContext = context.Background()

	removeOpts = types.ContainerRemoveOptions{
		RemoveVolumes: true,
		RemoveLinks:   false,
		Force:         true,
	}

	logsOpts = types.ContainerLogsOptions{
		Follow:     true,
		ShowStdout: true,
		ShowStderr: true,
		Details:    false,
		Timestamps: false,
	}
)
