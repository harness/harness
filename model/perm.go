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

// PermStore persists repository permissions information to storage.
type PermStore interface {
	PermFind(user *User, repo *Repo) (*Perm, error)
	PermUpsert(perm *Perm) error
	PermBatch(perms []*Perm) error
	PermDelete(perm *Perm) error
	PermFlush(user *User, before int64) error
}

// Perm defines a repository permission for an individual user.
type Perm struct {
	UserID int64  `json:"-"      meddler:"perm_user_id"`
	RepoID int64  `json:"-"      meddler:"perm_repo_id"`
	Repo   string `json:"-"      meddler:"-"`
	Pull   bool   `json:"pull"   meddler:"perm_pull"`
	Push   bool   `json:"push"   meddler:"perm_push"`
	Admin  bool   `json:"admin"  meddler:"perm_admin"`
	Synced int64  `json:"synced" meddler:"perm_synced"`
}
