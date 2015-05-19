package mocks

import (
	"io"

	common "github.com/drone/drone/pkg/types"
	"github.com/stretchr/testify/mock"
)

type Store struct {
	mock.Mock
}

func (m *Store) User(id int64) (*common.User, error) {
	ret := m.Called(id)

	var r0 *common.User
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.User)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) UserLogin(_a0 string) (*common.User, error) {
	ret := m.Called(_a0)

	var r0 *common.User
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.User)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) UserList() ([]*common.User, error) {
	ret := m.Called()

	var r0 []*common.User
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*common.User)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) UserFeed(_a0 *common.User, _a1 int, _a2 int) ([]*common.RepoCommit, error) {
	ret := m.Called(_a0, _a1, _a2)

	var r0 []*common.RepoCommit
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*common.RepoCommit)
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
func (m *Store) AddUser(_a0 *common.User) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) SetUser(_a0 *common.User) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) DelUser(_a0 *common.User) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) Token(_a0 int64) (*common.Token, error) {
	ret := m.Called(_a0)

	var r0 *common.Token
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Token)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) TokenLabel(_a0 *common.User, _a1 string) (*common.Token, error) {
	ret := m.Called(_a0, _a1)

	var r0 *common.Token
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Token)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) TokenList(_a0 *common.User) ([]*common.Token, error) {
	ret := m.Called(_a0)

	var r0 []*common.Token
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*common.Token)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) AddToken(_a0 *common.Token) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) DelToken(_a0 *common.Token) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) Starred(_a0 *common.User, _a1 *common.Repo) (bool, error) {
	ret := m.Called(_a0, _a1)

	r0 := ret.Get(0).(bool)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) AddStar(_a0 *common.User, _a1 *common.Repo) error {
	ret := m.Called(_a0, _a1)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) DelStar(_a0 *common.User, _a1 *common.Repo) error {
	ret := m.Called(_a0, _a1)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) Repo(id int64) (*common.Repo, error) {
	ret := m.Called(id)

	var r0 *common.Repo
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Repo)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) RepoName(owner string, name string) (*common.Repo, error) {
	ret := m.Called(owner, name)

	var r0 *common.Repo
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Repo)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) RepoList(_a0 *common.User) ([]*common.Repo, error) {
	ret := m.Called(_a0)

	var r0 []*common.Repo
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*common.Repo)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) AddRepo(_a0 *common.Repo) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) SetRepo(_a0 *common.Repo) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) DelRepo(_a0 *common.Repo) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) Commit(_a0 int64) (*common.Commit, error) {
	ret := m.Called(_a0)

	var r0 *common.Commit
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Commit)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) CommitSeq(_a0 *common.Repo, _a1 int) (*common.Commit, error) {
	ret := m.Called(_a0, _a1)

	var r0 *common.Commit
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Commit)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) CommitLast(_a0 *common.Repo, _a1 string) (*common.Commit, error) {
	ret := m.Called(_a0, _a1)

	var r0 *common.Commit
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Commit)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) CommitList(_a0 *common.Repo, _a1 int, _a2 int) ([]*common.Commit, error) {
	ret := m.Called(_a0, _a1, _a2)

	var r0 []*common.Commit
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*common.Commit)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) AddCommit(_a0 *common.Commit) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) SetCommit(_a0 *common.Commit) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Store) KillCommits() error {
	ret := m.Called()

	r0 := ret.Error(0)

	return r0
}
func (m *Store) Build(_a0 int64) (*common.Build, error) {
	ret := m.Called(_a0)

	var r0 *common.Build
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Build)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) BuildSeq(_a0 *common.Commit, _a1 int) (*common.Build, error) {
	ret := m.Called(_a0, _a1)

	var r0 *common.Build
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Build)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) BuildList(_a0 *common.Commit) ([]*common.Build, error) {
	ret := m.Called(_a0)

	var r0 []*common.Build
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*common.Build)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) SetBuild(_a0 *common.Build) error {
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
func (m *Store) Agent(_a0 *common.Commit) (string, error) {
	ret := m.Called(_a0)

	r0 := ret.Get(0).(string)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Store) SetAgent(_a0 *common.Commit, _a1 string) error {
	ret := m.Called(_a0, _a1)

	r0 := ret.Error(0)

	return r0
}
