// auth.go 提供单用户口令认证：口令换会话 Cookie，写请求做 Origin 校验。
package host

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	sessionCookie    = "mica_session"
	sessionTTL       = 7 * 24 * time.Hour
	maxLoginFailures = 5
	loginLockWindow  = time.Minute
)

// Auth 是宿主唯一的认证入口。会话只存内存，重启后需重新登录。
type Auth struct {
	password string
	secure   bool

	mu       sync.Mutex
	sessions map[string]time.Time
	failures map[string]*loginFailure
}

type loginFailure struct {
	count int
	until time.Time
}

// NewAuth 创建认证器。password 为空时从 dataRoot/auth/password 读取，
// 文件不存在则生成随机口令并落盘（0600）。返回口令来源，供启动日志提示路径。
func NewAuth(dataRoot, password string, secure bool) (*Auth, string, error) {
	a := &Auth{
		secure:   secure,
		sessions: make(map[string]time.Time),
		failures: make(map[string]*loginFailure),
	}
	if password != "" {
		a.password = password
		return a, "环境变量 MICA_ADMIN_PASSWORD", nil
	}
	path := filepath.Join(dataRoot, "auth", "password")
	if data, err := os.ReadFile(path); err == nil {
		if v := strings.TrimSpace(string(data)); v != "" {
			a.password = v
			return a, path, nil
		}
	}
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return nil, "", fmt.Errorf("生成口令失败: %w", err)
	}
	a.password = hex.EncodeToString(b)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, "", fmt.Errorf("创建口令目录失败: %w", err)
	}
	if err := os.WriteFile(path, []byte(a.password+"\n"), 0o600); err != nil {
		return nil, "", fmt.Errorf("写入口令文件失败: %w", err)
	}
	return a, path + "（本次生成）", nil
}

// Middleware 拦截未认证请求：API 返回 401，页面跳转登录页。
func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if !safeMethod(r.Method) && !sameOrigin(r) {
			writeError(w, http.StatusForbidden, "来源被拒绝")
			return
		}
		if !a.authenticated(r) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				writeError(w, http.StatusUnauthorized, "未登录")
				return
			}
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Login 校验口令并签发会话 Cookie。
func (a *Auth) Login(w http.ResponseWriter, r *http.Request) {
	if !a.allowAttempt(r) {
		writeError(w, http.StatusTooManyRequests, "尝试过于频繁，请稍后再试")
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if subtle.ConstantTimeCompare([]byte(body.Password), []byte(a.password)) != 1 {
		a.noteFailure(r)
		writeError(w, http.StatusUnauthorized, "口令不正确")
		return
	}
	a.clearFailure(r)

	token := newSessionToken()
	a.mu.Lock()
	a.pruneSessions()
	a.sessions[token] = time.Now().Add(sessionTTL)
	a.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   a.secure,
		MaxAge:   int(sessionTTL / time.Second),
	})
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": true})
}

// Logout 销毁当前会话并清除 Cookie。
func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		a.mu.Lock()
		delete(a.sessions, c.Value)
		a.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   a.secure,
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
}

// Session 返回当前会话状态，供登录页与前端初始化判断。
func (a *Auth) Session(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": a.authenticated(r)})
}

func (a *Auth) authenticated(r *http.Request) bool {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	exp, ok := a.sessions[c.Value]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(a.sessions, c.Value)
		return false
	}
	return true
}

// pruneSessions 需在持有 a.mu 时调用。
func (a *Auth) pruneSessions() {
	now := time.Now()
	for token, exp := range a.sessions {
		if now.After(exp) {
			delete(a.sessions, token)
		}
	}
}

func (a *Auth) allowAttempt(r *http.Request) bool {
	ip := clientIP(r)
	a.mu.Lock()
	defer a.mu.Unlock()
	f := a.failures[ip]
	return f == nil || time.Now().After(f.until)
}

func (a *Auth) noteFailure(r *http.Request) {
	ip := clientIP(r)
	a.mu.Lock()
	defer a.mu.Unlock()
	f := a.failures[ip]
	if f == nil {
		f = &loginFailure{}
		a.failures[ip] = f
	}
	f.count++
	if f.count >= maxLoginFailures {
		f.count = 0
		f.until = time.Now().Add(loginLockWindow)
	}
}

func (a *Auth) clearFailure(r *http.Request) {
	ip := clientIP(r)
	a.mu.Lock()
	delete(a.failures, ip)
	a.mu.Unlock()
}

// isPublicPath 是无需登录即可访问的路径（登录页、认证接口、健康检查、文档与公共静态资源）。
func isPublicPath(p string) bool {
	switch p {
	case "/login", "/api/auth/login", "/api/auth/session", "/api/health",
		"/theme.css", "/theme.js", "/docs.css", "/docs.js":
		return true
	}
	return p == "/docs" || strings.HasPrefix(p, "/docs/") || strings.HasPrefix(p, "/fonts/")
}

// sameOrigin 校验写请求的 Origin；浏览器跨站写会被 SameSite=Lax 挡下，这里再补一层。
// 非浏览器调用通常不带 Origin，此时放行，避免破坏脚本/监控。
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

func safeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}

func clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func newSessionToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}
