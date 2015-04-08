package session

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/drone/drone/common"
	"github.com/drone/drone/common/httputil"
	"github.com/drone/drone/settings"
	"github.com/gorilla/securecookie"
)

type Session interface {
	GenerateToken(*http.Request, *common.User) (string, error)
	GetLogin(*http.Request) string
}

type session struct {
	secret []byte
	expire time.Duration
}

func New(s *settings.Session) Session {
	secret := securecookie.GenerateRandomKey(32)
	expire := time.Hour * 72
	return &session{
		secret: secret,
		expire: expire,
	}
}

// GenerateToken generates a JWT token for the user session
// that can be appended to the #access_token segment to
// facilitate client-based OAuth2.
func (s *session) GenerateToken(r *http.Request, user *common.User) (string, error) {
	token := jwt.New(jwt.GetSigningMethod("HS256"))
	token.Claims["user_id"] = user.Login
	token.Claims["audience"] = httputil.GetURL(r)
	token.Claims["expires"] = time.Now().UTC().Add(s.expire).Unix()
	return token.SignedString(s.secret)
}

// GetLogin gets the currently authenticated user for the
// http.Request. The user details will be stored as either
// a simple API token or JWT bearer token.
func (s *session) GetLogin(r *http.Request) (_ string) {
	token := getToken(r)
	if len(token) == 0 {
		return
	}

	claims := getClaims(token, s.secret)
	if claims == nil || claims["user_id"] == nil {
		return
	}

	userid, ok := claims["user_id"].(string)
	if !ok {
		return
	}

	// tokenid, ok := claims["token_id"].(string)
	// if ok {
	// 	_, err := datastore.GetToken(c, int64(tokenid))
	// 	if err != nil {
	// 		return nil
	// 	}
	// }

	return userid
}

// getToken is a helper function that extracts the token
// from the http.Request.
func getToken(r *http.Request) string {
	token := getTokenHeader(r)
	if len(token) == 0 {
		token = getTokenParam(r)
	}
	return token
}

// getTokenHeader parses the JWT token value from
// the http Authorization header.
func getTokenHeader(r *http.Request) string {
	var tokenstr = r.Header.Get("Authorization")
	fmt.Sscanf(tokenstr, "Bearer %s", &tokenstr)
	return tokenstr
}

// getTokenParam parses the JWT token value from
// the http Request's query parameter.
func getTokenParam(r *http.Request) string {
	return r.FormValue("access_token")
}

// getClaims is a helper function that extracts the token
// claims from the JWT token string.
func getClaims(token string, secret []byte) map[string]interface{} {
	t, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil || !t.Valid {
		return nil
	}
	return t.Claims
}
