package sync

import (
	"log"
	"time"

	"code.google.com/p/go.net/context"
	"github.com/drone/drone/plugin/remote"
	"github.com/drone/drone/server/datastore"
	"github.com/drone/drone/shared/model"
)

func SyncUser(ctx context.Context, user *model.User, remote remote.Remote) {
	repos, err := remote.GetRepos(user)
	if err != nil {
		log.Println("Error syncing user account, listing repositories", user.Login, err)
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
				log.Println("Error adding repo.", user.Login, repo.Name, err)
				continue
			}
		}

		// add user permissions
		perm := model.Perm{
			UserID: user.ID,
			RepoID: repo.ID,
			Read:   role.Read,
			Write:  role.Write,
			Admin:  role.Admin,
		}
		if err := datastore.PostPerm(ctx, &perm); err != nil {
			log.Println("Error adding permissions.", user.Login, repo.Name, err)
			continue
		}

		log.Println("Successfully syced repo.", user.Login+"/"+repo.Name)
	}

	user.Synced = time.Now().UTC().Unix()
	user.Syncing = false
	if err := datastore.PutUser(ctx, user); err != nil {
		log.Println("Error syncing user account, updating sync date", user.Login, err)
		return
	}
}
