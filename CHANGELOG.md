# CHANGELOG

## 2026-10-28

### v0.1.0 · P0 骨架
- 新增 Go 宿主内核 `main.go` + `internal/host` + `internal/module`：内嵌前端、模块目录扫描、安装状态持久化、请求日志。
- 新增宿主 API：`/api/health`、`/api/modules`、`/api/modules/{id}/install|uninstall`、`/api/m/{id}/info`、`/api/m/{id}/ui/*`。
- 新增模块契约：`manifest.json` + `frontend/index.js`（导出 `mount(el, api)`），模块前端由宿主动态 import 挂载。
- 新增示例模块：`hello`（默认启用，演示 API 注入）、`notes`（本地存储便签）。
- 新增宿主外壳页 `web/index.html`：沿用浅色毛玻璃视觉，数据全部来自宿主 API，含筛选、搜索、安装/卸载与模块舞台弹层。
- 新增 `Dockerfile`（多阶段，容器内编译，本地无需 Go）+ `docker-compose.yml` + `README.md` + `.gitignore`。
- 早期 UI 原型 `index.html` 归档到 `design/marketplace-prototype.html`。

### 修复
- 模块舞台打开后永久停留在"正在加载模块前端…"：`openModule` 先取 token 再调用会自增 token 的 `cleanupStage()`，导致守卫误判提前返回。改为先失效再取 token。
- 顶栏移除无实际用途的"模块 API"链接（直接打开原始 JSON，没有可用界面）。
- 模块卡片移除悬停时白光扫过效果，保留上浮、阴影与边框高亮。

### 早前
- 确认架构方向：Go 宿主 + WASM（wazero）后端模块 + 远程 ESM 前端 + PostgreSQL schema-per-module + OCI 模块仓库，Docker 单容器分发。
