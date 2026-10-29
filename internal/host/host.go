// Package host 提供宿主 HTTP 服务：静态外壳、模块 API 与模块前端资源挂载。
package host

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fushuqian/mica_home/internal/module"
	"github.com/fushuqian/mica_home/internal/sidecar"
)

// Options 是宿主服务的构造参数。
type Options struct {
	Registry *module.Registry
	Assets   fs.FS
	Version  string
	Sidecars *sidecar.Manager
}

// Server 是宿主 HTTP 服务。
type Server struct {
	reg      *module.Registry
	assets   fs.FS
	version  string
	started  time.Time
	sidecars *sidecar.Manager
}

// New 创建宿主服务。
func New(o Options) *Server {
	return &Server{reg: o.Registry, assets: o.Assets, version: o.Version, started: time.Now(), sidecars: o.Sidecars}
}

// Handler 返回路由表。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/system", s.systemInfo)
	mux.HandleFunc("GET /api/modules", s.listModules)
	mux.HandleFunc("POST /api/modules/install", s.installArchive)
	mux.HandleFunc("POST /api/modules/{id}/enable", s.enable)
	mux.HandleFunc("POST /api/modules/{id}/disable", s.disable)
	mux.HandleFunc("POST /api/modules/{id}/confirm", s.confirm)
	mux.HandleFunc("DELETE /api/modules/{id}", s.removeModule)
	mux.HandleFunc("POST /api/modules/{id}/install", s.enable)    // 兼容旧调用：安装即启用
	mux.HandleFunc("POST /api/modules/{id}/uninstall", s.removeModule) // 兼容旧调用：卸载即删除
	mux.HandleFunc("GET /docs", s.docsIndex)
	mux.HandleFunc("GET /docs/modules", s.htmlPage("docs-modules.html"))
	mux.HandleFunc("GET /docs/theme", s.htmlPage("docs-theme.html"))
	mux.HandleFunc("GET /docs/agent", s.agentGuide)
	mux.HandleFunc("GET /docs/agent.md", s.agentGuide)
	mux.HandleFunc("GET /manage", s.htmlPage("manage.html"))
	mux.HandleFunc("GET /status", s.htmlPage("status.html"))
	mux.HandleFunc("/api/m/", s.moduleRoute)
	mux.Handle("/", http.FileServer(http.FS(s.assets)))
	return logRequests(mux)
}

// moduleRoute 统一处理 /api/m/<id>/*：info、ui、__status、__logs 与 sidecar 反代。
func (s *Server) moduleRoute(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/m/")
	parts := strings.SplitN(rest, "/", 2)
	id := parts[0]
	sub := ""
	if len(parts) > 1 {
		sub = parts[1]
	}
	if id == "" {
		writeError(w, http.StatusNotFound, "模块不存在")
		return
	}
	r.SetPathValue("id", id)

	switch {
	case sub == "info" && r.Method == http.MethodGet:
		s.moduleInfo(w, r)
	case strings.HasPrefix(sub, "ui/") && r.Method == http.MethodGet:
		r.SetPathValue("path", strings.TrimPrefix(sub, "ui/"))
		s.moduleAsset(w, r)
	case sub == "__status" && r.Method == http.MethodGet:
		s.sidecarStatus(w, r)
	case sub == "__logs" && r.Method == http.MethodGet:
		s.sidecarLogs(w, r)
	case strings.HasPrefix(sub, "__host"):
		writeError(w, http.StatusForbidden, "管理通道不对外开放")
	default:
		r.SetPathValue("path", sub)
		s.sidecarProxy(w, r)
	}
}

// docsIndex 把旧地址 /docs 跳到模块开发册。
func (s *Server) docsIndex(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/docs/modules", http.StatusFound)
}

// agentGuide 返回面向 AI Agent 的单文件 Markdown 说明。
func (s *Server) agentGuide(w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(s.assets, "agent-guide.md")
	if err != nil {
		http.Error(w, "文档未找到", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Write(data)
}

// htmlPage 返回内嵌的 HTML 页面。
func (s *Server) htmlPage(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(s.assets, name)
		if err != nil {
			http.Error(w, "文档未找到", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(data)
	}
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	list := s.reg.List()
	enabled := 0
	for _, m := range list {
		if m.Enabled {
			enabled++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"version":   s.version,
		"go":        runtime.Version(),
		"modules":   len(list),
		"installed": len(list),
		"enabled":   enabled,
		"uptimeSec": int(time.Since(s.started).Seconds()),
		"warnings":  s.reg.Warnings(),
	})
}

// moduleView 是列表项：模块信息 + sidecar 运行状态 + 权限确认状态。
type moduleView struct {
	module.View
	SidecarState string `json:"sidecarState,omitempty"`
	SidecarPID   int    `json:"sidecarPid,omitempty"`
	SidecarError string `json:"sidecarError,omitempty"`
	NeedConfirm  bool   `json:"needConfirm"`
}

func (s *Server) listModules(w http.ResponseWriter, r *http.Request) {
	list := s.reg.List()
	out := make([]moduleView, 0, len(list))
	for _, v := range list {
		mv := moduleView{View: v, NeedConfirm: s.reg.RequireConfirm(v.ID) != ""}
		if v.Sidecar != nil && s.sidecars != nil {
			st := s.sidecars.Status(v.ID)
			mv.SidecarState = st.State
			mv.SidecarPID = st.PID
			mv.SidecarError = st.LastError
		}
		out = append(out, mv)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) enable(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	m, ok := s.reg.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, module.ErrNotFound.Error())
		return
	}
	if fp := s.reg.RequireConfirm(id); fp != "" {
		writeJSON(w, http.StatusPreconditionRequired, map[string]any{
			"error":       "需要确认 sidecar 权限",
			"needConfirm": true,
			"permissions": m.Sidecar.Permissions,
			"fingerprint": fp,
		})
		return
	}
	if err := s.reg.Enable(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var sidecarErr string
	if m.Sidecar != nil && s.sidecars != nil {
		if err := s.sidecars.Start(m); err != nil {
			sidecarErr = err.Error()
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "enabled": true, "sidecarError": sidecarErr})
}

// confirm 记录 sidecar 权限确认。
func (s *Server) confirm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	fp, err := s.reg.Confirm(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "fingerprint": fp})
}

func (s *Server) disable(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if s.sidecars != nil {
		s.sidecars.Stop(id)
	}
	if err := s.reg.Disable(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "enabled": false})
}

func (s *Server) removeModule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if s.sidecars != nil {
		s.sidecars.Stop(id)
		s.sidecars.Forget(id)
	}
	if err := s.reg.Remove(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "installed": false})
}

// StartSidecars 在宿主启动时为所有已启用且声明 sidecar 的模块拉起进程。
func (s *Server) StartSidecars() {
	if s.sidecars == nil {
		return
	}
	for _, v := range s.reg.List() {
		if !v.Enabled || v.Sidecar == nil {
			continue
		}
		if s.reg.RequireConfirm(v.ID) != "" {
			log.Printf("sidecar %s 权限尚未确认，跳过启动", v.ID)
			continue
		}
		if m, ok := s.reg.Get(v.ID); ok {
			if err := s.sidecars.Start(m); err != nil {
				log.Printf("启动 sidecar %s 失败: %v", v.ID, err)
			}
		}
	}
}

// ShutdownSidecars 停止全部 sidecar，供宿主退出时调用。
func (s *Server) ShutdownSidecars() {
	if s.sidecars != nil {
		s.sidecars.Shutdown()
	}
}

// maxUploadBytes 是模块包上传上限。
const maxUploadBytes = 256 << 20

// systemInfo 返回安装限额与数据盘容量，供安装页展示。
func (s *Server) systemInfo(w http.ResponseWriter, r *http.Request) {
	limits := module.Limits()
	limits["uploadBytes"] = maxUploadBytes
	free, total := diskUsage(s.reg.DataRoot())
	writable, reason := checkDataWritable(s.reg.DataRoot())
	writeJSON(w, http.StatusOK, map[string]any{
		"limits": limits,
		"data": map[string]any{
			"freeBytes":  free,
			"totalBytes": total,
			"writable":   writable,
			"error":      reason,
		},
	})
}

// checkDataWritable 探测数据目录是否可创建临时文件，提前暴露 bind mount 权限问题。
func checkDataWritable(root string) (bool, string) {
	tmpDir := filepath.Join(root, "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return false, err.Error()
	}
	f, err := os.CreateTemp(tmpDir, "probe-*")
	if err != nil {
		return false, err.Error()
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true, ""
}

// installArchive 接收 multipart 字段 file（zip），流式落盘后安装。
// 声明的 sidecar 权限未确认时只安装不启用，由管理页确认后再启用。
func (s *Server) installArchive(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	mr, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "需要 multipart/form-data 上传")
		return
	}
	tmpDir := filepath.Join(s.reg.DataRoot(), "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		log.Printf("安装失败（临时目录不可写）: %v", err)
		writeError(w, http.StatusInternalServerError, "创建临时目录失败: "+err.Error())
		return
	}
	var tmp *os.File
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, "读取上传内容失败: "+err.Error())
			return
		}
		if part.FormName() != "file" {
			io.Copy(io.Discard, part)
			part.Close()
			continue
		}
		tmp, err = os.CreateTemp(tmpDir, "upload-*.zip")
		if err != nil {
			part.Close()
			log.Printf("安装失败（无法创建临时文件）: %v", err)
			writeError(w, http.StatusInternalServerError, "创建临时文件失败: "+err.Error())
			return
		}
		n, copyErr := io.Copy(tmp, io.LimitReader(part, maxUploadBytes+1))
		part.Close()
		if copyErr != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			writeError(w, http.StatusBadRequest, "读取上传内容失败: "+copyErr.Error())
			return
		}
		if n > maxUploadBytes {
			tmp.Close()
			os.Remove(tmp.Name())
			writeError(w, http.StatusBadRequest, "模块包超过 160 MB")
			return
		}
		break
	}
	if tmp == nil {
		writeError(w, http.StatusBadRequest, "缺少 file 字段")
		return
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	m, upgraded, err := s.reg.InstallArchiveFile(tmp.Name())
	if err != nil {
		log.Printf("安装失败: %v", err)
		status := http.StatusBadRequest
		if errors.Is(err, module.ErrAlreadyExists) {
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return
	}

	// 升级时旧 sidecar 可能仍在运行，先停掉再按新版本启动。
	if upgraded && s.sidecars != nil {
		s.sidecars.Stop(m.ID)
	}

	var permissions []string
	if m.Sidecar != nil {
		permissions = m.Sidecar.Permissions
	}
	fingerprint := s.reg.RequireConfirm(m.ID)
	resp := map[string]any{
		"id":           m.ID,
		"name":         m.Name,
		"version":      m.Version,
		"upgraded":     upgraded,
		"enabled":      s.reg.IsEnabled(m.ID),
		"needConfirm":  fingerprint != "",
		"permissions":  permissions,
		"fingerprint":  fingerprint,
		"sidecarError": "",
	}
	if fingerprint != "" {
		writeJSON(w, http.StatusOK, resp)
		return
	}

	if err := s.reg.Enable(m.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp["enabled"] = true
	if m.Sidecar != nil && s.sidecars != nil {
		if err := s.sidecars.Start(m); err != nil {
			resp["sidecarError"] = err.Error()
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) moduleInfo(w http.ResponseWriter, r *http.Request) {
	m, ok := s.reg.Get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, module.ErrNotFound.Error())
		return
	}
	writeJSON(w, http.StatusOK, m.Manifest)
}

// sidecarStatus 返回 sidecar 进程状态。
func (s *Server) sidecarStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if s.sidecars == nil {
		writeJSON(w, http.StatusOK, map[string]string{"state": "stopped"})
		return
	}
	writeJSON(w, http.StatusOK, s.sidecars.Status(id))
}

// sidecarLogs 返回 sidecar 日志尾部。
func (s *Server) sidecarLogs(w http.ResponseWriter, r *http.Request) {
	tail := 200
	if v := r.URL.Query().Get("tail"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 2000 {
			tail = n
		}
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if s.sidecars != nil {
		io.WriteString(w, s.sidecars.Logs(r.PathValue("id"), tail))
	}
}

// sidecarProxy 把 /api/m/<id>/* 转发给模块 sidecar。
// 使用 ReverseProxy，支持 WebSocket 升级、流式响应与 Range。
func (s *Server) sidecarProxy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.reg.IsEnabled(id) {
		writeError(w, http.StatusForbidden, "模块未启用")
		return
	}
	if strings.HasPrefix(r.PathValue("path"), "__host") {
		writeError(w, http.StatusForbidden, "管理通道不对外开放")
		return
	}
	if s.sidecars == nil {
		writeError(w, http.StatusNotFound, "sidecar 未运行")
		return
	}
	proxy, ok := s.sidecars.Proxy(id, "/api/m/"+id)
	if !ok {
		writeError(w, http.StatusNotFound, "sidecar 未运行")
		return
	}
	proxy.ServeHTTP(w, r)
}

// moduleAsset 挂载模块前端：/api/m/<id>/ui/<相对路径>。
func (s *Server) moduleAsset(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	m, ok := s.reg.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, module.ErrNotFound.Error())
		return
	}
	if !s.reg.IsEnabled(id) {
		writeError(w, http.StatusForbidden, "模块未启用")
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
