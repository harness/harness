package mocks

import (
	"io"

	"github.com/drone/drone/common"
	"github.com/stretchr/testify/mock"
)

type Datastore struct {
	mock.Mock
}

func (m *Datastore) User(id int64) (*common.User, error) {
	ret := m.Called(id)

	var r0 *common.User
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.User)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) UserLogin(_a0 string) (*common.User, error) {
	ret := m.Called(_a0)

	var r0 *common.User
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.User)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) UserList() ([]*common.User, error) {
	ret := m.Called()

	var r0 []*common.User
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*common.User)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) UserFeed(_a0 *common.User, _a1 int, _a2 int) ([]*common.RepoCommit, error) {
	ret := m.Called(_a0, _a1, _a2)

	var r0 []*common.RepoCommit
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*common.RepoCommit)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) UserCount() (int, error) {
	ret := m.Called()

	r0 := ret.Get(0).(int)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) AddUser(_a0 *common.User) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) SetUser(_a0 *common.User) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) DelUser(_a0 *common.User) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) Token(_a0 int64) (*common.Token, error) {
	ret := m.Called(_a0)

	var r0 *common.Token
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Token)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) TokenLabel(_a0 *common.User, _a1 string) (*common.Token, error) {
	ret := m.Called(_a0, _a1)

	var r0 *common.Token
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Token)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) TokenList(_a0 *common.User) ([]*common.Token, error) {
	ret := m.Called(_a0)

	var r0 []*common.Token
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*common.Token)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) AddToken(_a0 *common.Token) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) DelToken(_a0 *common.Token) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) Starred(_a0 *common.User, _a1 *common.Repo) (bool, error) {
	ret := m.Called(_a0, _a1)

	r0 := ret.Get(0).(bool)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) AddStar(_a0 *common.User, _a1 *common.Repo) error {
	ret := m.Called(_a0, _a1)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) DelStar(_a0 *common.User, _a1 *common.Repo) error {
	ret := m.Called(_a0, _a1)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) Repo(id int64) (*common.Repo, error) {
	ret := m.Called(id)

	var r0 *common.Repo
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Repo)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) RepoName(owner string, name string) (*common.Repo, error) {
	ret := m.Called(owner, name)

	var r0 *common.Repo
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Repo)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) RepoList(_a0 *common.User) ([]*common.Repo, error) {
	ret := m.Called(_a0)

	var r0 []*common.Repo
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*common.Repo)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) AddRepo(_a0 *common.Repo) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) SetRepo(_a0 *common.Repo) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) DelRepo(_a0 *common.Repo) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) Commit(_a0 int64) (*common.Commit, error) {
	ret := m.Called(_a0)

	var r0 *common.Commit
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Commit)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) CommitSeq(_a0 *common.Repo, _a1 int) (*common.Commit, error) {
	ret := m.Called(_a0, _a1)

	var r0 *common.Commit
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Commit)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) CommitLast(_a0 *common.Repo, _a1 string) (*common.Commit, error) {
	ret := m.Called(_a0, _a1)

	var r0 *common.Commit
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Commit)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) CommitList(_a0 *common.Repo, _a1 int, _a2 int) ([]*common.Commit, error) {
	ret := m.Called(_a0, _a1, _a2)

	var r0 []*common.Commit
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*common.Commit)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) AddCommit(_a0 *common.Commit) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) SetCommit(_a0 *common.Commit) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) KillCommits() error {
	ret := m.Called()

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) Build(_a0 int64) (*common.Build, error) {
	ret := m.Called(_a0)

	var r0 *common.Build
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Build)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) BuildSeq(_a0 *common.Commit, _a1 int) (*common.Build, error) {
	ret := m.Called(_a0, _a1)

	var r0 *common.Build
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Build)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) BuildList(_a0 *common.Commit) ([]*common.Build, error) {
	ret := m.Called(_a0)

	var r0 []*common.Build
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*common.Build)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) SetBuild(_a0 *common.Build) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) GetBlob(path string) ([]byte, error) {
	ret := m.Called(path)

	var r0 []byte
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]byte)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) GetBlobReader(path string) (io.ReadCloser, error) {
	ret := m.Called(path)

	r0 := ret.Get(0).(io.ReadCloser)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) SetBlob(path string, data []byte) error {
	ret := m.Called(path, data)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) SetBlobReader(path string, r io.Reader) error {
	ret := m.Called(path, r)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) DelBlob(path string) error {
	ret := m.Called(path)

	r0 := ret.Error(0)

	return r0
}
