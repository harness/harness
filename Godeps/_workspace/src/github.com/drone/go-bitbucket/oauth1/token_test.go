package oauth1

import (
	"net/url"
	"strconv"
	"testing"
)

// Test the ability to parse a URL query string and unmarshal to a RequestToken.
func TestParseRequestTokenStr(t *testing.T) {
	oauth_token:="c0cf8793d39d46ab"
	oauth_token_secret:="FMMj3w7plPEyhK8ZZ9lBsp"
	oauth_callback_confirmed:=true

	values := url.Values{}
	values.Set("oauth_token", oauth_token)
	values.Set("oauth_token_secret", oauth_token_secret)
	values.Set("oauth_callback_confirmed", strconv.FormatBool(oauth_callback_confirmed))

	token, err := ParseRequestTokenStr(values.Encode())
	if err != nil {
		t.Errorf("Expected Request Token parsed, got Error %s", err.Error())
	}
	if token.token != oauth_token {
		t.Errorf("Expected Request Token %v, got %v", oauth_token, token.token)
	}
	if token.secret != oauth_token_secret {
		t.Errorf("Expected Request Token Secret %v, got %v", oauth_token_secret, token.secret)
	}
}

// Test the ability to Encode a RequestToken to a URL query string.
func TestEncodeRequestToken(t *testing.T) {
	token := RequestToken {
		token             : "c0cf8793d39d46ab",
		secret            : "FMMj3w7plPEyhK8ZZ9lBsp",
		callbackConfirmed : true,
	}

	tokenStr := token.Encode()
	expectedStr := "oauth_token_secret=FMMj3w7plPEyhK8ZZ9lBsp&oauth_token=c0cf8793d39d46ab&oauth_callback_confirmed=true"
	if tokenStr != expectedStr {
		t.Errorf("Expected Request Token Encoded as %v, got %v", expectedStr, tokenStr)
	}
}

// Test the ability to parse a URL query string and unmarshal to an AccessToken.
func TestEncodeAccessTokenStr(t *testing.T) {
	oauth_token:="c0cf8793d39d46ab"
	oauth_token_secret:="FMMj3w7plPEyhK8ZZ9lBsp"
	oauth_callback_confirmed:=true

	values := url.Values{}
	values.Set("oauth_token", oauth_token)
	values.Set("oauth_token_secret", oauth_token_secret)
	values.Set("oauth_callback_confirmed", strconv.FormatBool(oauth_callback_confirmed))

	token, err := ParseAccessTokenStr(values.Encode())
	if err != nil {
		t.Errorf("Expected Access Token parsed, got Error %s", err.Error())
	}
	if token.token != oauth_token {
		t.Errorf("Expected Access Token %v, got %v", oauth_token, token.token)
	}
	if token.secret != oauth_token_secret {
		t.Errorf("Expected Access Token Secret %v, got %v", oauth_token_secret, token.secret)
	}
}

// Test the ability to Encode an AccessToken to a URL query string.
func TestEncodeAccessToken(t *testing.T) {
	token := AccessToken {
		token  : "c0cf8793d39d46ab",
		secret : "FMMj3w7plPEyhK8ZZ9lBsp",
		params : map[string]string{ "user" : "dr_van_nostrand" },
	}

	tokenStr := token.Encode()
	expectedStr := "user=dr_van_nostrand&oauth_token_secret=FMMj3w7plPEyhK8ZZ9lBsp&oauth_token=c0cf8793d39d46ab"
	if tokenStr != expectedStr {
		t.Errorf("Expected Access Token Encoded as %v, got %v", expectedStr, tokenStr)
	}
}
