package oauth2

// Out-Of-Band mode, used for applications that do not have
// a callback URL, such as mobile phones or command-line
// utilities.
const OOB = "urn:ietf:wg:oauth:2.0:oob"

// Enumerates authorization grants (grant_type) used by the
// client to obtain an access token.
const (
	// grant_type for requesting a refresh_token
	GrantTypeRefreshToken = "refresh_token"

	// grant_type for requesting an access_token using an
	// authorization code.
	GrantTypeAuthorizationCode = "authorization_code"

	// grant_type for requesting access to resources owned by the
	// registered application (client). 
	GrantTypeClientCredentials = "client_credentials"

	// grant_type for exchanging a username and password for
	// an access_token
	GrantTypePassword = "password"
)

const (
	// response_type for requesting an authoriztion code
	ResponseTypeCode = "code"

	// response_type for requesting or refreshing an access_token
	ResponseTypeToken = "token"
)

const (
	TokenBearer = "bearer"
	TokenMac    = "mac"
)

// Enumerates ASCII [USASCII] error code returned by the
// OAuth2.0 Error Response.
const (
	// The request is missing a required parameter, includes an
	// unsupported parameter value (other than grant type), repeats
	// a parameter, includes multiple credentials, utilizes more than
	// one mechanism for authenticating the client, or is otherwise
	// malformed.
	ErrorCodeInvalidRequest = "invalid_request"

	// Client authentication failed (e.g. unknown client, no client
	// authentication included, or unsupported authentication method).
	ErrorCodeInvalidClient = "invalid_client"

	// The provided authorization grant (e.g. authorization
	// code, resource owner credentials) or refresh token is
	// invalid, expired, revoked, does not match the redirection
	// URI used in the authorization request, or was issued to
	// another client.
	ErrorCodeInvalidGrant = "invalid_grant"

	// The authenticated client is not authorized to use this
	// authorization grant type.
	ErrorCodeUnauthorizedClient = "unauthorized_client"

	// The authorization grant type is not supported by the
	// authorization server.
	ErrorCodeUnsupportedGrantType = "unsupported_grant_type"

	// The requested scope is invalid, unknown, malformed, or
	// exceeds the scope granted by the resource owner.
	ErrorCodeInvalidScope = "invalid_scope"
)
