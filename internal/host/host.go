// Package host 提供宿主 HTTP 服务：静态外壳、模块 API 与模块前端资源挂载。
package host

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fushuqian/mica_home/internal/module"
)

// Options 是宿主服务的构造参数。
type Options struct {
	Registry *module.Registry
	Assets   fs.FS
	Version  string
}

// Server 是宿主 HTTP 服务。
type Server struct {
	reg     *module.Registry
	assets  fs.FS
	version string
	started time.Time
}

// New 创建宿主服务。
func New(o Options) *Server {
	return &Server{reg: o.Registry, assets: o.Assets, version: o.Version, started: time.Now()}
}

// Handler 返回路由表。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/modules", s.listModules)
	mux.HandleFunc("POST /api/modules/{id}/install", s.install)
	mux.HandleFunc("POST /api/modules/{id}/uninstall", s.uninstall)
	mux.HandleFunc("GET /api/m/{id}/info", s.moduleInfo)
	mux.HandleFunc("GET /api/m/{id}/ui/{path...}", s.moduleAsset)
	mux.Handle("GET /", http.FileServer(http.FS(s.assets)))
	return logRequests(mux)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	list := s.reg.List()
	installed := 0
	for _, m := range list {
		if m.Installed {
			installed++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "ok",
		"version":    s.version,
		"go":         runtime.Version(),
		"modules":    len(list),
		"installed":  installed,
		"uptimeSec":  int(time.Since(s.started).Seconds()),
		"warnings":   s.reg.Warnings(),
	})
}

func (s *Server) listModules(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.reg.List())
}

func (s *Server) install(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.reg.Install(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "installed": true})
}

func (s *Server) uninstall(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.reg.Uninstall(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "installed": false})
}

func (s *Server) moduleInfo(w http.ResponseWriter, r *http.Request) {
	m, ok := s.reg.Get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, module.ErrNotFound.Error())
		return
	}
	writeJSON(w, http.StatusOK, m.Manifest)
}

// moduleAsset 挂载模块前端：/api/m/<id>/ui/<相对路径>。
func (s *Server) moduleAsset(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	m, ok := s.reg.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, module.ErrNotFound.Error())
		return
	}
	if !s.reg.IsInstalled(id) {
		writeError(w, http.StatusForbidden, "模块未安装")
		return
	}

	rel := r.PathValue("path")
	if rel == "" {
		rel = m.Frontend.Entry
	}
	full, err := m.AssetPath(rel)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	f, err := os.Open(full)
	if err != nil {
		writeError(w, http.StatusNotFound, "文件不存在")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		writeError(w, http.StatusNotFound, "文件不存在")
		return
	}

	w.Header().Set("Content-Type", contentType(full))
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeContent(w, r, filepath.Base(full), info.ModTime(), f)
}

func contentType(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".wasm":
		return "application/wasm"
	default:
		return "application/octet-stream"
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("写入响应失败: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Millisecond))
	})
}
