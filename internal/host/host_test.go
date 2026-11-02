package host

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

// TestStatusWriterKeepsHijack 保证日志中间件不会挡住 WebSocket 升级：
// statusWriter 必须通过 Unwrap 把底层的 Hijacker 暴露给 http.ResponseController。
func TestStatusWriterKeepsHijack(t *testing.T) {
	hijackErr := make(chan error, 1)
	srv := httptest.NewServer(logRequests(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _, err := http.NewResponseController(w).Hijack()
		if err != nil {
			hijackErr <- err
			return
		}
		defer conn.Close()
		io.WriteString(conn, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
	})))
	defer srv.Close()

	conn, err := net.Dial("tcp", strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "GET /ws HTTP/1.1\r\nHost: localhost\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n")
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-hijackErr:
		t.Fatalf("Hijack 失败（中间件吞掉了 Unwrap？）: %v", err)
	default:
	}
	if !strings.Contains(line, "101") {
		t.Fatalf("升级响应不是 101: %q", line)
	}
}
