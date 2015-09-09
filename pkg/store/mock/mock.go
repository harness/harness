package mocks

import (
	"io"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/stretchr/testify/mock"
	"github.com/drone/drone/pkg/types"
)

type Store struct {
	mock.Mock
}

func (m *Store) User(id int64) (*types.User, error) {
	ret := m.Called(id)

	var r0 *types.User
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*types.User)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) UserLogin(_a0 string) (*types.User, error) {
	ret := m.Called(_a0)

	var r0 *types.User
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*types.User)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) UserList() ([]*types.User, error) {
	ret := m.Called()

	var r0 []*types.User
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*types.User)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) UserFeed(_a0 *types.User, _a1 int, _a2 int) ([]*types.RepoCommit, error) {
	ret := m.Called(_a0, _a1, _a2)

	var r0 []*types.RepoCommit
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*types.RepoCommit)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) UserCount() (int, error) {
	ret := m.Called()

	r0 := ret.Get(0).(int)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) AddUser(_a0 *types.User) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) SetUser(_a0 *types.User) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) DelUser(_a0 *types.User) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) Starred(_a0 *types.User, _a1 *types.Repo) (bool, error) {
	ret := m.Called(_a0, _a1)

	r0 := ret.Get(0).(bool)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) AddStar(_a0 *types.User, _a1 *types.Repo) error {
	ret := m.Called(_a0, _a1)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) DelStar(_a0 *types.User, _a1 *types.Repo) error {
	ret := m.Called(_a0, _a1)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) Repo(id int64) (*types.Repo, error) {
	ret := m.Called(id)

	var r0 *types.Repo
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*types.Repo)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) RepoName(owner string, name string) (*types.Repo, error) {
	ret := m.Called(owner, name)

	var r0 *types.Repo
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*types.Repo)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) RepoList(_a0 *types.User) ([]*types.Repo, error) {
	ret := m.Called(_a0)

	var r0 []*types.Repo
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*types.Repo)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) AddRepo(_a0 *types.Repo) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) SetRepo(_a0 *types.Repo) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) DelRepo(_a0 *types.Repo) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) Build(_a0 int64) (*types.Build, error) {
	ret := m.Called(_a0)

	var r0 *types.Build
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*types.Build)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) BuildNumber(_a0 *types.Repo, _a1 int) (*types.Build, error) {
	ret := m.Called(_a0, _a1)

	var r0 *types.Build
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*types.Build)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) BuildPullRequestNumber(_a0 *types.Repo, _a1 int) (*types.Build, error) {
	ret := m.Called(_a0, _a1)

	var r0 *types.Build
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*types.Build)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) BuildSha(_a0 *types.Repo, _a1, _a2 string) (*types.Build, error) {
	ret := m.Called(_a0, _a1, _a2)

	var r0 *types.Build
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*types.Build)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) BuildLast(_a0 *types.Repo, _a1 string) (*types.Build, error) {
	ret := m.Called(_a0, _a1)

	var r0 *types.Build
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*types.Build)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) BuildList(_a0 *types.Repo, _a1 int, _a2 int) ([]*types.Build, error) {
	ret := m.Called(_a0, _a1, _a2)

	var r0 []*types.Build
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*types.Build)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) AddBuild(_a0 *types.Build) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) SetBuild(_a0 *types.Build) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) KillBuilds() error {
	ret := m.Called()

	r0 := ret.Error(0)

	return r0
}
func (m *Store) Job(_a0 int64) (*types.Job, error) {
	ret := m.Called(_a0)

	var r0 *types.Job
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*types.Job)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) JobNumber(_a0 *types.Build, _a1 int) (*types.Job, error) {
	ret := m.Called(_a0, _a1)

	var r0 *types.Job
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*types.Job)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) JobList(_a0 *types.Build) ([]*types.Job, error) {
	ret := m.Called(_a0)

	var r0 []*types.Job
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*types.Job)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) SetJob(_a0 *types.Job) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) GetBlob(path string) ([]byte, error) {
	ret := m.Called(path)

	var r0 []byte
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]byte)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) GetBlobReader(path string) (io.ReadCloser, error) {
	ret := m.Called(path)

	r0 := ret.Get(0).(io.ReadCloser)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) SetBlob(path string, data []byte) error {
	ret := m.Called(path, data)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) SetBlobReader(path string, r io.Reader) error {
	ret := m.Called(path, r)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) DelBlob(path string) error {
	ret := m.Called(path)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) Agent(_a0 *types.Build) (string, error) {
	ret := m.Called(_a0)

	r0 := ret.Get(0).(string)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) SetAgent(_a0 *types.Build, _a1 string) error {
	ret := m.Called(_a0, _a1)

	r0 := ret.Error(0)

	return r0
}
