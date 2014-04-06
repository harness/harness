package oauth2

// see https://github.com/litl/rauth/blob/master/examples/twitter-timeline-cli.py
// see http://tools.ietf.org/html/draft-ietf-oauth-v2-31

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Client represents an application making protected resource requests on
// behalf of the resource owner and with its authorization.
type Client struct {
	// The client_identifier issued to the client during the
	// registration process.
	ClientId string

	// The client_secret issued to the client during the
	// registration process.
	ClientSecret string

	// Used by the authorization server to return authorization credentials
	// responses to the client via the resource owner user-agent
	RedirectURL string

	// Used by the client to exchange an authorization grant for
	// an access token, typically with client authentication.
	AccessTokenURL string

	// Used by the client to obtain authorization from the resource
	// owner via user-agent redirection.
	AuthorizationURL string
}

// AuthorizeRedirect constructs the Authorization Endpoint, where the user
// can authorize the client to access protected resources.
func (c *Client) AuthorizeRedirect(scope, state string) string {
	// add required parameters
	params := make(url.Values)
	params.Add("response_type", ResponseTypeCode)
	//params.Set("redirect_uri", c.RedirectURL)
	params.Set("client_id", c.ClientId)

	// add optional redirect param
	// NOTE: this is optional for some providers, but not for others
	if len(c.RedirectURL) > 0 {
		params.Set("redirect_uri", c.RedirectURL)
	}

	// add optional scope param
	if len(scope) > 0 {
		params.Set("scope", scope)
	}

	// add optional state param
	if len(state) > 0 {
		params.Set("state", state)
	}

	// HACK: for google we must add access_type=offline in order
	//       to obtain a refresh token
	if strings.HasPrefix(c.AuthorizationURL, "https://accounts.google.com") {
		params.Set("access_type", "offline")
		//params.Set("approval_prompt","force")
	}

	// generate the URL
	endpoint, _ := url.Parse(c.AuthorizationURL)
	endpoint.RawQuery = params.Encode()

	//HACK: Google separates scopes using a "+", however, these get
	//      encoded and for some reason cause Google to fail the request.
	//      So we will decode all plus signs to make the Google happy
	endpoint.RawQuery = strings.Replace(endpoint.RawQuery, "%2B", "+", -1)
	return endpoint.String()
}

// AuthorizeRedirect redirects an http.Request to the Authorizatioin Endpoint,
// where the user can authorize the client to access protected resources.
//func (c *Client) AuthorizeRedirect(w http.ResponseWriter, r *http.Request, scope, state string) {
//	http.Redirect(w, r, c.GetAuthorizeRedirect(scope, state), http.StatusSeeOther)
//}

// GrantToken will attempt to grant an Access Token using
// the specified authorization code.
func (c *Client) GrantToken(code string) (*Token, error) {
	params := make(url.Values)
	params.Set("grant_type", GrantTypeAuthorizationCode)
	params.Set("code", code)
	params.Set("scope", "")
	return c.grantToken(params)
}

// GrantTokenCredentials will attempt to grant an Access Token
// for the Client to access protected resources the the Client owns.
//
// See http://tools.ietf.org/html/draft-ietf-oauth-v2-31#section-4.3
func (c *Client) GrantTokenCredentials(scope string) (*Token, error) {
	params := make(url.Values)
	params.Set("grant_type", GrantTypeClientCredentials)
	params.Set("scope", scope)
	return c.grantToken(params)
}

// GrantTokenPassword will attempt to grant an Access Token using the
// resource owner's credentials (username and password). The scope of
// the access request may be optinally included, or left empty.
//
// See http://tools.ietf.org/html/draft-ietf-oauth-v2-31#section-4.3
func (c *Client) GrantTokenPassword(username, password, scope string) (*Token, error) {
	params := make(url.Values)
	params.Set("grant_type", GrantTypePassword)
	params.Set("username", username)
	params.Set("password", password)
	params.Set("scope", scope)
	return c.grantToken(params)
}

// RefreshToken requests a new access token by authenticating with
// the authorization server and presenting the refresh token.
func (c *Client) RefreshToken(refreshToken string) (*Token, error) {
	params := make(url.Values)
	params.Set("grant_type", GrantTypeRefreshToken)
	params.Set("refresh_token", refreshToken)
	params.Set("scope", "")
	return c.grantToken(params)
}

// Token represents a successful response to an OAuth2.0 Access
// Token Request, including a Refresh Token request.
type Token struct {
	// The access token issued by the authorization server.
	AccessToken string `json:"access_token"`

	// The type of the token issued (bearer, mac, etc)
	TokenType string `json:"token_type"`

	// The refresh token, which can be used to obtain new
	// access tokens using the same authorization grant
	RefreshToken string `json:"refresh_token"`

	// The lifetime in seconds of the access token.  For
	// example, the value "3600" denotes that the access token will
	// expire in one hour from the time the response was generated.
	ExpiresIn int64 `json:"expires_in"`

	// The scope of the access token.
	Scope string
}

func (t Token) Token() string {
	return t.AccessToken
}

// Error represents a failed request to the OAuth2.0 Authorization
// or Resource server.
type Error struct {
	// A single ASCII [USASCII] error code
	Code string `json:"error"`

	// A human-readable ASCII [USASCII] text providing
	// additional information, used to assist the client developer in
	// understanding the error that occurred.
	Description string `json:"error_description"`

	// A URI identifying a human-readable web page with
	// information about the error, used to provide the client
	// developer with additional information about the error.
	URI string `json:"error_uri"`
}

// Error returns a string representation of the OAuth2
// error message.
func (e Error) Error() string {
	return e.Code
}

// helper function to retrieve a token from the server
func (c *Client) grantToken(params url.Values) (*Token, error) {
	// Create the access token url params
	if params == nil {
		params = make(url.Values)
	}

	// Add the client id, client secret and code to the query params
	params.Set("client_id", c.ClientId)
	params.Set("client_secret", c.ClientSecret)
	params.Set("redirect_uri", c.RedirectURL)

	// Create the access token request url
	endpoint, _ := url.Parse(c.AccessTokenURL)

	// Create the http request
	req := http.Request{
		URL:        endpoint,
		Method:     "POST",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Close:      true,
	}

	// Encode the URL paraemeters in the Body of the Request
	encParams := params.Encode()
	reader := strings.NewReader(encParams)
	req.Body = ioutil.NopCloser(reader)

	// Add the header params
	header := make(http.Header)
	header.Set("Accept", "application/json")
	header.Set("Content-Type", "application/x-www-form-urlencoded")
	header.Set("Content-length", strconv.Itoa(len(encParams)))
	req.Header = header

	// Do the http request and get the response
	resp, err := http.DefaultClient.Do(&req)
	if err != nil {
		return nil, err
	}

	// Get the response body
	raw, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	// Unmarshal the json response body to get the token
	token := Token{}
	if err := json.Unmarshal(raw, &token); err != nil {
		return nil, err
	}

	// If no access token is provided it must be an error. Normally
	// we would check the StatusCode, however, some providers return
	// a 200 Status OK even if there is an error :(
	if len(token.AccessToken) == 0 {
		oauthError := Error{}
		if err := json.Unmarshal(raw, &oauthError); err != nil {
			return nil, err
		}
		return nil, oauthError
	}

	return &token, nil
}
