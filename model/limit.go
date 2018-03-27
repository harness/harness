// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

// Limiter defines an interface for limiting repository creation.
// This could be used, for example, to limit repository creation to
// a specific organization or a specific set of users.
type Limiter interface {
	LimitUser(*User) error
	LimitRepo(*User, *Repo) error
	LimitRepos(*User, []*Repo) []*Repo
	LimitBuild(*User, *Repo, *Build) error
}

// NoLimit implements the Limiter interface without enforcing any
// actual limits. All limiting functions are no-ops.
type NoLimit struct{}

// LimitUser is a no-op for limiting user creation.
func (NoLimit) LimitUser(*User) error { return nil }

// LimitRepo is a no-op for limiting repo creation.
func (NoLimit) LimitRepo(*User, *Repo) error { return nil }

// LimitRepos is a no-op for limiting repository listings.
func (NoLimit) LimitRepos(user *User, repos []*Repo) []*Repo { return repos }

// LimitBuild is a no-op for limiting build creation.
func (NoLimit) LimitBuild(*User, *Repo, *Build) error { return nil }
