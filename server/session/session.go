package session

import (
	"fmt"
	"net/http"
	"time"

	"code.google.com/p/go.net/context"
	"github.com/dgrijalva/jwt-go"
	"github.com/drone/config"
	"github.com/drone/drone/server/datastore"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/model"
	"github.com/gorilla/securecookie"
)

// random key used to create jwt if none
// provided in the configuration.
var random = securecookie.GenerateRandomKey(32)

var (
	secret  = config.String("session-secret", string(random))
	expires = config.Duration("session-expires", time.Hour*72)
)

// GetUser gets the currently authenticated user for the
// http.Request. The user details will be stored as either
// a simple API token or JWT bearer token.
func GetUser(c context.Context, r *http.Request) *model.User {
	switch {
	case r.Header.Get("Authorization") != "":
		return getUserBearer(c, r)
	case r.FormValue("access_token") != "":
		return getUserToken(c, r)
	default:
		return nil
	}
}

// GenerateToken generates a JWT token for the user session
// that can be appended to the #access_token segment to
// facilitate client-based OAuth2.
func GenerateToken(c context.Context, r *http.Request, user *model.User) (string, error) {
	token := jwt.New(jwt.GetSigningMethod("HS256"))
	token.Claims["user_id"] = user.ID
	token.Claims["audience"] = httputil.GetURL(r)
	token.Claims["expires"] = time.Now().UTC().Add(time.Hour * 72).Unix()
	return token.SignedString([]byte(*secret))
}

// getUserToken gets the currently authenticated user for the given
// auth token.
func getUserToken(c context.Context, r *http.Request) *model.User {
	var token = r.FormValue("access_token")
	var user = getUserJWT(c, token)
	if user != nil {
		return user
	}
	user, _ = datastore.GetUserToken(c, token)
	return user
}

// getUserBearer gets the currently authenticated user for the given
// bearer token (JWT)
func getUserBearer(c context.Context, r *http.Request) *model.User {
	var tokenstr = r.Header.Get("Authorization")
	fmt.Sscanf(tokenstr, "Bearer %s", &tokenstr)
	return getUserJWT(c, tokenstr)
}

// getUserJWT is a helper function that parses the User ID
// and retrieves the User data from a JWT Token.
func getUserJWT(c context.Context, token string) *model.User {
	var t, err = jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(*secret), nil
	})
	if err != nil || !t.Valid {
		return nil
	}
	var userid, ok = t.Claims["user_id"].(float64)
	if !ok {
		return nil
	}
	var user, _ = datastore.GetUser(c, int64(userid))
	return user
}
