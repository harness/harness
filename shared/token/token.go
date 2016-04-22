package token

import (
	"fmt"
	"net/http"

	"github.com/dgrijalva/jwt-go"
)

type SecretFunc func(*Token) (string, error)

const (
	UserToken  = "user"
	SessToken  = "sess"
	HookToken  = "hook"
	CsrfToken  = "csrf"
	AgentToken = "agent"
)

// Default algorithm used to sign JWT tokens.
const SignerAlgo = "HS256"

type Token struct {
	Kind string
	Text string
}

func Parse(raw string, fn SecretFunc) (*Token, error) {
	token := &Token{}
	parsed, err := jwt.Parse(raw, keyFunc(token, fn))
	if err != nil {
		return nil, err
	} else if !parsed.Valid {
		return nil, jwt.ValidationError{}
	}
	return token, nil
}

func ParseRequest(r *http.Request, fn SecretFunc) (*Token, error) {
	var token = r.Header.Get("Authorization")

	// first we attempt to get the token from the
	// authorization header.
	if len(token) != 0 {
		token = r.Header.Get("Authorization")
		fmt.Sscanf(token, "Bearer %s", &token)
		return Parse(token, fn)
	}

	// then we attempt to get the token from the
	// access_token url query parameter
	token = r.FormValue("access_token")
	if len(token) != 0 {
		return Parse(token, fn)
	}

	// and finally we attempt to get the token from
	// the user session cookie
	cookie, err := r.Cookie("user_sess")
	if err != nil {
		return nil, err
	}
	return Parse(cookie.Value, fn)
}

func CheckCsrf(r *http.Request, fn SecretFunc) error {

	// get and options requests are always
	// enabled, without CSRF checks.
	switch r.Method {
	case "GET", "OPTIONS":
		return nil
	}

	// parse the raw CSRF token value and validate
	raw := r.Header.Get("X-CSRF-TOKEN")
	_, err := Parse(raw, fn)
	return err
}

func New(kind, text string) *Token {
	return &Token{Kind: kind, Text: text}
}

// Sign signs the token using the given secret hash
// and returns the string value.
func (t *Token) Sign(secret string) (string, error) {
	return t.SignExpires(secret, 0)
}

// Sign signs the token using the given secret hash
// with an expiration date.
func (t *Token) SignExpires(secret string, exp int64) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims["type"] = t.Kind
	token.Claims["text"] = t.Text
	if exp > 0 {
		token.Claims["exp"] = float64(exp)
	}
	return token.SignedString([]byte(secret))
}

func keyFunc(token *Token, fn SecretFunc) jwt.Keyfunc {
	return func(t *jwt.Token) (interface{}, error) {
		// validate the correct algorithm is being used
		if t.Method.Alg() != SignerAlgo {
			return nil, jwt.ErrSignatureInvalid
		}

		// extract the token kind and cast to
		// the expected type.
		kindv, ok := t.Claims["type"]
		if !ok {
			return nil, jwt.ValidationError{}
		}
		token.Kind, _ = kindv.(string)

		// extract the token value and cast to
		// exepected type.
		textv, ok := t.Claims["text"]
		if !ok {
			return nil, jwt.ValidationError{}
		}
		token.Text, _ = textv.(string)

		// invoke the callback function to retrieve
		// the secret key used to verify
		secret, err := fn(token)
		return []byte(secret), err
	}
}
