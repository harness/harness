package session

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/drone/drone/common"
	"github.com/drone/drone/settings"
	"github.com/gorilla/securecookie"
)

type Session interface {
	GenerateToken(*http.Request, *common.Token) (string, error)
	GetLogin(*http.Request) *common.Token
}

type session struct {
	secret []byte
	expire time.Duration
}

func New(s *settings.Session) Session {
	// TODO (bradrydzewski) hook up the Session.Expires
	secret := []byte(s.Secret)
	expire := time.Hour * 72
	if len(secret) == 0 {
		securecookie.GenerateRandomKey(32)
	}
	return &session{
		secret: secret,
		expire: expire,
	}
}

// GenerateToken generates a JWT token for the user session
// that can be appended to the #access_token segment to
// facilitate client-based OAuth2.
func (s *session) GenerateToken(r *http.Request, t *common.Token) (string, error) {
	token := jwt.New(jwt.GetSigningMethod("HS256"))
	token.Claims["login"] = t.Login
	token.Claims["expiry"] = t.Expiry

	// add optional repos that can be
	// access from this session.
	if len(t.Repos) != 0 {
		token.Claims["repos"] = t.Repos
	}
	// add optional scopes that can be
	// applied to this session.
	if len(t.Scopes) != 0 {
		token.Claims["scope"] = t.Scopes
	}
	return token.SignedString(s.secret)
}

// GetLogin gets the currently authenticated user for the
// http.Request. The user details will be stored as either
// a simple API token or JWT bearer token.
func (s *session) GetLogin(r *http.Request) *common.Token {
	t := getToken(r)
	if len(t) == 0 {
		return nil
	}

	claims := getClaims(t, s.secret)
	if claims == nil || claims["login"] == nil {
		return nil
	}

	loginv, ok := claims["login"]
	if !ok {
		return nil
	}

	loginstr, ok := loginv.(string)
	if !ok {
		return nil
	}

	return &common.Token{Login: loginstr}
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
