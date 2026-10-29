// Mica Home —— 模块化 Web 宿主。单二进制启动，模块即插即用。
package main

import (
	"context"
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fushuqian/mica_home/internal/host"
	"github.com/fushuqian/mica_home/internal/module"
	"github.com/fushuqian/mica_home/internal/sidecar"
)

const version = "0.1.0"

//go:embed all:web
var webFS embed.FS

func main() {
	addr := flag.String("addr", env("MICA_ADDR", ":8080"), "监听地址")
	modulesDir := flag.String("modules", env("MICA_MODULES_DIR", "modules"), "模块目录")
	dataDir := flag.String("data", env("MICA_DATA_DIR", "data"), "数据目录")
	flag.Parse()

	assets, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("加载内置前端失败: %v", err)
	}

	reg := module.NewRegistry(*modulesDir, filepath.Join(*dataDir, "state.json"))
	for _, warning := range reg.Scan() {
		log.Printf("模块告警: %s", warning)
	}

	srv := host.New(host.Options{
		Registry: reg,
		Assets:   assets,
		Version:  version,
		Sidecars: sidecar.New(*dataDir),
	})
	srv.StartSidecars()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	httpSrv := &http.Server{Addr: *addr, Handler: srv.Handler()}
	go func() {
		<-ctx.Done()
		srv.ShutdownSidecars()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	log.Printf("Mica Home v%s 启动于 %s（模块目录: %s）", version, *addr, *modulesDir)
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("服务退出: %v", err)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
