// Package module 负责模块清单（manifest.json）的解析、扫描与安装状态管理。
package module

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// ErrNotFound 表示模块不存在。
var ErrNotFound = errors.New("模块不存在")

var idPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,31}$`)

// Frontend 描述模块前端的入口文件（远程 ESM，必须导出 mount(el, api)）。
type Frontend struct {
	Entry string `json:"entry"`
}

// Backend 描述模块后端（P0 仅做契约占位，运行时为 wazero/wasm32-wasi）。
type Backend struct {
	Type  string `json:"type"`
	Entry string `json:"entry"`
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
	Running   bool `json:"running"`
}

type state struct {
	Installed []string `json:"installed"`
}

// Registry 管理模块目录扫描结果与已安装集合。
type Registry struct {
	mu        sync.RWMutex
	dir       string
	statePath string
	modules   map[string]*Module
	installed map[string]bool
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
		installed: make(map[string]bool),
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
		if !entry.IsDir() {
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
	if r.fresh && len(r.installed) == 0 {
		if _, ok := modules[defaultModule]; ok {
			r.installed[defaultModule] = true
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

// IsInstalled 返回模块是否已启用。
func (r *Registry) IsInstalled(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.installed[id]
}

// List 返回按 ID 排序的模块视图。
func (r *Registry) List() []View {
	r.mu.RLock()
	defer r.mu.RUnlock()

	views := make([]View, 0, len(r.modules))
	for _, m := range r.modules {
		views = append(views, View{
			Manifest:  m.Manifest,
			Installed: r.installed[m.ID],
			Running:   r.installed[m.ID],
		})
	}
	sort.Slice(views, func(i, j int) bool { return views[i].ID < views[j].ID })
	return views
}

// Install 将模块标记为已安装。
func (r *Registry) Install(id string) error {
	r.mu.Lock()
	if _, ok := r.modules[id]; !ok {
		r.mu.Unlock()
		return ErrNotFound
	}
	r.installed[id] = true
	r.mu.Unlock()
	return r.saveState()
}

// Uninstall 卸载模块（P0 仅移除启用状态）。
func (r *Registry) Uninstall(id string) error {
	r.mu.Lock()
	if _, ok := r.modules[id]; !ok {
		r.mu.Unlock()
		return ErrNotFound
	}
	delete(r.installed, id)
	r.mu.Unlock()
	return r.saveState()
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
	for _, id := range s.Installed {
		r.installed[id] = true
	}
}

func (r *Registry) saveState() error {
	r.mu.RLock()
	s := state{Installed: make([]string, 0, len(r.installed))}
	for id := range r.installed {
		s.Installed = append(s.Installed, id)
	}
	r.mu.RUnlock()
	sort.Strings(s.Installed)

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
