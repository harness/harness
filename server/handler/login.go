package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/drone/drone/plugin/remote"
	"github.com/drone/drone/server/database"
	"github.com/drone/drone/server/session"
	"github.com/drone/drone/shared/model"
	"github.com/gorilla/pat"
)

type LoginHandler struct {
	users database.UserManager
	repos database.RepoManager
	perms database.PermManager
	//conf    database.ConfigManager
	sess    session.Session
	remotes database.RemoteManager
}

func NewLoginHandler(users database.UserManager, repos database.RepoManager, perms database.PermManager, sess session.Session /*conf database.ConfigManager,*/, remotes database.RemoteManager) *LoginHandler {
	return &LoginHandler{users, repos, perms /*conf,*/, sess, remotes}
}

// GetLogin gets the login to the 3rd party remote system.
// GET /login/:host
func (h *LoginHandler) GetLogin(w http.ResponseWriter, r *http.Request) error {
	host := r.FormValue(":host")
	redirect := "/"

	remoteServer, err := h.remotes.FindType(host)
	if err != nil {
		return notFound{err}
	}

	remotePlugin, ok := remote.Lookup(remoteServer.Type)
	if !ok {
		return notFound{}
	}

	// get the remote system's client.
	plugin := remotePlugin(remoteServer)

	// authenticate the user
	login, err := plugin.GetLogin(w, r)
	if err != nil {
		return badRequest{err}
	} else if login == nil {
		// in this case we probably just redirected
		// the user, so we can exit with no error
		return nil
	}

	// get the user from the database
	u, err := h.users.FindLogin(host, login.Login)
	if err != nil {
		// if self-registration is disabled we should
		// return a notAuthorized error. the only exception
		// is if no users exist yet in the system we'll proceed.
		if remoteServer.Open == false && h.users.Exist() {
			return notAuthorized{}
		}

		// create the user account
		u = model.NewUser(plugin.GetName(), login.Login, login.Email)
		u.Name = login.Name
		u.SetEmail(login.Email)

		// insert the user into the database
		if err := h.users.Insert(u); err != nil {
			return badRequest{err}
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
	u.Syncing = u.IsStale()
	if err := h.users.Update(u); err != nil {
		return badRequest{err}
	}

	// look at the last synchronized date to determine if
	// we need to re-sync the account.
	//
	// todo(bradrydzewski) this should move to a server/sync package and
	//      should be injected into this struct, just like the database code.
	//
	// todo(bradrydzewski) this login should be a bit more intelligent
	//      than the current implementation.
	//
	// todo(bradrydzewski) the github implementation will only sync a
	//      maximum of 100 repositories due to the api pagination. need to fix.
	if u.Syncing {
		redirect = "/sync"
		log.Println("sync user account.", u.Login)

		// sync inside a goroutine. This should eventually be moved to
		// its own package / sync utility.
		go func() {
			// list all repositories
			client := plugin.GetClient(u.Access, u.Secret)
			repos, err := client.GetRepos("")
			if err != nil {
				log.Println("Error syncing user account, listing repositories", u.Login, err)
				return
			}

			// insert all repositories
			for _, remoteRepo := range repos {
				repo, _ := model.NewRepo(plugin.GetName(), remoteRepo.Owner, remoteRepo.Name)
				repo.Private = remoteRepo.Private
				repo.Host = remoteRepo.Host
				repo.CloneURL = remoteRepo.Clone
				repo.GitURL = remoteRepo.Git
				repo.SSHURL = remoteRepo.SSH
				repo.URL = remoteRepo.URL

				if err := h.repos.Insert(repo); err != nil {
					log.Println("Error adding repo.", u.Login, remoteRepo.Name, err)
					// this is probably because the repository already exists.
					// we should still attempt to grant the user access, however.
					//
					// todo(bradrydzewski) we should inspect the response to ensure
					//      the failure is caused by a primary key violation.
				}

				// add user permissions
				if err := h.perms.Grant(u, repo, remoteRepo.Pull, remoteRepo.Push, remoteRepo.Admin); err != nil {
					log.Println("Error adding permissions.", u.Login, remoteRepo.Name, err)
					continue
				}

				log.Println("Successfully syced repo.", u.Login+"/"+remoteRepo.Name)
			}

			u.Synced = time.Now().UTC().Unix()
			u.Syncing = false
			if err := h.users.Update(u); err != nil {
				log.Println("Error syncing user account, updating sync date", u.Login, err)
				return
			}
		}()
	}

	// (re)-create the user session
	h.sess.SetUser(w, r, u)

	// redirect the user to their dashboard
	http.Redirect(w, r, redirect, http.StatusSeeOther)
	return nil
}

// GetLogout terminates the current user session
// GET /logout
func (h *LoginHandler) GetLogout(w http.ResponseWriter, r *http.Request) error {
	h.sess.Clear(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
	return nil
}

func (h *LoginHandler) Register(r *pat.Router) {
	r.Get("/login/{host}", errorHandler(h.GetLogin))
	r.Get("/logout", errorHandler(h.GetLogout))
}
