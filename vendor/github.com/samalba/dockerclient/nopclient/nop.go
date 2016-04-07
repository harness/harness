package nopclient

import (
	"errors"
	"io"

	"github.com/samalba/dockerclient"
)

var (
	ErrNoEngine = errors.New("Engine no longer exists")
)

type NopClient struct {
}

func NewNopClient() *NopClient {
	return &NopClient{}
}

func (client *NopClient) Info() (*dockerclient.Info, error) {
	return nil, ErrNoEngine
}

func (client *NopClient) ListContainers(all bool, size bool, filters string) ([]dockerclient.Container, error) {
	return nil, ErrNoEngine
}

func (client *NopClient) InspectContainer(id string) (*dockerclient.ContainerInfo, error) {
	return nil, ErrNoEngine
}

func (client *NopClient) InspectImage(id string) (*dockerclient.ImageInfo, error) {
	return nil, ErrNoEngine
}

func (client *NopClient) CreateContainer(config *dockerclient.ContainerConfig, name string, authConfig *dockerclient.AuthConfig) (string, error) {
	return "", ErrNoEngine
}

func (client *NopClient) ContainerLogs(id string, options *dockerclient.LogOptions) (io.ReadCloser, error) {
	return nil, ErrNoEngine
}

func (client *NopClient) ContainerChanges(id string) ([]*dockerclient.ContainerChanges, error) {
	return nil, ErrNoEngine
}

func (client *NopClient) ContainerStats(id string, stopChan <-chan struct{}) (<-chan dockerclient.StatsOrError, error) {
	return nil, ErrNoEngine
}

func (client *NopClient) AttachContainer(id string, options *dockerclient.AttachOptions) (io.ReadCloser, error) {
	return nil, ErrNoEngine
}

func (client *NopClient) StartContainer(id string, config *dockerclient.HostConfig) error {
	return ErrNoEngine
}

func (client *NopClient) StopContainer(id string, timeout int) error {
	return ErrNoEngine
}

func (client *NopClient) RestartContainer(id string, timeout int) error {
	return ErrNoEngine
}

func (client *NopClient) KillContainer(id, signal string) error {
	return ErrNoEngine
}

func (client *NopClient) Wait(id string) <-chan dockerclient.WaitResult {
	return nil
}

func (client *NopClient) MonitorEvents(options *dockerclient.MonitorEventsOptions, stopChan <-chan struct{}) (<-chan dockerclient.EventOrError, error) {
	return nil, ErrNoEngine
}

func (client *NopClient) StartMonitorEvents(cb dockerclient.Callback, ec chan error, args ...interface{}) {
	return
}

func (client *NopClient) StopAllMonitorEvents() {
	return
}

func (client *NopClient) TagImage(nameOrID string, repo string, tag string, force bool) error {
	return ErrNoEngine
}

func (client *NopClient) StartMonitorStats(id string, cb dockerclient.StatCallback, ec chan error, args ...interface{}) {
	return
}

func (client *NopClient) StopAllMonitorStats() {
	return
}

func (client *NopClient) Version() (*dockerclient.Version, error) {
	return nil, ErrNoEngine
}

func (client *NopClient) PullImage(name string, auth *dockerclient.AuthConfig) error {
	return ErrNoEngine
}

func (client *NopClient) PushImage(name, tag string, auth *dockerclient.AuthConfig) error {
	return ErrNoEngine
}

func (client *NopClient) LoadImage(reader io.Reader) error {
	return ErrNoEngine
}

func (client *NopClient) RemoveContainer(id string, force, volumes bool) error {
	return ErrNoEngine
}

func (client *NopClient) ListImages(all bool) ([]*dockerclient.Image, error) {
	return nil, ErrNoEngine
}

func (client *NopClient) RemoveImage(name string, force bool) ([]*dockerclient.ImageDelete, error) {
	return nil, ErrNoEngine
}

func (client *NopClient) SearchImages(query, registry string, authConfig *dockerclient.AuthConfig) ([]dockerclient.ImageSearch, error) {
	return nil, ErrNoEngine
}

func (client *NopClient) PauseContainer(name string) error {
	return ErrNoEngine
}

func (client *NopClient) UnpauseContainer(name string) error {
	return ErrNoEngine
}

func (client *NopClient) ExecCreate(config *dockerclient.ExecConfig) (string, error) {
	return "", ErrNoEngine
}

func (client *NopClient) ExecStart(id string, config *dockerclient.ExecConfig) error {
	return ErrNoEngine
}

func (client *NopClient) ExecResize(id string, width, height int) error {
	return ErrNoEngine
}

func (client *NopClient) RenameContainer(oldName string, newName string) error {
	return ErrNoEngine
}

func (client *NopClient) ImportImage(source string, repository string, tag string, tar io.Reader) (io.ReadCloser, error) {
	return nil, ErrNoEngine
}

func (client *NopClient) BuildImage(image *dockerclient.BuildImage) (io.ReadCloser, error) {
	return nil, ErrNoEngine
}

func (client *NopClient) ListVolumes() ([]*dockerclient.Volume, error) {
	return nil, ErrNoEngine
}

func (client *NopClient) RemoveVolume(name string) error {
	return ErrNoEngine
}

func (client *NopClient) CreateVolume(request *dockerclient.VolumeCreateRequest) (*dockerclient.Volume, error) {
	return nil, ErrNoEngine
}

func (client *NopClient) ListNetworks(filters string) ([]*dockerclient.NetworkResource, error) {
	return nil, ErrNoEngine
}

func (client *NopClient) InspectNetwork(id string) (*dockerclient.NetworkResource, error) {
	return nil, ErrNoEngine
}

func (client *NopClient) CreateNetwork(config *dockerclient.NetworkCreate) (*dockerclient.NetworkCreateResponse, error) {
	return nil, ErrNoEngine
}

func (client *NopClient) ConnectNetwork(id, container string) error {
	return ErrNoEngine
}

func (client *NopClient) DisconnectNetwork(id, container string, force bool) error {
	return ErrNoEngine
}

func (client *NopClient) RemoveNetwork(id string) error {
	return ErrNoEngine
}
