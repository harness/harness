package oauth1

import (
	"errors"
	"io"
	"io/ioutil"
	"net/url"
	"strconv"
)

type Token interface {
	Token()  string // Gets the oauth_token value.
	Secret() string // Gets the oauth_token_secret.
	Encode() string // Encode encodes the token into “URL encoded” form.
}

// AccessToken represents a value used by the Consumer to gain access
// to the Protected Resources on behalf of the User, instead of using
// the User's Service Provider credentials.
type AccessToken struct {
	token  string            // the oauth_token value
	secret string            // the oauth_token_secret value
	params map[string]string // additional params, as defined by the Provider.
}

// NewAccessToken returns a new instance of AccessToken with the specified
// token, secret and additional parameters.
func NewAccessToken(token, secret string, params map[string]string) *AccessToken {
	return &AccessToken {
		token  : token,
		secret : secret,
		params : params,
	}
}

// ParseAccessToken parses the URL-encoded query string from the Reader
// and returns an AccessToken.
func ParseAccessToken(reader io.ReadCloser) (*AccessToken, error) {
	body, err := ioutil.ReadAll(reader)
	reader.Close()
	if err != nil {
		return nil, err
	}

	return ParseAccessTokenStr(string(body))
}

// ParseAccessTokenStr parses the URL-encoded query string and returns
// an AccessToken.
func ParseAccessTokenStr(str string) (*AccessToken, error) {
	token := AccessToken{}
	token.params = map[string]string{}

	//parse the request token from the body
	parts, err := url.ParseQuery(str)
	if err != nil {
		return nil, err
	}

	//loop through parts to create Token
	for key, val := range parts {
		switch key {
		case "oauth_token"        : token.token = val[0]
		case "oauth_token_secret" : token.secret = val[0]
		default                   : token.params[key] = val[0]
		}
	}

	//some error checking ...
	switch {
	case len(token.token) == 0  : return nil, errors.New(str)
	case len(token.secret) == 0 : return nil, errors.New(str)
	}

	return &token, nil
}

// Encode encodes the values into “URL encoded” form of the AccessToken.
// e.g. "oauth_token=foo&oauth_token_secret=baz"
func (a *AccessToken) Encode() string {
	values := url.Values{}
	values.Set("oauth_token", a.token)
	values.Set("oauth_token_secret", a.secret)
	if a.params != nil {
		for key, val := range a.params {
			values.Set(key, val)
		}
	}
	return values.Encode()
}

// Gets the oauth_token value
func (a *AccessToken) Token()  string { return a.token }

// Gets the oauth_token_secret value
func (a *AccessToken) Secret() string { return a.secret }

// Gets any additional parameters, as defined by the Service Provider.
func (a *AccessToken) Params() map[string]string { return a.params }


// RequestToken represents a value used by the Consumer to obtain
// authorization from the User, and exchanged for an Access Token.
type RequestToken struct {
	token  string // the oauth_token value
	secret string // the oauth_token_secret value
	callbackConfirmed bool
}

// ParseRequestToken parses the URL-encoded query string from the Reader
// and returns a RequestToken.
func ParseRequestToken(reader io.ReadCloser) (*RequestToken, error) {
	body, err := ioutil.ReadAll(reader)
	reader.Close()
	if err != nil {
		return nil, err
	}

	return ParseRequestTokenStr(string(body))
}

// ParseRequestTokenStr parses the URL-encoded query string and returns
// a RequestToken.
func ParseRequestTokenStr(str string) (*RequestToken, error) {
	//parse the request token from the body
	parts, err := url.ParseQuery(str)
	if err != nil {
		return nil, err
	}

	token := RequestToken{}
	token.token  = parts.Get("oauth_token")
	token.secret = parts.Get("oauth_token_secret")
	token.callbackConfirmed = parts.Get("oauth_callback_confirmed") == "true"

	//some error checking ...
	switch {
	case len(token.token) == 0  : return nil, errors.New(str)
	case len(token.secret) == 0 : return nil, errors.New(str)
	}

	return &token, nil
}

// Encode encodes the values into “URL encoded” form of the ReqeustToken.
// e.g. "oauth_token=foo&oauth_token_secret=baz"
func (r *RequestToken) Encode() string {
	values := url.Values{}
	values.Set("oauth_token", r.token)
	values.Set("oauth_token_secret", r.secret)
	values.Set("oauth_callback_confirmed", strconv.FormatBool(r.callbackConfirmed))
	return values.Encode()
}

// Gets the oauth_token value
func (r *RequestToken) Token()  string { return r.token }

// Gets the oauth_token_secret value
func (r *RequestToken) Secret() string { return r.secret }
