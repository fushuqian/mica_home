package sidecar

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestProxy(t *testing.T, target string) *Manager {
	t.Helper()
	p := &process{
		addr:  "tcp://" + strings.TrimPrefix(target, "http://"),
		token: "sidecar-token",
		state: "running",
	}
	return &Manager{procs: map[string]*process{"demo": p}}
}

func TestProxyPassesPathQueryBodyAndToken(t *testing.T) {
	var gotPath, gotQuery, gotMethod, gotBody, gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotPath, gotQuery, gotMethod = r.URL.Path, r.URL.RawQuery, r.Method
		gotBody, gotToken = string(body), r.Header.Get("X-Mica-Token")
		w.Header().Set("X-Sidecar", "yes")
		w.WriteHeader(http.StatusTeapot)
		io.WriteString(w, "ok")
	}))
	defer srv.Close()

	m := newTestProxy(t, srv.URL)
	proxy, ok := m.Proxy("demo", "/api/m/demo")
	if !ok {
		t.Fatal("proxy 创建失败")
	}
	req := httptest.NewRequest(http.MethodPost, "/api/m/demo/v1/items?q=2", strings.NewReader("hello"))
	req.Header.Set("X-Mica-Token", "from-client")
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if gotPath != "/v1/items" || gotQuery != "q=2" || gotMethod != http.MethodPost || gotBody != "hello" {
		t.Fatalf("转发内容不符: path=%q query=%q method=%q body=%q", gotPath, gotQuery, gotMethod, gotBody)
	}
	if gotToken != "sidecar-token" {
		t.Fatalf("token 未替换: %q", gotToken)
	}
	if rec.Code != http.StatusTeapot || rec.Header().Get("X-Sidecar") != "yes" {
		t.Fatalf("响应未透传: code=%d header=%q", rec.Code, rec.Header().Get("X-Sidecar"))
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("响应体不符: %q", rec.Body.String())
	}
}

func TestProxyUnknownModule(t *testing.T) {
	m := &Manager{procs: map[string]*process{"demo": {addr: "tcp://127.0.0.1:1", state: "stopped"}}}
	if _, ok := m.Proxy("demo", "/api/m/demo"); ok {
		t.Fatal("已停止的 sidecar 不应创建 proxy")
	}
}

func TestProxyWebSocketUpgrade(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			http.Error(w, "missing upgrade", http.StatusBadRequest)
			return
		}
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "no hijacker", http.StatusInternalServerError)
			return
		}
		conn, buf, err := hj.Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		fmt.Fprintf(buf, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nX-Mica-Token: %s\r\n\r\n", r.Header.Get("X-Mica-Token"))
		buf.Flush()
		line, err := buf.ReadString('\n')
		if err == nil {
			buf.WriteString("echo:" + line)
			buf.Flush()
		}
	}))
	defer srv.Close()

	m := newTestProxy(t, srv.URL)
	proxy, ok := m.Proxy("demo", "/api/m/demo")
	if !ok {
		t.Fatal("proxy 创建失败")
	}
	front := httptest.NewServer(proxy)
	defer front.Close()

	conn, err := net.Dial("tcp", strings.TrimPrefix(front.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	fmt.Fprintf(conn, "GET /api/m/demo/ws HTTP/1.1\r\nHost: localhost\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n")

	br := bufio.NewReader(conn)
	line, err := br.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(line, "101") {
		t.Fatalf("升级失败: %q", line)
	}
	for {
		h, err := br.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if h == "\r\n" {
			break
		}
		if strings.HasPrefix(strings.ToLower(h), "x-mica-token:") && !strings.Contains(h, "sidecar-token") {
			t.Fatalf("token 未注入: %q", h)
		}
	}
	if _, err := fmt.Fprintf(conn, "ping\n"); err != nil {
		t.Fatal(err)
	}
	echo, err := br.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(echo, "echo:ping") {
		t.Fatalf("升级后数据未透传: %q", echo)
	}
}
