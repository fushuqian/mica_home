// Package module 负责模块清单（manifest.json）的解析、扫描与安装状态管理。
package module

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// ErrNotFound 表示模块不存在。
var ErrNotFound = errors.New("模块不存在")

// ErrAlreadyExists 表示同 ID 模块已存在，拒绝覆盖安装。
var ErrAlreadyExists = errors.New("模块已存在")

const (
	maxArchiveBytes      = 160 << 20
	maxFrontendBytes     = 20 << 20
	maxSidecarBytes      = 128 << 20
	maxArchiveEntryBytes = 128 << 20
	maxArchiveEntryCount = 1000
)

var idPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,31}$`)

// Frontend 描述模块前端的入口文件（远程 ESM，必须导出 mount(el, api)）。
type Frontend struct {
	Entry string `json:"entry"`
	Card  string `json:"card,omitempty"`
}

// Backend 描述模块后端（P0 仅做契约占位，运行时为 wazero/wasm32-wasi）。
type Backend struct {
	Type  string `json:"type"`
	Entry string `json:"entry"`
}

// Sidecar 描述模块附带的容器内常驻进程。
type Sidecar struct {
	Binary      string            `json:"binary"`
	Executables []string          `json:"executables"`
	Platform    string            `json:"platform"`
	SHA256      map[string]string `json:"sha256"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env"`
	HealthPath  string            `json:"healthPath"`
	SocketName  string            `json:"socketName"`
	Permissions []string          `json:"permissions"`
}

// Manifest 是模块与宿主之间唯一的契约。字段一旦定型，后续改动等同于重做模块生态。
type Manifest struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Category    string   `json:"category"`
	Icon        string   `json:"icon"`
	Tone        string   `json:"tone"`
	Permissions []string `json:"permissions"`
	Frontend    Frontend `json:"frontend"`
	Sidecar     *Sidecar `json:"sidecar,omitempty"`
	Backend     *Backend `json:"backend,omitempty"`
}

// Module 是磁盘上的一个模块实例。
type Module struct {
	Manifest
	Dir string `json:"-"`
}

// View 是暴露给前端的模块视图。
type View struct {
	Manifest
	Installed bool `json:"installed"`
	Enabled   bool `json:"enabled"`
	Running   bool `json:"running"`
}

type state struct {
	Enabled   []string `json:"enabled"`
	Installed []string `json:"installed,omitempty"` // 旧版本字段，仅用于迁移
}

// Registry 管理模块目录扫描结果与启用集合。
type Registry struct {
	mu        sync.RWMutex
	installMu sync.Mutex
	dir       string
	statePath string
	modules   map[string]*Module
	enabled   map[string]bool
	warnings  []string
	fresh     bool
}

// defaultModule 首次启动时默认启用的示例模块。
const defaultModule = "hello"

// NewRegistry 创建一个注册表。statePath 为安装状态持久化文件。
func NewRegistry(dir, statePath string) *Registry {
	r := &Registry{
		dir:       dir,
		statePath: statePath,
		modules:   make(map[string]*Module),
		enabled:   make(map[string]bool),
	}
	if _, err := os.Stat(statePath); errors.Is(err, os.ErrNotExist) {
		r.fresh = true
	}
	r.loadState()
	return r
}

// Scan 重新扫描模块目录，返回不致命的模块问题（如缺字段、入口缺失）。
func (r *Registry) Scan() []string {
	entries, err := os.ReadDir(r.dir)
	if errors.Is(err, os.ErrNotExist) {
		r.mu.Lock()
		r.modules = map[string]*Module{}
		r.warnings = nil
		r.mu.Unlock()
		return nil
	}
	if err != nil {
		return []string{"读取模块目录失败: " + err.Error()}
	}

	modules := make(map[string]*Module, len(entries))
	var warnings []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		m, err := load(filepath.Join(r.dir, entry.Name()))
		if err != nil {
			warnings = append(warnings, entry.Name()+": "+err.Error())
			continue
		}
		if _, dup := modules[m.ID]; dup {
			warnings = append(warnings, entry.Name()+": 模块 ID 重复，已忽略")
			continue
		}
		modules[m.ID] = m
	}

	r.mu.Lock()
	r.modules = modules
	r.warnings = warnings
	seed := false
	if r.fresh && len(r.enabled) == 0 {
		if _, ok := modules[defaultModule]; ok {
			r.enabled[defaultModule] = true
			r.warnings = append(r.warnings, "首次启动，已默认启用示例模块 "+defaultModule)
			seed = true
		}
	}
	r.fresh = false
	r.mu.Unlock()
	if seed {
		_ = r.saveState()
	}
	return warnings
}

// Warnings 返回最近一次扫描的问题列表。
func (r *Registry) Warnings() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.warnings))
	copy(out, r.warnings)
	return out
}

// Get 按 ID 取模块。
func (r *Registry) Get(id string) (*Module, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.modules[id]
	return m, ok
}

// IsEnabled 返回模块是否处于启用状态。
func (r *Registry) IsEnabled(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.enabled[id]
}

// List 返回按 ID 排序的模块视图。扫描到的模块都算已安装，enabled 表示是否启用。
func (r *Registry) List() []View {
	r.mu.RLock()
	defer r.mu.RUnlock()

	views := make([]View, 0, len(r.modules))
	for _, m := range r.modules {
		on := r.enabled[m.ID]
		views = append(views, View{
			Manifest:  m.Manifest,
			Installed: true,
			Enabled:   on,
			Running:   on,
		})
	}
	sort.Slice(views, func(i, j int) bool { return views[i].ID < views[j].ID })
	return views
}

// Enable 启用模块。
func (r *Registry) Enable(id string) error {
	r.mu.Lock()
	if _, ok := r.modules[id]; !ok {
		r.mu.Unlock()
		return ErrNotFound
	}
	r.enabled[id] = true
	r.mu.Unlock()
	return r.saveState()
}

// Disable 停用模块，保留模块目录。
func (r *Registry) Disable(id string) error {
	r.mu.Lock()
	if _, ok := r.modules[id]; !ok {
		r.mu.Unlock()
		return ErrNotFound
	}
	delete(r.enabled, id)
	r.mu.Unlock()
	return r.saveState()
}

// Remove 卸载模块：删除模块目录并清除启用状态。
func (r *Registry) Remove(id string) error {
	r.installMu.Lock()
	defer r.installMu.Unlock()

	m, ok := r.Get(id)
	if !ok {
		return ErrNotFound
	}
	if err := os.RemoveAll(m.Dir); err != nil {
		return fmt.Errorf("删除模块目录失败: %w", err)
	}
	if err := os.RemoveAll(r.SidecarDir(id)); err != nil {
		return fmt.Errorf("删除 sidecar 目录失败: %w", err)
	}

	r.mu.Lock()
	delete(r.modules, id)
	delete(r.enabled, id)
	r.mu.Unlock()
	return r.saveState()
}

// InstallArchive 从 zip 包安装模块并启用。包根必须含 manifest.json 与 frontend/，
// 可选 sidecar/。frontend 进入模块目录，sidecar 进入 /data/sidecar/<id>。
func (r *Registry) InstallArchive(data []byte) (*Module, error) {
	r.installMu.Lock()
	defer r.installMu.Unlock()

	if len(data) == 0 {
		return nil, errors.New("模块包为空")
	}
	if len(data) > maxArchiveBytes {
		return nil, fmt.Errorf("模块包超过 %d MB", maxArchiveBytes>>20)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("不是有效的 zip 包: %w", err)
	}

	dataRoot := r.DataRoot()
	if err := os.MkdirAll(filepath.Join(dataRoot, "tmp"), 0o755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}
	moduleTmp, err := os.MkdirTemp(r.dir, ".install-*")
	if err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}
	sideTmp, err := os.MkdirTemp(filepath.Join(dataRoot, "tmp"), "sidecar-*")
	if err != nil {
		os.RemoveAll(moduleTmp)
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer func() {
		if moduleTmp != "" {
			os.RemoveAll(moduleTmp)
		}
		if sideTmp != "" {
			os.RemoveAll(sideTmp)
		}
	}()

	if err := extractZip(zr, moduleTmp, sideTmp); err != nil {
		return nil, err
	}
	m, err := load(moduleTmp)
	if err != nil {
		return nil, err
	}
	if m.Sidecar != nil {
		if err := verifySidecar(m.Sidecar, sideTmp); err != nil {
			return nil, err
		}
	}
	if _, ok := r.Get(m.ID); ok {
		return nil, fmt.Errorf("%w: %s", ErrAlreadyExists, m.ID)
	}

	moduleTarget := filepath.Join(r.dir, m.ID)
	sideTarget := r.SidecarDir(m.ID)
	if _, err := os.Stat(moduleTarget); err == nil {
		return nil, fmt.Errorf("%w: %s", ErrAlreadyExists, m.ID)
	}
	if m.Sidecar != nil {
		if _, err := os.Stat(sideTarget); err == nil {
			return nil, fmt.Errorf("%w: sidecar %s", ErrAlreadyExists, m.ID)
		}
	}
	if err := os.Rename(moduleTmp, moduleTarget); err != nil {
		return nil, fmt.Errorf("写入模块目录失败: %w", err)
	}
	moduleTmp = ""
	if m.Sidecar != nil {
		if err := os.MkdirAll(filepath.Dir(sideTarget), 0o755); err != nil {
			os.RemoveAll(moduleTarget)
			return nil, err
		}
		if err := os.Rename(sideTmp, sideTarget); err != nil {
			os.RemoveAll(moduleTarget)
			return nil, fmt.Errorf("写入 sidecar 目录失败: %w", err)
		}
		sideTmp = ""
	}
	if err := os.MkdirAll(r.ModuleDataDir(m.ID), 0o755); err != nil {
		os.RemoveAll(moduleTarget)
		return nil, fmt.Errorf("创建模块数据目录失败: %w", err)
	}

	r.Scan()
	if err := r.Enable(m.ID); err != nil {
		return nil, err
	}
	m.Dir = moduleTarget
	return m, nil
}

// extractZip 把 zip 拆到两个目标：前端/清单进入 moduleDst，sidecar 进入 sideDst。
func extractZip(zr *zip.Reader, moduleDst, sideDst string) error {
	if len(zr.File) > maxArchiveEntryCount {
		return fmt.Errorf("模块包文件数超过 %d", maxArchiveEntryCount)
	}
	var frontendTotal, sidecarTotal int64
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if f.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("模块包不允许符号链接: %s", f.Name)
		}
		name := path.Clean(strings.ReplaceAll(f.Name, "\\", "/"))
		if name == "." || name == "" {
			continue
		}
		if strings.HasPrefix(name, "..") || strings.HasPrefix(name, "/") || strings.Contains(name, ":") {
			return fmt.Errorf("非法文件路径: %s", f.Name)
		}
		if f.UncompressedSize64 > maxArchiveEntryBytes {
			return fmt.Errorf("文件超过 %d MB: %s", maxArchiveEntryBytes>>20, f.Name)
		}

		var dstDir, rel string
		switch {
		case name == "manifest.json":
			dstDir, rel = moduleDst, "manifest.json"
		case strings.HasPrefix(name, "frontend/"):
			frontendTotal += int64(f.UncompressedSize64)
			if frontendTotal > maxFrontendBytes {
				return fmt.Errorf("前端包解压后超过 %d MB", maxFrontendBytes>>20)
			}
			dstDir, rel = moduleDst, name
		case strings.HasPrefix(name, "sidecar/"):
			sidecarTotal += int64(f.UncompressedSize64)
			if sidecarTotal > maxSidecarBytes {
				return fmt.Errorf("sidecar 解压后超过 %d MB", maxSidecarBytes>>20)
			}
			dstDir, rel = sideDst, strings.TrimPrefix(name, "sidecar/")
		default:
			return fmt.Errorf("模块包只允许 manifest.json、frontend/ 与 sidecar/：%s", f.Name)
		}
		if rel == "" {
			continue
		}
		if err := writeZipEntry(f, dstDir, rel); err != nil {
			return err
		}
	}
	return nil
}

func writeZipEntry(f *zip.File, dstDir, rel string) error {
	dest := filepath.Join(dstDir, filepath.FromSlash(rel))
	if !strings.HasPrefix(dest, dstDir+string(os.PathSeparator)) {
		return fmt.Errorf("非法文件路径: %s", f.Name)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("读取 %s 失败: %w", f.Name, err)
	}
	defer rc.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("写入 %s 失败: %w", f.Name, err)
	}
	n, copyErr := io.Copy(out, io.LimitReader(rc, maxArchiveEntryBytes+1))
	out.Close()
	if copyErr != nil {
		return fmt.Errorf("解压 %s 失败: %w", f.Name, copyErr)
	}
	if n > maxArchiveEntryBytes {
		return fmt.Errorf("文件超过 %d MB: %s", maxArchiveEntryBytes>>20, f.Name)
	}
	return nil
}

// verifySidecar 校验平台、SHA256，并给可执行文件加上执行位。
func verifySidecar(s *Sidecar, dir string) error {
	if s.Platform != runtime.GOOS+"/"+runtime.GOARCH {
		return fmt.Errorf("sidecar 平台 %s 与宿主 %s/%s 不匹配", s.Platform, runtime.GOOS, runtime.GOARCH)
	}
	binRel, err := sidecarRel(s.Binary)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, binRel)); err != nil {
		return fmt.Errorf("sidecar 入口 %s 不存在", s.Binary)
	}

	execs := s.Executables
	if len(execs) == 0 {
		execs = []string{s.Binary}
	}
	for _, rel := range execs {
		clean, err := sidecarRel(rel)
		if err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(dir, clean)); err != nil {
			return fmt.Errorf("sidecar 可执行文件 %s 不存在", rel)
		}
		if err := os.Chmod(filepath.Join(dir, clean), 0o755); err != nil {
			return err
		}
	}
	for rel, want := range s.SHA256 {
		clean, err := sidecarRel(rel)
		if err != nil {
			return err
		}
		sum, err := fileSHA256(filepath.Join(dir, clean))
		if err != nil {
			return fmt.Errorf("校验 %s 失败: %w", rel, err)
		}
		if sum != strings.ToLower(want) {
			return fmt.Errorf("sidecar 文件 %s 的 SHA256 不匹配", rel)
		}
	}
	return nil
}

func sidecarRel(rel string) (string, error) {
	clean := path.Clean(strings.ReplaceAll(rel, "\\", "/"))
	if !strings.HasPrefix(clean, "sidecar/") {
		return "", fmt.Errorf("sidecar 路径必须以 sidecar/ 开头: %s", rel)
	}
	clean = strings.TrimPrefix(clean, "sidecar/")
	if clean == "" || strings.HasPrefix(clean, "..") || strings.Contains(clean, ":") {
		return "", fmt.Errorf("非法 sidecar 路径: %s", rel)
	}
	return filepath.FromSlash(clean), nil
}

func fileSHA256(name string) (string, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// DataRoot 返回运行数据根目录（state.json 所在目录）。
func (r *Registry) DataRoot() string {
	return filepath.Dir(r.statePath)
}

// SidecarDir 返回模块 sidecar 的安装目录。
func (r *Registry) SidecarDir(id string) string {
	return filepath.Join(r.DataRoot(), "sidecar", id)
}

// ModuleDataDir 返回模块的用户数据目录。
func (r *Registry) ModuleDataDir(id string) string {
	return filepath.Join(r.DataRoot(), "modules", id)
}

// RunDir 返回 sidecar 的 socket 目录。
func (r *Registry) RunDir() string {
	return filepath.Join(r.DataRoot(), "run")
}

// LogDir 返回 sidecar 日志目录。
func (r *Registry) LogDir() string {
	return filepath.Join(r.DataRoot(), "logs")
}

func (r *Registry) loadState() {
	data, err := os.ReadFile(r.statePath)
	if err != nil {
		return
	}
	var s state
	if json.Unmarshal(data, &s) != nil {
		return
	}
	// 新字段优先；旧版本只有 installed，迁移为 enabled。
	list := s.Enabled
	if len(list) == 0 {
		list = s.Installed
	}
	for _, id := range list {
		r.enabled[id] = true
	}
}

func (r *Registry) saveState() error {
	r.mu.RLock()
	s := state{Enabled: make([]string, 0, len(r.enabled))}
	for id := range r.enabled {
		s.Enabled = append(s.Enabled, id)
	}
	r.mu.RUnlock()
	sort.Strings(s.Enabled)

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.statePath), 0o755); err != nil {
		return err
	}
	tmp := r.statePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, r.statePath)
}

func load(dir string) (*Module, error) {
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, errors.New("缺少 manifest.json")
	}
	// 兼容 Windows 编辑器保存的 UTF-8 BOM。
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest.json 解析失败: %w", err)
	}
	if !idPattern.MatchString(m.ID) {
		return nil, fmt.Errorf("id %q 非法（小写字母开头，仅含小写字母、数字与短横线）", m.ID)
	}
	if m.Name == "" {
		return nil, errors.New("缺少 name")
	}
	if m.Version == "" {
		return nil, errors.New("缺少 version")
	}
	if m.Frontend.Entry == "" {
		return nil, errors.New("缺少 frontend.entry")
	}
	mod := &Module{Manifest: m, Dir: dir}
	full, err := mod.AssetPath(m.Frontend.Entry)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(full); err != nil {
		return nil, fmt.Errorf("前端入口 %s 不存在", m.Frontend.Entry)
	}
	if m.Frontend.Card != "" {
		cardPath, err := mod.AssetPath(m.Frontend.Card)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(cardPath); err != nil {
			return nil, fmt.Errorf("卡片入口 %s 不存在", m.Frontend.Card)
		}
	}
	if m.Sidecar != nil {
		if m.Sidecar.Binary == "" {
			return nil, errors.New("sidecar 缺少 binary")
		}
		if _, err := sidecarRel(m.Sidecar.Binary); err != nil {
			return nil, err
		}
		if m.Sidecar.Platform == "" {
			m.Sidecar.Platform = runtime.GOOS + "/" + runtime.GOARCH
		}
		if m.Sidecar.HealthPath == "" {
			m.Sidecar.HealthPath = "/health"
		}
		if !strings.HasPrefix(m.Sidecar.HealthPath, "/") {
			return nil, errors.New("sidecar.healthPath 必须以 / 开头")
		}
		if m.Sidecar.SocketName != "" {
			if strings.ContainsAny(m.Sidecar.SocketName, `/\:`) || m.Sidecar.SocketName == "." || m.Sidecar.SocketName == ".." {
				return nil, errors.New("sidecar.socketName 非法")
			}
		}
		for _, rel := range m.Sidecar.Executables {
			if _, err := sidecarRel(rel); err != nil {
				return nil, err
			}
		}
		for rel := range m.Sidecar.SHA256 {
			if _, err := sidecarRel(rel); err != nil {
				return nil, err
			}
		}
	}
	return mod, nil
}

// AssetPath 把模块内的相对路径解析为绝对路径，禁止越出 frontend 目录。
func (m *Module) AssetPath(rel string) (string, error) {
	root := filepath.Join(m.Dir, "frontend")
	clean := filepath.Clean("/" + rel)
	full := filepath.Join(root, clean)
	if !strings.HasPrefix(full, root+string(os.PathSeparator)) && full != root {
		return "", errors.New("非法路径: " + rel)
	}
	return full, nil
}
