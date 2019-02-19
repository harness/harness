// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/drone/drone/core"
	"github.com/drone/drone/logger"
	"github.com/drone/go-login/login"

	"github.com/dchest/uniuri"
	"github.com/sirupsen/logrus"
)

// period at which the user account is synchronized
// with the remote system. Default is weekly.
var syncPeriod = time.Hour * 24 * 7

// period at which the sync should timeout
var syncTimeout = time.Minute * 30

// HandleLogin creates and http.HandlerFunc that handles user
// authentication and session initialization.
func HandleLogin(
	users core.UserStore,
	userz core.UserService,
	syncer core.Syncer,
	session core.Session,
	admission core.AdmissionService,
	sender core.WebhookSender,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		err := login.ErrorFrom(ctx)
		if err != nil {
			writeLoginError(w, r, err)
			logrus.Debugf("cannot authenticate user: %s", err)
			return
		}

		// The authorization token is passed from the
		// login middleware in the context.
		tok := login.TokenFrom(ctx)

		account, err := userz.Find(ctx, tok.Access, tok.Refresh)
		if err != nil {
			writeLoginError(w, r, err)
			logrus.Debugf("cannot find remote user: %s", err)
			return
		}

		logger := logrus.WithField("login", account.Login)
		logger.Debugf("attempting authentication")

		user, err := users.FindLogin(ctx, account.Login)
		if err == sql.ErrNoRows {
			user = &core.User{
				Login:     account.Login,
				Email:     account.Email,
				Avatar:    account.Avatar,
				Admin:     false,
				Machine:   false,
				Active:    true,
				Syncing:   true,
				Synced:    0,
				LastLogin: time.Now().Unix(),
				Created:   time.Now().Unix(),
				Updated:   time.Now().Unix(),
				Token:     tok.Access,
				Refresh:   tok.Refresh,
				Hash:      uniuri.NewLen(32),
			}
			if !tok.Expires.IsZero() {
				user.Expiry = tok.Expires.Unix()
			}

			err = admission.Admit(ctx, user)
			if err != nil {
				writeLoginError(w, r, err)
				logger.Errorf("cannot admit user: %s", err)
				return
			}

			err = users.Create(ctx, user)
			if err != nil {
				writeLoginError(w, r, err)
				logger.Errorf("cannot create user: %s", err)
				return
			}

			err = sender.Send(ctx, &core.WebhookData{
				Event:  core.WebhookEventUser,
				Action: core.WebhookActionCreated,
				User:   user,
			})
			if err != nil {
				logger.Errorf("cannot send webhook: %s", err)
			} else {
				logger.Debugf("successfully created user")
			}
		} else if err != nil {
			writeLoginError(w, r, err)
			logger.Errorf("cannot find user: %s", err)
			return
		}

		if user.Machine {
			writeLoginErrorStr(w, r, "Machine account login is forbidden")
			return
		}

		if user.Active == false {
			writeLoginErrorStr(w, r, "Account is not active")
			return
		}

		user.Avatar = account.Avatar
		user.Email = account.Email
		user.Token = tok.Access
		user.Refresh = tok.Refresh
		user.LastLogin = time.Now().Unix()
		if !tok.Expires.IsZero() {
			user.Expiry = tok.Expires.Unix()
		}

		// If the user account has never been synchrnoized we
		// execute the synchonrization logic.
		if time.Unix(user.Synced, 0).Add(syncPeriod).Before(time.Now()) {
			user.Syncing = true
		}

		err = users.Update(ctx, user)
		if err != nil {
			// if the account update fails we should still
			// proceed to create the user session. This is
			// considered a non-fatal error.
			logger.Errorf("cannot update user: %s", err)
		}

		// launch the synchrnoization process in a go-routine,
		// since it is a long-running process and can take up
		// to a few minutes.
		if user.Syncing {
			go synchornize(ctx, syncer, user)
		}

		logger.Debugf("authentication successful")

		session.Create(w, user)
		http.Redirect(w, r, "/", 303)
	}
}

func synchornize(ctx context.Context, syncer core.Syncer, user *core.User) {
	log := logrus.WithField("login", user.Login)
	log.Debugf("begin synchronization")

	timeout, cancel := context.WithTimeout(context.Background(), syncTimeout)
	timeout = logger.WithContext(timeout, log)
	defer cancel()
	_, err := syncer.Sync(timeout, user)
	if err != nil {
		log.Debugf("synchronization failed: %s", err)
	} else {
		log.Debugf("synchronization success")
	}
}

func writeLoginError(w http.ResponseWriter, r *http.Request, err error) {
	http.Redirect(w, r, "/login/error?message="+err.Error(), 303)
}

func writeLoginErrorStr(w http.ResponseWriter, r *http.Request, s string) {
	writeLoginError(w, r, errors.New(s))
}

func writeCookie(w http.ResponseWriter, cookie *http.Cookie) {
	w.Header().Set("Set-Cookie", cookie.String()+"; SameSite=lax")
}

// HandleLoginForm creates and http.HandlerFunc that presents the
// user with an Login form for password-based authentication.
func HandleLoginForm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, loginForm)
	}
}

// html page displayed to collect credentials.
var loginForm = `
<form method="POST" action="/login">
<input type="text" name="username" />
<input type="password" name="password" />
<input type="submit" />
</form>
`
