package mocks

import "github.com/stretchr/testify/mock"

import "io"
import "github.com/drone/drone/common"

type Datastore struct {
	mock.Mock
}

func (m *Datastore) User(_a0 string) (*common.User, error) {
	ret := m.Called(_a0)

	var r0 *common.User
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.User)
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
func (m *Datastore) UserList() ([]*common.User, error) {
	ret := m.Called()

	var r0 []*common.User
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*common.User)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) SetUser(_a0 *common.User) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) SetUserNotExists(_a0 *common.User) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) DelUser(_a0 *common.User) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) Token(_a0 string, _a1 string) (*common.Token, error) {
	ret := m.Called(_a0, _a1)

	var r0 *common.Token
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Token)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) TokenList(_a0 string) ([]*common.Token, error) {
	ret := m.Called(_a0)

	var r0 []*common.Token
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*common.Token)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) SetToken(_a0 *common.Token) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) DelToken(_a0 *common.Token) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) Subscribed(_a0 string, _a1 string) (bool, error) {
	ret := m.Called(_a0, _a1)

	r0 := ret.Get(0).(bool)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) SetSubscriber(_a0 string, _a1 string) error {
	ret := m.Called(_a0, _a1)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) DelSubscriber(_a0 string, _a1 string) error {
	ret := m.Called(_a0, _a1)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) Repo(_a0 string) (*common.Repo, error) {
	ret := m.Called(_a0)

	var r0 *common.Repo
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Repo)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) RepoList(_a0 string) ([]*common.Repo, error) {
	ret := m.Called(_a0)

	var r0 []*common.Repo
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*common.Repo)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) RepoParams(_a0 string) (map[string]string, error) {
	ret := m.Called(_a0)

	var r0 map[string]string
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(map[string]string)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) RepoKeypair(_a0 string) (*common.Keypair, error) {
	ret := m.Called(_a0)

	var r0 *common.Keypair
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Keypair)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) SetRepo(_a0 *common.Repo) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) SetRepoNotExists(_a0 *common.User, _a1 *common.Repo) error {
	ret := m.Called(_a0, _a1)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) SetRepoParams(_a0 string, _a1 map[string]string) error {
	ret := m.Called(_a0, _a1)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) SetRepoKeypair(_a0 string, _a1 *common.Keypair) error {
	ret := m.Called(_a0, _a1)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) DelRepo(_a0 *common.Repo) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) Build(_a0 string, _a1 int) (*common.Build, error) {
	ret := m.Called(_a0, _a1)

	var r0 *common.Build
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Build)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) BuildList(_a0 string) ([]*common.Build, error) {
	ret := m.Called(_a0)

	var r0 []*common.Build
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*common.Build)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) BuildLast(_a0 string) (*common.Build, error) {
	ret := m.Called(_a0)

	var r0 *common.Build
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Build)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) SetBuild(_a0 string, _a1 *common.Build) error {
	ret := m.Called(_a0, _a1)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) Status(_a0 string, _a1 int, _a2 string) (*common.Status, error) {
	ret := m.Called(_a0, _a1, _a2)

	var r0 *common.Status
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.Status)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) StatusList(_a0 string, _a1 int) ([]*common.Status, error) {
	ret := m.Called(_a0, _a1)

	var r0 []*common.Status
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*common.Status)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) SetStatus(_a0 string, _a1 int, _a2 *common.Status) error {
	ret := m.Called(_a0, _a1, _a2)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) LogReader(_a0 string, _a1 int, _a2 int) (io.Reader, error) {
	ret := m.Called(_a0, _a1, _a2)

	r0 := ret.Get(0).(io.Reader)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Datastore) SetLogs(_a0 string, _a1 int, _a2 int, _a3 []byte) error {
	ret := m.Called(_a0, _a1, _a2, _a3)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) SetBuildState(_a0 string, _a1 *common.Build) error {
	ret := m.Called(_a0, _a1)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) SetBuildStatus(_a0 string, _a1 int, _a2 *common.Status) error {
	ret := m.Called(_a0, _a1, _a2)

	r0 := ret.Error(0)

	return r0
}
func (m *Datastore) SetBuildTask(_a0 string, _a1 int, _a2 *common.Task) error {
	ret := m.Called(_a0, _a1, _a2)

	r0 := ret.Error(0)

	return r0
}
