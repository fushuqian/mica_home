// Package sidecar 管理模块附带的容器内常驻进程（watch）。
package sidecar

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fushuqian/mica_home/internal/module"
)

// Status 是宿主暴露的 sidecar 运行状态。
type Status struct {
	State     string `json:"state"`
	PID       int    `json:"pid"`
	StartedAt string `json:"startedAt,omitempty"`
	Restarts  int    `json:"restarts"`
	ExitCode  int    `json:"exitCode"`
	LastError string `json:"lastError"`
	Addr      string `json:"-"`
}

type process struct {
	mod       *module.Module
	sidecarID string
	dataRoot  string
	binPath   string
	workDir   string
	sidecarDir string
	dataDir    string
	logPath    string
	addr       string
	token      string
	healthPath string

	mu        sync.Mutex
	cmd       *exec.Cmd
	state     string
	pid       int
	startedAt time.Time
	restarts  int
	exitCode  int
	lastError string
	stopping  bool
	exited    chan struct{}
	ready     chan struct{}
	readyOnce sync.Once
}

// Manager 管理所有模块的 sidecar 进程。
type Manager struct {
	mu      sync.Mutex
	procs   map[string]*process
	dataRoot string
}

// New 创建 sidecar 管理器。
func New(dataRoot string) *Manager {
	return &Manager{procs: make(map[string]*process), dataRoot: dataRoot}
}

// Start 启动模块的 sidecar；未声明 sidecar 的模块直接返回。
func (m *Manager) Start(mod *module.Module) error {
	if mod == nil || mod.Sidecar == nil {
		return nil
	}
	m.mu.Lock()
	old := m.procs[mod.ID]
	if old != nil && old.isActive() {
		m.mu.Unlock()
		return nil
	}
	p, err := newProcess(m.dataRoot, mod)
	if err != nil {
		m.mu.Unlock()
		return err
	}
	m.procs[mod.ID] = p
	m.mu.Unlock()

	if old != nil {
		old.stop()
	}
	return p.start()
}

// Stop 停止模块 sidecar，并清理 socket。
func (m *Manager) Stop(id string) {
	m.mu.Lock()
	p := m.procs[id]
	m.mu.Unlock()
	if p != nil {
		p.stop()
	}
}

// Status 返回 sidecar 状态；不存在时返回 stopped。
func (m *Manager) Status(id string) Status {
	m.mu.Lock()
	p := m.procs[id]
	m.mu.Unlock()
	if p == nil {
		return Status{State: "stopped"}
	}
	return p.status()
}

// Endpoint 返回代理所需的地址与 token。
func (m *Manager) Endpoint(id string) (addr, token string, path string, ok bool) {
	m.mu.Lock()
	p := m.procs[id]
	m.mu.Unlock()
	if p == nil || !p.isActive() {
		return "", "", "", false
	}
	return p.addr, p.token, p.healthPath, true
}

// Logs 返回日志尾部。
func (m *Manager) Logs(id string, tail int) string {
	m.mu.Lock()
	p := m.procs[id]
	m.mu.Unlock()
	if p == nil {
		return ""
	}
	return tailFile(p.logPath, tail)
}

// Shutdown 停止全部 sidecar。
func (m *Manager) Shutdown() {
	m.mu.Lock()
	list := make([]*process, 0, len(m.procs))
	for _, p := range m.procs {
		list = append(list, p)
	}
	m.mu.Unlock()
	for _, p := range list {
		p.stop()
	}
}

func newProcess(dataRoot string, mod *module.Module) (*process, error) {
	sc := mod.Sidecar
	binRel, err := sidecarRel(sc.Binary)
	if err != nil {
		return nil, err
	}
	sidecarDir := filepath.Join(dataRoot, "sidecar", mod.ID)
	dataDir := filepath.Join(dataRoot, "modules", mod.ID)
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(dataRoot, "logs"), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(dataRoot, "run"), 0o755); err != nil {
		return nil, err
	}

	addr := ""
	if sc.SocketName != "" {
		addr = "unix://" + filepath.Join(dataRoot, "run", sc.SocketName)
	} else {
		port, err := freePort()
		if err != nil {
			return nil, err
		}
		addr = "tcp://127.0.0.1:" + strconv.Itoa(port)
	}
	health := sc.HealthPath
	if health == "" {
		health = "/health"
	}
	return &process{
		mod:        mod,
		sidecarID:  mod.ID,
		dataRoot:   dataRoot,
		binPath:    filepath.Join(sidecarDir, binRel),
		workDir:    dataDir,
		sidecarDir: sidecarDir,
		dataDir:    dataDir,
		logPath:    filepath.Join(dataRoot, "logs", mod.ID+".log"),
		addr:       addr,
		token:      randomToken(),
		healthPath: health,
		state:      "stopped",
		exited:     make(chan struct{}),
		ready:      make(chan struct{}),
	}, nil
}

func (p *process) isActive() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state == "running" || p.state == "starting" || p.state == "degraded"
}

func (p *process) status() Status {
	p.mu.Lock()
	defer p.mu.Unlock()
	st := ""
	if !p.startedAt.IsZero() {
		st = p.startedAt.Format(time.RFC3339)
	}
	return Status{
		State: p.state, PID: p.pid, StartedAt: st,
		Restarts: p.restarts, ExitCode: p.exitCode,
		LastError: p.lastError, Addr: p.addr,
	}
}

func (p *process) start() error {
	p.mu.Lock()
	if p.state == "running" || p.state == "starting" {
		p.mu.Unlock()
		return nil
	}
	p.stopping = false
	p.lastError = ""
	p.state = "starting"
	env := p.env()
	args := p.args(env)
	logFile, err := os.OpenFile(p.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		p.state = "degraded"
		p.lastError = err.Error()
		p.mu.Unlock()
		return err
	}
	cmd := exec.Command(p.binPath, args...)
	cmd.Dir = p.workDir
	cmd.Env = envList(env)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = sysProcAttr()
	p.cmd = cmd
	exited := make(chan struct{})
	p.exited = exited
	p.mu.Unlock()

	if err := cmd.Start(); err != nil {
		logFile.Close()
		p.mu.Lock()
		p.state = "degraded"
		p.lastError = "启动失败: " + err.Error()
		p.mu.Unlock()
		return err
	}
	p.mu.Lock()
	p.pid = cmd.Process.Pid
	p.startedAt = time.Now()
	p.mu.Unlock()

	go func() {
		err := cmd.Wait()
		logFile.Close()
		close(exited)
		p.mu.Lock()
		if p.stopping {
			p.state = "stopped"
			p.mu.Unlock()
			return
		}
		p.state = "degraded"
		p.restarts++
		if err != nil {
			p.lastError = "进程退出: " + err.Error()
			if ee, ok := err.(*exec.ExitError); ok {
				p.exitCode = ee.ExitCode()
			}
		} else {
			p.lastError = "进程已退出"
		}
		p.mu.Unlock()
	}()

	go p.checkHealth()
	return nil
}

func (p *process) checkHealth() {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if p.isStopping() {
			return
		}
		code, err := p.get(p.healthPath)
		if err == nil && code == http.StatusOK {
			p.mu.Lock()
			p.state = "running"
			p.lastError = ""
			p.mu.Unlock()
			p.readyOnce.Do(func() { close(p.ready) })
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	p.mu.Lock()
	if p.state == "starting" {
		p.state = "degraded"
		p.lastError = "健康检查超时"
	}
	p.mu.Unlock()
}

func (p *process) stop() {
	p.mu.Lock()
	if p.state == "stopped" || p.cmd == nil || p.cmd.Process == nil {
		p.state = "stopped"
		p.mu.Unlock()
		return
	}
	p.stopping = true
	p.state = "stopping"
	cmd := p.cmd
	exited := p.exited
	p.mu.Unlock()

	killGroup(cmd, false)
	select {
	case <-exited:
	case <-time.After(10 * time.Second):
		killGroup(cmd, true)
		select {
		case <-exited:
		case <-time.After(3 * time.Second):
		}
	}
	if strings.HasPrefix(p.addr, "unix://") {
		os.Remove(strings.TrimPrefix(p.addr, "unix://"))
	}
	p.mu.Lock()
	p.state = "stopped"
	p.pid = 0
	p.mu.Unlock()
}

func (p *process) isStopping() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stopping
}

func (p *process) env() map[string]string {
	env := map[string]string{
		"MICA_DATA_DIR":         p.dataRoot,
		"MICA_MODULE_ID":        p.sidecarID,
		"MICA_MODULE_DIR":       p.mod.Dir,
		"MICA_MODULE_DATA_DIR":  p.dataDir,
		"MICA_SIDECAR_DIR":      p.sidecarDir,
		"MICA_SIDECAR_ADDR":     p.addr,
		"MICA_SIDECAR_TOKEN":    p.token,
		"MICA_HOST_TOKEN":       p.token,
	}
	for k, v := range p.mod.Sidecar.Env {
		env[k] = v
	}
	return env
}

func (p *process) args(env map[string]string) []string {
	out := make([]string, 0, len(p.mod.Sidecar.Args))
	for _, a := range p.mod.Sidecar.Args {
		for k, v := range env {
			a = strings.ReplaceAll(a, "${"+k+"}", v)
		}
		out = append(out, a)
	}
	return out
}

func envList(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}

func (p *process) get(path string) (int, error) {
	client, base, err := p.client()
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequest(http.MethodGet, base+path, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("X-Mica-Token", p.token)
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

// Client 返回访问该 sidecar 的 HTTP 客户端与基地址。
func (m *Manager) Client(id string) (*http.Client, string, string, bool) {
	m.mu.Lock()
	p := m.procs[id]
	m.mu.Unlock()
	if p == nil || !p.isActive() {
		return nil, "", "", false
	}
	client, base, err := p.client()
	if err != nil {
		return nil, "", "", false
	}
	return client, base, p.token, true
}

func (p *process) client() (*http.Client, string, error) {
	if strings.HasPrefix(p.addr, "unix://") {
		sock := strings.TrimPrefix(p.addr, "unix://")
		tr := &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", sock)
			},
		}
		return &http.Client{Transport: tr, Timeout: 30 * time.Second}, "http://sidecar", nil
	}
	host := strings.TrimPrefix(p.addr, "tcp://")
	return &http.Client{Timeout: 30 * time.Second}, "http://" + host, nil
}

func freePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func randomToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}

func sidecarRel(rel string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(strings.ReplaceAll(rel, "\\", "/")))
	if !strings.HasPrefix(clean, "sidecar/") {
		return "", fmt.Errorf("sidecar 路径必须以 sidecar/ 开头: %s", rel)
	}
	clean = strings.TrimPrefix(clean, "sidecar/")
	if clean == "" || strings.HasPrefix(clean, "..") || strings.Contains(clean, ":") {
		return "", fmt.Errorf("非法 sidecar 路径: %s", rel)
	}
	return clean, nil
}

func tailFile(name string, n int) string {
	f, err := os.Open(name)
	if err != nil {
		return ""
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return ""
	}
	const maxRead = 64 * 1024
	if st.Size() > maxRead {
		f.Seek(st.Size()-maxRead, io.SeekStart)
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if n > 0 && len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
