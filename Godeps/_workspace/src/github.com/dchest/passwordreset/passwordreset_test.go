package passwordreset

import (
	"errors"
	"testing"
	"time"
)

var (
	testLogin      = "test user"
	testPwdVal     = []byte("test password value")
	testSecret     = []byte("secret key")
	testLoginError = errors.New("test unknown login error")
)

func getPwdVal(login string) ([]byte, error) {
	if login == testLogin {
		return testPwdVal, nil
	}
	return testPwdVal, testLoginError
	//     ^ return it anyway to test that it's not begin used
}

func TestNew(t *testing.T) {
	pwdVal, _ := getPwdVal(testLogin)
	token := NewToken(testLogin, 100 * time.Second, pwdVal, testSecret)
	login, err := VerifyToken(token, getPwdVal, testSecret)
	if err != nil {
		t.Errorf("unexpected error %q", err)
	}
	if login != testLogin {
		t.Errorf("login: expected %q, got %q", testLogin, login)
	}
}

func TestVerify(t *testing.T) {
	bad := []string{
		"",
		"bad token",
		"Talo3mRjaGVzdITUAGOXYZwCMq7EtHfYH4ILcBgKaoWXDHTJOIlBUfcr",
	}
	for i, token := range bad {
		login, err := VerifyToken(token, getPwdVal, testSecret)
		if login != "" {
			t.Errorf(`%d: login for bad token: expected "", got %q`, i, login)
		}
		if err == nil {
			t.Errorf("%d: expected error")
		}
	}
	// Test expiration
	pwdVal, _ := getPwdVal(testLogin)
	token := NewToken(testLogin, -1, pwdVal, testSecret)
	if _, err := VerifyToken(token, getPwdVal, testSecret); err == nil {
		t.Errorf("verified expired token")
	}
	// Test wrong password value
	pwdVal = []byte("wrong value")
	token = NewToken(testLogin, -1, pwdVal, testSecret)
	if _, err := VerifyToken(token, getPwdVal, testSecret); err == nil {
		t.Errorf("verified with wrong password value")
	}
	// Test password value error return
	login := "unknown login"
	_, errVal := getPwdVal(login)
	token = NewToken(login, 100 * time.Second, testPwdVal, testSecret)
	if _, err := VerifyToken(token, getPwdVal, testSecret); err != errVal {
		t.Errorf("err: expected %q, got %q", errVal, err)
	}
}
