package module

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func makeTestZip(t *testing.T, path, id, version string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	add := func(name, body string) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, body); err != nil {
			t.Fatal(err)
		}
	}
	add("manifest.json", fmt.Sprintf(`{"id":%q,"name":"Demo","version":%q,"frontend":{"entry":"index.js"}}`, id, version))
	add("frontend/index.js", "export function mount(){}")
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

func newTestRegistry(t *testing.T) (*Registry, string) {
	t.Helper()
	root := t.TempDir()
	mods := filepath.Join(root, "app", "modules")
	if err := os.MkdirAll(mods, 0o755); err != nil {
		t.Fatal(err)
	}
	return NewRegistry(mods, filepath.Join(root, "data", "state.json")), root
}

func TestInstallUpgradeAndSameVersion(t *testing.T) {
	r, root := newTestRegistry(t)
	zip1 := filepath.Join(root, "v1.zip")
	zip2 := filepath.Join(root, "v2.zip")
	makeTestZip(t, zip1, "demo", "1.0.0")
	makeTestZip(t, zip2, "demo", "2.0.0")

	if _, upgraded, err := r.InstallArchiveFile(zip1); err != nil || upgraded {
		t.Fatalf("首次安装失败: upgraded=%v err=%v", upgraded, err)
	}
	m, upgraded, err := r.InstallArchiveFile(zip2)
	if err != nil || !upgraded {
		t.Fatalf("升级失败: upgraded=%v err=%v", upgraded, err)
	}
	if m.Version != "2.0.0" {
		t.Fatalf("升级后版本 = %s", m.Version)
	}
	if got, _ := r.Get("demo"); got == nil || got.Version != "2.0.0" {
		t.Fatal("升级后注册表未加载新版本")
	}
	if _, _, err := r.InstallArchiveFile(zip2); err == nil {
		t.Fatal("同版本重装应返回错误")
	}
}

func TestUpgradeRollback(t *testing.T) {
	r, root := newTestRegistry(t)
	zip1 := filepath.Join(root, "v1.zip")
	zip2 := filepath.Join(root, "v2.zip")
	makeTestZip(t, zip1, "demo", "1.0.0")
	makeTestZip(t, zip2, "demo", "2.0.0")
	if _, _, err := r.InstallArchiveFile(zip1); err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.InstallArchiveFile(zip2); err != nil {
		t.Fatal(err)
	}

	// 用不存在的临时目录触发替换失败，验证回滚到 2.0.0。
	bad := &Module{Manifest: Manifest{ID: "demo", Version: "3.0.0"}}
	missing := filepath.Join(root, "tmp", "missing")
	if _, _, err := r.upgrade(bad, &Module{Manifest: Manifest{ID: "demo", Version: "2.0.0"}}, missing, missing, filepath.Join(r.dir, "demo"), r.SidecarDir("demo")); err == nil {
		t.Fatal("替换失败时应返回错误")
	}
	got, ok := r.Get("demo")
	if !ok || got.Version != "2.0.0" {
		t.Fatalf("回滚后版本异常: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(r.dir, ".demo.bak")); err == nil {
		t.Fatal("回滚后不应残留备份目录")
	}
}

func TestRemoveCleansLogsAndEmptyData(t *testing.T) {
	r, root := newTestRegistry(t)
	zip1 := filepath.Join(root, "v1.zip")
	makeTestZip(t, zip1, "demo", "1.0.0")
	if _, _, err := r.InstallArchiveFile(zip1); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(r.LogDir(), "demo.log")
	if err := os.MkdirAll(r.LogDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := r.Remove("demo"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatal("卸载后日志未删除")
	}
	if _, err := os.Stat(r.ModuleDataDir("demo")); !os.IsNotExist(err) {
		t.Fatal("空用户数据目录应删除")
	}
}

func TestRemoveKeepsUserData(t *testing.T) {
	r, root := newTestRegistry(t)
	zip1 := filepath.Join(root, "v1.zip")
	makeTestZip(t, zip1, "demo", "1.0.0")
	if _, _, err := r.InstallArchiveFile(zip1); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(r.ModuleDataDir("demo"), "config.yaml")
	if err := os.WriteFile(keep, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := r.Remove("demo"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatal("非空用户数据应保留")
	}
}
