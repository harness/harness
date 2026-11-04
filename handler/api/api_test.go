package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/mock"
	"github.com/golang/mock/gomock"
)

func TestHandlerRejectsCrossOriginWrites(t *testing.T) {
	controller := gomock.NewController(t)
	session := mock.NewMockSession(controller)
	session.EXPECT().Get(gomock.Any()).Return(&core.User{ID: 1, Login: "admin", Admin: true}, nil).AnyTimes()
	users := mock.NewMockUserStore(controller)
	users.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, *core.User) error {
		t.Fatal("cross-origin request reached a state-changing handler")
		return nil
	}).AnyTimes()
	server := Server{
		Session: session, Users: users,
		System: &core.System{Proto: "https", Host: "ci.example.com", Link: "https://ci.example.com"},
	}
	// A simple POST can send same-site cookies without a CORS preflight.
	// Denying response access alone must not leave the write authorized.
	r := httptest.NewRequest(http.MethodPost, "https://ci.example.com/users/", strings.NewReader(`{"login":"attacker","admin":true,"machine":true}`))
	r.Header.Set("Origin", "https://evil.example.com")
	r.Header.Set("Content-Type", "text/plain")
	r.AddCookie(&http.Cookie{Name: "_session_", Value: "test-session"})
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("want cross-origin write rejected, got %d", w.Code)
	}
}

func TestHandlerAllowsTrustedWrites(t *testing.T) {
	for _, origin := range []string{"", "https://ci.example.com"} {
		t.Run(origin, func(t *testing.T) {
			controller := gomock.NewController(t)
			session := mock.NewMockSession(controller)
			session.EXPECT().Get(gomock.Any()).Return(&core.User{ID: 1, Login: "admin", Admin: true}, nil)
			users := mock.NewMockUserStore(controller)
			users.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			sender := mock.NewMockWebhookSender(controller)
			sender.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil)
			server := Server{
				Session: session, Users: users, Webhook: sender,
				System: &core.System{Proto: "https", Host: "ci.example.com", Link: "https://ci.example.com"},
			}
			r := httptest.NewRequest(http.MethodPost, "https://ci.example.com/users/", strings.NewReader(`{"login":"machine","machine":true}`))
			r.Header.Set("Origin", origin)
			w := httptest.NewRecorder()
			server.Handler().ServeHTTP(w, r)
			if w.Code != http.StatusOK {
				t.Fatalf("trusted write failed: %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestHandlerCORS(t *testing.T) {
	controller := gomock.NewController(t)
	session := mock.NewMockSession(controller)
	session.EXPECT().Get(gomock.Any()).Return(nil, nil).AnyTimes()
	server := Server{
		Session: session,
		System:  &core.System{Proto: "https", Host: "ci.example.com", Link: "https://ci.example.com"},
	}
	handler := server.Handler()
	for _, origin := range []string{
		"https://ci.example.com", "https://evil.example.com", "https://attacker.invalid",
		"https://ci.example.com.attacker.invalid", "http://ci.example.com", "https://ci.example.com:444", "null", "",
	} {
		for _, method := range []string{http.MethodGet, http.MethodOptions} {
			t.Run(method+"/"+origin, func(t *testing.T) {
				r := httptest.NewRequest(method, "https://ci.example.com/user", nil)
				r.Header.Set("Origin", origin)
				if method == http.MethodOptions {
					r.Header.Set("Access-Control-Request-Method", http.MethodPost)
					r.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
				}
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, r)
				if origin == "https://ci.example.com" {
					if got := w.Header().Get("Access-Control-Allow-Origin"); got != origin {
						t.Errorf("want allowed origin %q, got %q", origin, got)
					}
					if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
						t.Errorf("want credentials for the server origin, got %q", got)
					}
				} else if w.Header().Get("Access-Control-Allow-Origin") != "" || w.Header().Get("Access-Control-Allow-Credentials") != "" {
					t.Fatal("untrusted origin received credentialed CORS access")
				}
				if method == http.MethodGet && w.Code != http.StatusUnauthorized {
					t.Errorf("normal API authentication changed: got status %d", w.Code)
				}
			})
		}
	}
}
