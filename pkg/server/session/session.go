package session

import (
	"fmt"
	"net/http"
	"time"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/dgrijalva/jwt-go"
	"github.com/drone/drone/pkg/config"
	common "github.com/drone/drone/pkg/types"
)

type Session interface {
	GenerateToken(*common.Token) (string, error)
	GetLogin(*http.Request) *common.Token
}

type session struct {
	secret []byte
	expire time.Duration
}

func New(s *config.Config) Session {
	secret := []byte(s.Session.Secret)
	expire := time.Hour * 72
	return &session{
		secret: secret,
		expire: expire,
	}
}

// GenerateToken generates a JWT token for the user session
// that can be appended to the #access_token segment to
// facilitate client-based OAuth2.
func (s *session) GenerateToken(t *common.Token) (string, error) {
	token := jwt.New(jwt.GetSigningMethod("HS256"))
	token.Claims["user"] = t.Login
	token.Claims["kind"] = t.Kind
	token.Claims["date"] = t.Issued
	token.Claims["label"] = t.Label
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
	if claims == nil || claims["user"] == nil || claims["date"] == nil || claims["label"] == nil || claims["kind"] == nil {
		return nil
	}

	token := &common.Token{
		Kind:   claims["kind"].(string),
		Login:  claims["user"].(string),
		Label:  claims["label"].(string),
		Issued: int64(claims["date"].(float64)),
	}
	if token.Kind != common.TokenSess {
		return token
	}

	if time.Unix(token.Issued, 0).Add(s.expire).Before(time.Now()) {
		return nil
	}
	return token
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
