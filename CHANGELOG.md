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
- 新增开发文档页 `/docs`（`web/docs.html`）：职责边界、模块结构、manifest 字段表、前端契约与 api 说明、宿主 API 表 + 接口试跑、本地开发流程、能力边界；顶栏新增"开发文档"入口。

### 新增 · 主题切换与主题模板
- 新增 `web/theme.css`：以 `--mh-*` CSS 自定义属性定义浅色/深色两套令牌（背景、文字、线条、玻璃表面、强调色、阴影、遮罩、代码块、警告、圆角、缓动、字体），并保留旧的 `--bg` / `--ink` / `--teal` 等别名。
- 新增 `web/theme.js`：默认跟随系统 `prefers-color-scheme`，手动选择记忆在 `localStorage['mica.theme']`；暴露 `window.MicaTheme`（`get/set/toggle`）并广播 `mh:theme` 事件。
- 宿主外壳页与 `/docs` 顶栏新增主题切换按钮，图标随主题切换；两页全部硬编码颜色改为令牌引用。
- 示例模块 `hello` / `notes` 改用主题令牌，模块面板随宿主主题自动变化（无需监听）。
- 开发文档新增「8 · 主题开发规范」：原则、命名规则、最小令牌集、允许与禁止、样式注入约定、可复制模板、自检清单、验证方法。

### 修复
- 模块舞台打开后永久停留在"正在加载模块前端…"：`openModule` 先取 token 再调用会自增 token 的 `cleanupStage()`，导致守卫误判提前返回。改为先失效再取 token。
- 顶栏移除无实际用途的"模块 API"链接（直接打开原始 JSON，没有可用界面）。
- 模块卡片移除悬停时白光扫过效果，保留上浮、阴影与边框高亮。

### 早前
- 确认架构方向：Go 宿主 + WASM（wazero）后端模块 + 远程 ESM 前端 + PostgreSQL schema-per-module + OCI 模块仓库，Docker 单容器分发。
