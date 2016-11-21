package session

import (
	"github.com/drone/drone/cache"
	"github.com/drone/drone/model"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
)

func TeamPerm(c *gin.Context) *model.Perm {
	user := User(c)
	team := c.Param("team")
	perm := &model.Perm{}

	switch {
		// if the user is not authenticated
		case user == nil:
			perm.Admin = false
			perm.Pull  = false
			perm.Push  = false

		// if the user is a DRONE_ADMIN
		case user.Admin:
			perm.Admin = true
			perm.Pull  = true
			perm.Push  = true

		// otherwise if the user is authenticated we should
		// check the remote system to get the users permissiosn.
		default:
			log.Debugf("Fetching team permission for %s %s",
				user.Login, team)

			var err error
			perm, err = cache.GetTeamPerms(c, user, team)
			if err != nil {
				// debug
				log.Errorf("Error fetching team permission for %s %s",
					user.Login, team)

				perm.Admin = false
				perm.Pull  = false
				perm.Push  = false
			}
	}

	if user != nil {
		log.Debugf("%s granted %+v team permission to %s",
			user.Login, perm, team)
	} else {
		log.Debugf("Guest granted %+v to %s", perm, team)

		perm.Admin = false
		perm.Pull  = false
		perm.Push  = false
	}

	return perm
}

func MustTeamAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		perm := TeamPerm(c)

		if perm.Admin {
			c.Next()
		} else {
			c.String(401, "User not authorized")
			c.Abort()	
		}
	}
}
