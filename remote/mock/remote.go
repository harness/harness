package mock

import "github.com/stretchr/testify/mock"

import "net/http"
import "github.com/drone/drone/model"

type Remote struct {
	mock.Mock
}

func (_m *Remote) Login(w http.ResponseWriter, r *http.Request) (*model.User, bool, error) {
	ret := _m.Called(w, r)

	var r0 *model.User
	if rf, ok := ret.Get(0).(func(http.ResponseWriter, *http.Request) *model.User); ok {
		r0 = rf(w, r)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.User)
		}
	}

	var r1 bool
	if rf, ok := ret.Get(1).(func(http.ResponseWriter, *http.Request) bool); ok {
		r1 = rf(w, r)
	} else {
		r1 = ret.Get(1).(bool)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(http.ResponseWriter, *http.Request) error); ok {
		r2 = rf(w, r)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *Remote) Auth(token string, secret string) (string, error) {
	ret := _m.Called(token, secret)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string) string); ok {
		r0 = rf(token, secret)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(token, secret)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Remote) Repo(u *model.User, owner string, repo string) (*model.Repo, error) {
	ret := _m.Called(u, owner, repo)

	var r0 *model.Repo
	if rf, ok := ret.Get(0).(func(*model.User, string, string) *model.Repo); ok {
		r0 = rf(u, owner, repo)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.Repo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*model.User, string, string) error); ok {
		r1 = rf(u, owner, repo)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Remote) Repos(u *model.User) ([]*model.RepoLite, error) {
	ret := _m.Called(u)

	var r0 []*model.RepoLite
	if rf, ok := ret.Get(0).(func(*model.User) []*model.RepoLite); ok {
		r0 = rf(u)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*model.RepoLite)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*model.User) error); ok {
		r1 = rf(u)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Remote) Perm(u *model.User, owner string, repo string) (*model.Perm, error) {
	ret := _m.Called(u, owner, repo)

	var r0 *model.Perm
	if rf, ok := ret.Get(0).(func(*model.User, string, string) *model.Perm); ok {
		r0 = rf(u, owner, repo)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.Perm)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*model.User, string, string) error); ok {
		r1 = rf(u, owner, repo)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Remote) Script(u *model.User, r *model.Repo, b *model.Build) ([]byte, []byte, error) {
	ret := _m.Called(u, r, b)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(*model.User, *model.Repo, *model.Build) []byte); ok {
		r0 = rf(u, r, b)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 []byte
	if rf, ok := ret.Get(1).(func(*model.User, *model.Repo, *model.Build) []byte); ok {
		r1 = rf(u, r, b)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]byte)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(*model.User, *model.Repo, *model.Build) error); ok {
		r2 = rf(u, r, b)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *Remote) Status(u *model.User, r *model.Repo, b *model.Build, link string) error {
	ret := _m.Called(u, r, b, link)

	var r0 error
	if rf, ok := ret.Get(0).(func(*model.User, *model.Repo, *model.Build, string) error); ok {
		r0 = rf(u, r, b, link)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Remote) Netrc(u *model.User, r *model.Repo) (*model.Netrc, error) {
	ret := _m.Called(u, r)

	var r0 *model.Netrc
	if rf, ok := ret.Get(0).(func(*model.User, *model.Repo) *model.Netrc); ok {
		r0 = rf(u, r)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.Netrc)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*model.User, *model.Repo) error); ok {
		r1 = rf(u, r)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Remote) Activate(u *model.User, r *model.Repo, k *model.Key, link string) error {
	ret := _m.Called(u, r, k, link)

	var r0 error
	if rf, ok := ret.Get(0).(func(*model.User, *model.Repo, *model.Key, string) error); ok {
		r0 = rf(u, r, k, link)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Remote) Deactivate(u *model.User, r *model.Repo, link string) error {
	ret := _m.Called(u, r, link)

	var r0 error
	if rf, ok := ret.Get(0).(func(*model.User, *model.Repo, string) error); ok {
		r0 = rf(u, r, link)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Remote) Hook(r *http.Request) (*model.Repo, *model.Build, error) {
	ret := _m.Called(r)

	var r0 *model.Repo
	if rf, ok := ret.Get(0).(func(*http.Request) *model.Repo); ok {
		r0 = rf(r)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.Repo)
		}
	}

	var r1 *model.Build
	if rf, ok := ret.Get(1).(func(*http.Request) *model.Build); ok {
		r1 = rf(r)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*model.Build)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(*http.Request) error); ok {
		r2 = rf(r)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
