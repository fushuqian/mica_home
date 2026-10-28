// Mica Home —— 模块化 Web 宿主。单二进制启动，模块即插即用。
package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/fushuqian/mica_home/internal/host"
	"github.com/fushuqian/mica_home/internal/module"
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

	srv := host.New(host.Options{Registry: reg, Assets: assets, Version: version})

	log.Printf("Mica Home v%s 启动于 %s（模块目录: %s）", version, *addr, *modulesDir)
	if err := http.ListenAndServe(*addr, srv.Handler()); err != nil {
		log.Fatalf("服务退出: %v", err)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
