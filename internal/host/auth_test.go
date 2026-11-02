package host

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
)

func newAuthTestServer(t *testing.T, a *Auth) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/login", a.Login)
	mux.HandleFunc("POST /api/auth/logout", a.Logout)
	mux.HandleFunc("GET /api/auth/session", a.Session)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/modules", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []string{})
	})
	mux.HandleFunc("POST /api/modules/install", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"ok": "1"})
	})
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "home")
	})
	return httptest.NewServer(a.Middleware(mux))
}

func newNoRedirectClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func loginRequest(t *testing.T, client *http.Client, base, password string) *http.Response {
	t.Helper()
	body := strings.NewReader(`{"password":"` + password + `"}`)
	req, err := http.NewRequest(http.MethodPost, base+"/api/auth/login", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestLoginSetsSessionCookie(t *testing.T) {
	a, _, err := NewAuth(t.TempDir(), "s3cret", false)
	if err != nil {
		t.Fatal(err)
	}
	srv := newAuthTestServer(t, a)
	defer srv.Close()
	client := newNoRedirectClient(t)

	resp := loginRequest(t, client, srv.URL, "wrong")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("错误口令应返回 401，实际 %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = loginRequest(t, client, srv.URL, "s3cret")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("正确口令应返回 200，实际 %d", resp.StatusCode)
	}
	raw := resp.Header.Get("Set-Cookie")
	for _, want := range []string{sessionCookie + "=", "HttpOnly", "SameSite=Lax", "Path=/"} {
		if !strings.Contains(raw, want) {
			t.Fatalf("Set-Cookie 缺少 %q: %s", want, raw)
		}
	}

	authed, err := client.Get(srv.URL + "/api/modules")
	if err != nil {
		t.Fatal(err)
	}
	defer authed.Body.Close()
	if authed.StatusCode != http.StatusOK {
		t.Fatalf("带 Cookie 访问应 200，实际 %d", authed.StatusCode)
	}

	anon, err := newNoRedirectClient(t).Get(srv.URL + "/api/modules")
	if err != nil {
		t.Fatal(err)
	}
	defer anon.Body.Close()
	if anon.StatusCode != http.StatusUnauthorized {
		t.Fatalf("未登录访问 API 应 401，实际 %d", anon.StatusCode)
	}
}

func TestSessionAndLogout(t *testing.T) {
	a, _, err := NewAuth(t.TempDir(), "s3cret", false)
	if err != nil {
		t.Fatal(err)
	}
	srv := newAuthTestServer(t, a)
	defer srv.Close()
	client := newNoRedirectClient(t)

	resp := loginRequest(t, client, srv.URL, "s3cret")
	resp.Body.Close()

	sess, err := client.Get(srv.URL + "/api/auth/session")
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]bool
	if err := json.NewDecoder(sess.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	sess.Body.Close()
	if !body["authenticated"] {
		t.Fatal("登录后会话状态应为 true")
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/logout", nil)
	if err != nil {
		t.Fatal(err)
	}
	out, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	out.Body.Close()

	after, err := client.Get(srv.URL + "/api/modules")
	if err != nil {
		t.Fatal(err)
	}
	defer after.Body.Close()
	if after.StatusCode != http.StatusUnauthorized {
		t.Fatalf("登出后应 401，实际 %d", after.StatusCode)
	}
}

func TestLoginRateLimit(t *testing.T) {
	a, _, err := NewAuth(t.TempDir(), "s3cret", false)
	if err != nil {
		t.Fatal(err)
	}
	srv := newAuthTestServer(t, a)
	defer srv.Close()
	client := newNoRedirectClient(t)

	var last int
	for i := 0; i < maxLoginFailures; i++ {
		resp := loginRequest(t, client, srv.URL, "wrong")
		last = resp.StatusCode
		resp.Body.Close()
	}
	if last != http.StatusUnauthorized {
		t.Fatalf("前 %d 次失败应为 401，实际 %d", maxLoginFailures, last)
	}
	resp := loginRequest(t, client, srv.URL, "wrong")
	deferred := resp.StatusCode
	resp.Body.Close()
	if deferred != http.StatusTooManyRequests {
		t.Fatalf("超过失败次数应 429，实际 %d", deferred)
	}
}

func TestOriginRejected(t *testing.T) {
	a, _, err := NewAuth(t.TempDir(), "s3cret", false)
	if err != nil {
		t.Fatal(err)
	}
	srv := newAuthTestServer(t, a)
	defer srv.Close()
	client := newNoRedirectClient(t)

	resp := loginRequest(t, client, srv.URL, "s3cret")
	resp.Body.Close()

	doPost := func(origin string) int {
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/modules/install", nil)
		if err != nil {
			t.Fatal(err)
		}
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		return res.StatusCode
	}

	if code := doPost("http://evil.example"); code != http.StatusForbidden {
		t.Fatalf("跨站写请求应 403，实际 %d", code)
	}
	if code := doPost(srv.URL); code != http.StatusOK {
		t.Fatalf("同源写请求应 200，实际 %d", code)
	}
	if code := doPost(""); code != http.StatusOK {
		t.Fatalf("无 Origin 的脚本调用应放行，实际 %d", code)
	}
}

func TestPageRedirectAndPublicPaths(t *testing.T) {
	a, _, err := NewAuth(t.TempDir(), "s3cret", false)
	if err != nil {
		t.Fatal(err)
	}
	srv := newAuthTestServer(t, a)
	defer srv.Close()
	client := newNoRedirectClient(t)

	resp, err := client.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != "/login" {
		t.Fatalf("未登录页面应跳登录，实际 %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}

	health, err := client.Get(srv.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	health.Body.Close()
	if health.StatusCode != http.StatusOK {
		t.Fatalf("健康检查应公开，实际 %d", health.StatusCode)
	}
}

func TestNewAuthGeneratesPasswordFile(t *testing.T) {
	root := t.TempDir()
	a, source, err := NewAuth(root, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source, "password") {
		t.Fatalf("口令来源应为文件路径，实际 %q", source)
	}
	if len(a.password) < 32 {
		t.Fatalf("生成口令过短: %d", len(a.password))
	}
}
