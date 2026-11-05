package api

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/drone/drone/core"
	"github.com/drone/drone/mock"
	"github.com/drone/drone/session"
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
			if origin != "" {
				r.Header.Set("Origin", origin)
			}
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
					r.Header.Set("Access-Control-Request-Headers", "Content-Type")
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

func TestHandlerBrowserWriteProtection(t *testing.T) {
	for _, tt := range []struct {
		name    string
		headers http.Header
		allowed bool
		query   string
	}{
		{"public origin behind proxy", http.Header{"Origin": {"https://ci.example.com"}}, true, ""},
		{"equivalent default port", http.Header{"Origin": {"https://CI.EXAMPLE.COM:443"}}, true, ""},
		{"cross origin cookie", http.Header{"Origin": {"https://evil.example.com"}}, false, ""},
		{"null origin cookie", http.Header{"Origin": {"null"}}, false, ""},
		{"empty origin", http.Header{"Origin": {""}}, false, ""},
		{"duplicate origins", http.Header{"Origin": {"https://ci.example.com", "https://evil.example.com"}}, false, ""},
		{"origin with credentials", http.Header{"Origin": {"https://evil@ci.example.com"}}, false, ""},
		{"origin with path", http.Header{"Origin": {"https://ci.example.com/"}}, false, ""},
		{"wrong scheme", http.Header{"Origin": {"http://ci.example.com"}}, false, ""},
		{"wrong port", http.Header{"Origin": {"https://ci.example.com:444"}}, false, ""},
		{"non-browser without provenance headers", nil, true, ""},
		{"missing origin cross site", http.Header{"Sec-Fetch-Site": {"cross-site"}}, false, ""},
		{"missing origin sibling site", http.Header{"Sec-Fetch-Site": {"same-site"}}, false, ""},
		{"missing origin same origin", http.Header{"Sec-Fetch-Site": {"same-origin"}}, true, ""},
		{"missing origin trusted referer", http.Header{"Referer": {"https://ci.example.com/pipelines?tab=builds"}}, true, ""},
		{"missing origin hostile referer", http.Header{"Referer": {"https://evil.example.com/form"}}, false, ""},
		{"metadata cannot override referer", http.Header{"Sec-Fetch-Site": {"same-origin"}, "Referer": {"https://evil.example.com/form"}}, false, ""},
		{"forwarded host cannot grant trust", http.Header{"Origin": {"https://evil.example.com"}, "X-Forwarded-Host": {"evil.example.com"}, "X-Forwarded-Proto": {"https"}}, false, ""},
		{"bearer token cross origin", http.Header{"Origin": {"https://client.example.net"}, "Authorization": {"Bearer api-token"}}, true, ""},
		{"legacy header token cross origin", http.Header{"Origin": {"https://client.example.net"}, "Authorization": {"api-token"}}, true, ""},
		{"invalid token with valid cookie", http.Header{"Origin": {"https://client.example.net"}, "Authorization": {"Bearer invalid"}}, false, ""},
		{"legacy query token", http.Header{"Origin": {"https://client.example.net"}}, true, "?access_token=api-token"},
		{"invalid query token with valid cookie", http.Header{"Origin": {"https://client.example.net"}}, false, "?access_token=invalid"},
		{"invalid header overrides query token", http.Header{"Origin": {"https://client.example.net"}, "Authorization": {"Bearer invalid"}}, false, "?access_token=api-token"},
		{"opaque origin requires explicit token", http.Header{"Origin": {"null"}, "Authorization": {"Bearer api-token"}}, true, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			admin := &core.User{ID: 1, Login: "admin", Admin: true, Hash: "api-token"}
			users := mock.NewMockUserStore(controller)
			users.EXPECT().FindLogin(gomock.Any(), admin.Login).Return(admin, nil).AnyTimes()
			users.EXPECT().FindToken(gomock.Any(), admin.Hash).Return(admin, nil).AnyTimes()
			users.EXPECT().FindToken(gomock.Any(), "invalid").Return(nil, sql.ErrNoRows).AnyTimes()
			sessions := session.New(users, session.NewConfig("test-only-secret", time.Hour, true))
			cookie := httptest.NewRecorder()
			if err := sessions.Create(cookie, admin); err != nil {
				t.Fatal(err)
			}
			sender := mock.NewMockWebhookSender(controller)
			if tt.allowed {
				users.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
				sender.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil)
			}
			server := Server{Session: sessions, Users: users, Webhook: sender,
				System: &core.System{Proto: "https", Host: "ci.example.com"}}
			// The proxy serves HTTPS publicly and forwards HTTP to an internal host.
			r := httptest.NewRequest(http.MethodPost, "http://drone.internal:8080/users/"+tt.query, strings.NewReader(`{"login":"machine","machine":true}`))
			for name, values := range tt.headers {
				r.Header[name] = values
			}
			r.AddCookie(cookie.Result().Cookies()[0])
			w := httptest.NewRecorder()
			server.Handler().ServeHTTP(w, r)
			want := http.StatusForbidden
			if tt.allowed {
				want = http.StatusOK
			}
			if w.Code != want {
				t.Fatalf("want %d, got %d: %s", want, w.Code, w.Body.String())
			}
			if origin := tt.headers.Get("Origin"); tt.allowed && origin != "" && w.Header().Get("Access-Control-Allow-Origin") != origin {
				t.Error("authenticated client did not receive CORS access")
			}
			if !tt.allowed && w.Header().Get("Access-Control-Allow-Origin") != "" {
				t.Error("rejected request received CORS access")
			}
		})
	}
}

func TestHandlerFailsClosedWithoutPublicOrigin(t *testing.T) {
	for _, system := range []*core.System{nil, {}, {Proto: "https"}, {Host: "ci.example.com"}} {
		handler := browserSecurity(system)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) }))
		for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodOptions} {
			r := httptest.NewRequest(method, "http://drone.internal/user", nil)
			r.Header.Set("Origin", "https://attacker.example")
			r.Header.Set("Access-Control-Request-Method", "POST")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Header().Get("Access-Control-Allow-Origin") != "" || (method == http.MethodPost && w.Code != http.StatusForbidden) {
				t.Fatalf("missing public origin failed open: %s %d %v", method, w.Code, w.Header())
			}
		}
	}
}

func TestHandlerTokenPreflight(t *testing.T) {
	controller := gomock.NewController(t)
	sessions := mock.NewMockSession(controller)
	sessions.EXPECT().Get(gomock.Any()).Return(nil, nil).AnyTimes()
	handler := (Server{Session: sessions, System: &core.System{Proto: "https", Host: "ci.example.com"}}).Handler()
	for _, origin := range []string{"https://client.example.net", "null"} {
		for _, headers := range []string{"Content-Type", "Content-Type, authorization", "X-Authorization"} {
			r := httptest.NewRequest(http.MethodOptions, "http://drone.internal/user", nil)
			r.Header.Set("Origin", origin)
			r.Header.Set("Access-Control-Request-Method", "POST")
			r.Header.Set("Access-Control-Request-Headers", headers)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			allowed := w.Header().Get("Access-Control-Allow-Origin") == origin
			if allowed != (headers == "Content-Type, authorization") {
				t.Fatalf("unexpected preflight policy for %q: %v", headers, w.Header())
			}
		}
	}
}
