// Copyright 2019 Drone IO, Inc.
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

package bootstrap

import (
	"context"
	"errors"
	"time"

	"github.com/dchest/uniuri"
	"github.com/drone/drone/core"
	"github.com/drone/drone/logger"

	"github.com/sirupsen/logrus"
)

var errMissingToken = errors.New("You must provide the machine account token")

// New returns a new account bootstrapper.
func New(users core.UserStore) *Bootstrapper {
	return &Bootstrapper{
		users: users,
	}
}

// Bootstrapper bootstraps the system with the initial account.
type Bootstrapper struct {
	users core.UserStore
}

// Bootstrap creates the user account. If the account already exists,
// no account is created, and a nil error is returned.
func (b *Bootstrapper) Bootstrap(ctx context.Context, user *core.User) error {
	if user.Login == "" {
		return nil
	}

	log := logrus.WithFields(
		logrus.Fields{
			"login":   user.Login,
			"admin":   user.Admin,
			"machine": user.Machine,
			"token":   user.Hash,
		},
	)

	log.Debugln("bootstrap: create account")

	existingUser, err := b.users.FindLogin(ctx, user.Login)
	if err == nil {
		ctx = logger.WithContext(ctx, log)
		return b.update(ctx, user, existingUser)
	}

	if user.Machine && user.Hash == "" {
		log.Errorln("bootstrap: cannot create account, missing token")
		return errMissingToken
	}

	user.Active = true
	user.Created = time.Now().Unix()
	user.Updated = time.Now().Unix()
	if user.Hash == "" {
		user.Hash = uniuri.NewLen(32)
	}

	err = b.users.Create(ctx, user)
	if err != nil {
		log = log.WithError(err)
		log.Errorln("bootstrap: cannot create account")
		return err
	}

	log = log.WithField("token", user.Hash)
	log.Infoln("bootstrap: account created")
	return nil
}

func (b *Bootstrapper) update(ctx context.Context, src, dst *core.User) error {
	log := logger.FromContext(ctx)
	log.Debugln("bootstrap: updating account")
	var updated bool
	if src.Hash != dst.Hash && src.Hash != "" {
		log.Infoln("bootstrap: found updated user token")
		dst.Hash = src.Hash
		updated = true
	}
	if src.Machine != dst.Machine {
		log.Infoln("bootstrap: found updated machine flag")
		dst.Machine = src.Machine
		updated = true
	}
	if src.Admin != dst.Admin {
		log.Infoln("bootstrap: found updated admin flag")
		dst.Admin = src.Admin
		updated = true
	}
	if !updated {
		log.Debugln("bootstrap: account already up-to-date")
		return nil
	}
	dst.Updated = time.Now().Unix()
	err := b.users.Update(ctx, dst)
	if err != nil {
		log = log.WithError(err)
		log.Errorln("bootstrap: cannot update account")
		return err
	}
	log.Infoln("bootstrap: account successfully updated")
	return nil
}
