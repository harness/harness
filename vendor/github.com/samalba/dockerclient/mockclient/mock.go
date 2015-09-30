package mockclient

import (
	"io"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/samalba/dockerclient"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/stretchr/testify/mock"
)

type MockClient struct {
	mock.Mock
}

func NewMockClient() *MockClient {
	return &MockClient{}
}

func (client *MockClient) Info() (*dockerclient.Info, error) {
	args := client.Mock.Called()
	return args.Get(0).(*dockerclient.Info), args.Error(1)
}

func (client *MockClient) ListContainers(all bool, size bool, filters string) ([]dockerclient.Container, error) {
	args := client.Mock.Called(all, size, filters)
	return args.Get(0).([]dockerclient.Container), args.Error(1)
}

func (client *MockClient) InspectContainer(id string) (*dockerclient.ContainerInfo, error) {
	args := client.Mock.Called(id)
	return args.Get(0).(*dockerclient.ContainerInfo), args.Error(1)
}

func (client *MockClient) CreateContainer(config *dockerclient.ContainerConfig, name string) (string, error) {
	args := client.Mock.Called(config, name)
	return args.String(0), args.Error(1)
}

func (client *MockClient) ContainerLogs(id string, options *dockerclient.LogOptions) (io.ReadCloser, error) {
	args := client.Mock.Called(id, options)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (client *MockClient) StartContainer(id string, config *dockerclient.HostConfig) error {
	args := client.Mock.Called(id, config)
	return args.Error(0)
}

func (client *MockClient) StopContainer(id string, timeout int) error {
	args := client.Mock.Called(id, timeout)
	return args.Error(0)
}

func (client *MockClient) RestartContainer(id string, timeout int) error {
	args := client.Mock.Called(id, timeout)
	return args.Error(0)
}

func (client *MockClient) KillContainer(id, signal string) error {
	args := client.Mock.Called(id, signal)
	return args.Error(0)
}

func (client *MockClient) StartMonitorEvents(cb dockerclient.Callback, ec chan error, args ...interface{}) {
	client.Mock.Called(cb, ec, args)
}

func (client *MockClient) StopAllMonitorEvents() {
	client.Mock.Called()
}

func (client *MockClient) Version() (*dockerclient.Version, error) {
	args := client.Mock.Called()
	return args.Get(0).(*dockerclient.Version), args.Error(1)
}

func (client *MockClient) PullImage(name string, auth *dockerclient.AuthConfig) error {
	args := client.Mock.Called(name, auth)
	return args.Error(0)
}

func (client *MockClient) RemoveContainer(id string, force, volumes bool) error {
	args := client.Mock.Called(id, force, volumes)
	return args.Error(0)
}

func (client *MockClient) ListImages() ([]*dockerclient.Image, error) {
	args := client.Mock.Called()
	return args.Get(0).([]*dockerclient.Image), args.Error(1)
}

func (client *MockClient) RemoveImage(name string) error {
	args := client.Mock.Called(name)
	return args.Error(0)
}

func (client *MockClient) PauseContainer(name string) error {
	args := client.Mock.Called(name)
	return args.Error(0)
}

func (client *MockClient) UnpauseContainer(name string) error {
	args := client.Mock.Called(name)
	return args.Error(0)
}

func (client *MockClient) Exec(config *dockerclient.ExecConfig) (string, error) {
	args := client.Mock.Called(config)
	return args.String(0), args.Error(1)
}
