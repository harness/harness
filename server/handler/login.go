package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/drone/drone/plugin/remote"
	"github.com/drone/drone/server/capability"
	"github.com/drone/drone/server/datastore"
	"github.com/drone/drone/server/session"
	"github.com/drone/drone/shared/model"
	"github.com/goji/context"
	"github.com/zenazn/goji/web"
)

// GetLogin accepts a request to authorize the user and to
// return a valid OAuth2 access token. The access token is
// returned as url segment #access_token
//
//     GET /login/:host
//
func GetLogin(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var host = c.URLParams["host"]
	var redirect = "/"
	var remote = remote.Lookup(host)
	if remote == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// authenticate the user
	login, err := remote.Authorize(w, r)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	} else if login == nil {
		// in this case we probably just redirected
		// the user, so we can exit with no error
		return
	}

	// get the user from the database
	u, err := datastore.GetUserLogin(ctx, host, login.Login)
	if err != nil {
		// if self-registration is disabled we should
		// return a notAuthorized error. the only exception
		// is if no users exist yet in the system we'll proceed.
		if capability.Enabled(ctx, capability.Registration) == false {
			users, err := datastore.GetUserList(ctx)
			if err != nil || len(users) != 0 {
				w.WriteHeader(http.StatusForbidden)
				return
			}
		}

		// create the user account
		u = model.NewUser(remote.GetKind(), login.Login, login.Email)
		u.Name = login.Name
		u.SetEmail(login.Email)

		// insert the user into the database
		if err := datastore.PostUser(ctx, u); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// if this is the first user, they
		// should be an admin.
		if u.ID == 1 {
			u.Admin = true
		}
	}

	// update the user access token
	// in case it changed in GitHub
	u.Access = login.Access
	u.Secret = login.Secret
	u.Name = login.Name
	u.SetEmail(login.Email)
	u.Syncing = true //u.IsStale() // todo (badrydzewski) should not always sync
	if err := datastore.PutUser(ctx, u); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// look at the last synchronized date to determine if
	// we need to re-sync the account.
	//
	// todo(bradrydzewski) this should move to a server/sync package and
	//      should be injected into this struct, just like the database code.
	//
	// todo(bradrydzewski) this login should be a bit more intelligent
	//      than the current implementation.
	if u.Syncing {
		redirect = "/sync"
		log.Println("sync user account.", u.Login)

		// sync inside a goroutine. This should eventually be moved to
		// its own package / sync utility.
		go func() {
			repos, err := remote.GetRepos(u)
			if err != nil {
				log.Println("Error syncing user account, listing repositories", u.Login, err)
				return
			}

			// insert all repositories
			for _, repo := range repos {
				var role = repo.Role
				if err := datastore.PostRepo(ctx, repo); err != nil {
					// typically we see a failure because the repository already exists
					// in which case, we can retrieve the existing record to get the ID.
					repo, err = datastore.GetRepoName(ctx, repo.Host, repo.Owner, repo.Name)
					if err != nil {
						log.Println("Error adding repo.", u.Login, repo.Name, err)
						continue
					}
				}

				// add user permissions
				perm := model.Perm{
					UserID: u.ID,
					RepoID: repo.ID,
					Read:   role.Read,
					Write:  role.Write,
					Admin:  role.Admin,
				}
				if err := datastore.PostPerm(ctx, &perm); err != nil {
					log.Println("Error adding permissions.", u.Login, repo.Name, err)
					continue
				}

				log.Println("Successfully syced repo.", u.Login+"/"+repo.Name)
			}

			u.Synced = time.Now().UTC().Unix()
			u.Syncing = false
			if err := datastore.PutUser(ctx, u); err != nil {
				log.Println("Error syncing user account, updating sync date", u.Login, err)
				return
			}
		}()
	}

	token, err := session.GenerateToken(ctx, r, u)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	redirect = redirect + "#access_token=" + token

	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

// GetLoginList accepts a request to retrive a list of
// all OAuth login options.
//
//     GET /api/remotes/login
//
func GetLoginList(c web.C, w http.ResponseWriter, r *http.Request) {
	var list = remote.Registered()
	var logins []interface{}
	for _, item := range list {
		logins = append(logins, struct {
			Type string `json:"type"`
			Host string `json:"host"`
		}{item.GetKind(), item.GetHost()})
	}
	json.NewEncoder(w).Encode(&logins)
}
