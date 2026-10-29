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

const (
	maxLogBytes = 5 << 20
	logBackups  = 2
	stableRun   = 60 * time.Second
	maxRestarts = 5
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
	mod        *module.Module
	sidecarID  string
	dataRoot   string
	binPath    string
	workDir    string
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
	stopCh    chan struct{}
	done      chan struct{}
}

// Manager 管理所有模块的 sidecar 进程。
type Manager struct {
	mu       sync.Mutex
	procs    map[string]*process
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
	if old != nil && old.supervising() {
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

// Forget 从管理表中移除已卸载的模块，避免残留状态。
func (m *Manager) Forget(id string) {
	m.mu.Lock()
	delete(m.procs, id)
	m.mu.Unlock()
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
	}, nil
}

func (p *process) isActive() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state == "running" || p.state == "starting" || p.state == "degraded"
}

// supervising 表示自动重启循环仍在运行。
func (p *process) supervising() bool {
	p.mu.Lock()
	done := p.done
	p.mu.Unlock()
	if done == nil {
		return false
	}
	select {
	case <-done:
		return false
	default:
		return true
	}
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
	if p.done != nil {
		select {
		case <-p.done:
		default:
			p.mu.Unlock()
			return nil
		}
	}
	if _, err := os.Stat(p.binPath); err != nil {
		p.state = "degraded"
		p.lastError = "sidecar 入口不存在: " + err.Error()
		p.mu.Unlock()
		return err
	}
	p.stopping = false
	p.lastError = ""
	p.state = "starting"
	p.stopCh = make(chan struct{})
	p.done = make(chan struct{})
	p.mu.Unlock()

	go p.loop()
	return nil
}

// loop 负责拉起进程，并在崩溃后按 1/2/5/10/10 秒退避重启。
func (p *process) loop() {
	backoff := []time.Duration{time.Second, 2 * time.Second, 5 * time.Second, 10 * time.Second, 10 * time.Second}
	attempt := 0
	defer close(p.done)
	for {
		began := time.Now()
		code, err := p.spawn()

		p.mu.Lock()
		stopping := p.stopping
		if !stopping {
			msg := "进程已退出"
			if err != nil {
				msg = "进程退出: " + err.Error()
			}
			p.pid = 0
			p.exitCode = code
			p.restarts++
			p.state = "degraded"
			p.lastError = msg
		}
		p.mu.Unlock()
		if stopping {
			return
		}
		reapZombies()
		// watch 崩溃后它拉起的子进程可能稍后才变成僵尸，延迟再收一次。
		time.AfterFunc(500*time.Millisecond, reapZombies)

		if time.Since(began) >= stableRun {
			attempt = 0
		}
		if attempt >= maxRestarts {
			p.mu.Lock()
			p.lastError = "连续异常退出，已停止自动重启；最后一次: " + p.lastError
			p.mu.Unlock()
			return
		}
		select {
		case <-p.stopCh:
			return
		case <-time.After(backoff[attempt]):
		}
		attempt++
	}
}

// spawn 启动一次进程并阻塞到它退出，返回退出码。
func (p *process) spawn() (int, error) {
	logFile, err := p.openLog()
	if err != nil {
		return 0, err
	}
	env := p.env()
	cmd := exec.Command(p.binPath, p.args(env)...)
	cmd.Dir = p.workDir
	cmd.Env = envList(env)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = sysProcAttr()

	p.mu.Lock()
	p.cmd = cmd
	p.mu.Unlock()

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return 0, fmt.Errorf("启动失败: %w", err)
	}

	p.mu.Lock()
	pid := cmd.Process.Pid
	p.pid = pid
	p.startedAt = time.Now()
	p.state = "starting"
	p.mu.Unlock()

	go p.checkHealth(pid)

	err = cmd.Wait()
	logFile.Close()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
	}
	return code, err
}

func (p *process) checkHealth(pid int) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if p.isStopping() {
			return
		}
		p.mu.Lock()
		alive := p.pid == pid
		p.mu.Unlock()
		if !alive {
			return
		}
		code, err := p.get(p.healthPath)
		if err == nil && code == http.StatusOK {
			p.mu.Lock()
			if p.pid == pid && p.state == "starting" {
				p.state = "running"
				p.lastError = ""
			}
			p.mu.Unlock()
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	p.mu.Lock()
	if p.pid == pid && p.state == "starting" {
		p.state = "degraded"
		p.lastError = "健康检查超时"
	}
	p.mu.Unlock()
}

func (p *process) stop() {
	p.mu.Lock()
	done := p.done
	stopCh := p.stopCh
	cmd := p.cmd
	if done == nil {
		p.state = "stopped"
		p.pid = 0
		p.mu.Unlock()
		return
	}
	select {
	case <-done:
		p.state = "stopped"
		p.pid = 0
		p.mu.Unlock()
		p.removeSocket()
		return
	default:
	}
	p.stopping = true
	p.state = "stopping"
	p.mu.Unlock()

	if stopCh != nil {
		select {
		case <-stopCh:
		default:
			close(stopCh)
		}
	}
	if cmd != nil && cmd.Process != nil {
		killGroup(cmd, false)
	}
	select {
	case <-done:
	case <-time.After(12 * time.Second):
		if cmd != nil && cmd.Process != nil {
			killGroup(cmd, true)
		}
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	}
	p.mu.Lock()
	p.state = "stopped"
	p.pid = 0
	p.stopping = false
	p.mu.Unlock()
	p.removeSocket()
	reapZombies()
}

func (p *process) removeSocket() {
	if strings.HasPrefix(p.addr, "unix://") {
		os.Remove(strings.TrimPrefix(p.addr, "unix://"))
	}
}

func (p *process) isStopping() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stopping
}

// openLog 打开日志文件，超过 5MB 时滚动为 .1 / .2。
func (p *process) openLog() (*os.File, error) {
	if st, err := os.Stat(p.logPath); err == nil && st.Size() >= maxLogBytes {
		os.Remove(fmt.Sprintf("%s.%d", p.logPath, logBackups))
		for i := logBackups - 1; i >= 1; i-- {
			os.Rename(fmt.Sprintf("%s.%d", p.logPath, i), fmt.Sprintf("%s.%d", p.logPath, i+1))
		}
		os.Rename(p.logPath, p.logPath+".1")
	}
	return os.OpenFile(p.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
}

func (p *process) env() map[string]string {
	env := map[string]string{
		"MICA_DATA_DIR":        p.dataRoot,
		"MICA_MODULE_ID":       p.sidecarID,
		"MICA_MODULE_DIR":      p.mod.Dir,
		"MICA_MODULE_DATA_DIR": p.dataDir,
		"MICA_SIDECAR_DIR":     p.sidecarDir,
		"MICA_SIDECAR_ADDR":    p.addr,
		"MICA_SIDECAR_TOKEN":   p.token,
		"MICA_HOST_TOKEN":      p.token,
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
