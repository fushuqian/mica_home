# Mica Home

模块化 Web 宿主。宿主只做最薄的一层——路由、身份、存储与沙箱，其余能力全部由模块提供：**装进来即生效，卸载不留痕**。

当前为 **v0.1.0 P0 骨架**：单二进制 Go 宿主 + 模块扫描 + 模块前端动态挂载 + Docker 分发。

## 快速开始

```bash
docker compose up -d --build
```

浏览器打开 <http://localhost:8080>。模块目录挂载自仓库的 `./modules`，新增模块后刷新页面即可看到。

宿主以 **Docker Linux 容器**运行，模块前端在用户浏览器中执行；容器内不提供本机进程、文件与命名管道能力。

停止：`docker compose down`（运行数据保存在 `mica-data` 卷中）。

## 目录结构

```
.
├── main.go                 # 宿主入口（flag/env 配置 + 内嵌前端）
├── internal/
│   ├── host/host.go        # HTTP 路由：静态外壳 / 模块 API / 模块前端挂载
│   └── module/module.go    # manifest 解析、目录扫描、安装状态持久化
├── modules/                # 模块目录（运行时挂载）
│   ├── hello/              # 示例：最小模块
│   └── notes/              # 示例：带本地数据的模块
├── web/index.html          # 宿主外壳页（编译进二进制）
├── web/manage.html         # 模块管理页 /manage（上传、启用、停用、卸载）
├── web/status.html         # 宿主机状态页 /status
├── web/docs-modules.html   # 模块开发指南 /docs/modules
├── web/docs-theme.html     # 主题设计指南 /docs/theme
├── web/docs.css            # 两册文档共用样式
├── web/docs.js             # 目录滚动高亮
├── web/theme.css           # 主题令牌（明/暗两套），宿主与模块共用
├── web/theme.js            # 主题切换与持久化
├── web/fonts/              # HarmonyOS Sans SC（Regular / Bold woff2）
├── docs/MICA-UI.md         # 面向其他系统/Agent 的单文件 UI 设计规范（令牌 + 组件 CSS）
├── docs/SIDECAR-CONTRACT.md # 模块 Sidecar 契约（宿主 ↔ 模块方）
├── design/                 # 早期 UI 原型存档
├── Dockerfile
└── docker-compose.yml
```

## 模块契约

一个模块就是一个目录：

```
modules/<id>/
├── manifest.json
└── frontend/
    └── index.js        # 入口，必须导出 mount(el, api)
```

`manifest.json`：

```json
{
  "id": "hello",
  "name": "Hello 模块",
  "version": "1.0.0",
  "description": "一句话说明",
  "author": "Mica Home",
  "category": "核心",
  "icon": "spark",
  "tone": "tone-a",
  "permissions": [],
  "frontend": { "entry": "index.js", "card": "card.js" }
}
```

- `id`：小写字母开头，仅含小写字母、数字、短横线，最长 32 字符。
- `category`：用于仓库页筛选。
- `icon` / `tone`：外观提示，可选。
- `permissions`：模块声明的能力，P0 仅展示，不做运行时拦截。
- `frontend.entry`：相对 `frontend/` 的入口文件。
- `frontend.card`：可选，首页卡片入口，导出 `renderCard(el, api)`；缺省用 manifest 默认卡片。

前端入口是**框架无关的 ES 模块**：

```js
export function mount(el, api) {
  el.innerHTML = '<p>Hello</p>';
  // 可选：返回卸载函数
  return () => { el.innerHTML = ''; };
}
```

`api` 由宿主注入：

| 字段 | 说明 |
|---|---|
| `api.id` / `api.version` | 当前模块标识 |
| `api.request(path, init?)` | 请求本模块命名空间 `/api/m/<id>/...`，自动加前缀 |
| `api.toast(text)` | 宿主统一提示 |
| `api.close()` | 关闭当前模块面板 |

## 宿主 API

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/health` | 宿主状态：版本、Go 运行时、模块数、运行时长、扫描告警 |
| GET | `/api/modules` | 全部模块清单（含 `sidecarState` / `sidecarPid` / `sidecarError` / `needConfirm`） |
| POST | `/api/modules/install` | 上传 zip 模块包（multipart 字段 `file`），流式落盘并安装；上限 160MB |
| POST | `/api/modules/{id}/enable` | 启用模块并启动 sidecar；权限未确认时返回 428 |
| POST | `/api/modules/{id}/confirm` | 确认 sidecar 权限，记录 `id + 指纹`；升级或权限变化后需重新确认 |
| POST | `/api/modules/{id}/disable` | 停用模块；同时停止 sidecar |
| DELETE | `/api/modules/{id}` | 卸载模块（删除模块、sidecar、日志与 socket，保留用户数据） |
| GET | `/api/m/{id}/info` | 单个模块的 manifest |
| GET | `/api/m/{id}/ui/{path}` | 模块前端静态资源（仅已启用模块可访问） |
| GET | `/api/m/{id}/__status` | sidecar 进程状态 |
| GET | `/api/m/{id}/__logs` | sidecar 日志尾部 |
| ANY | `/api/m/{id}/{path}` | 反代到模块 sidecar |

模块的启用状态持久化在 `MICA_DATA_DIR/state.json`。

## 主题

宿主提供浅色 / 深色两套主题，默认跟随系统 `prefers-color-scheme`，顶栏按钮可手动切换，选择记在 `localStorage['mica.theme']`。

主题的唯一定义在 `web/theme.css`：所有颜色、圆角、动效都以 `--mh-*` CSS 自定义属性暴露。宿主把当前主题写在 `<html data-theme="light|dark">`，模块只要引用令牌就会自动跟随，不需要任何监听代码。

```css
.mh-my-module { color: var(--mh-text); background: var(--mh-surface); border: 1px solid var(--mh-line); }
.mh-my-module button { background: var(--mh-text); color: var(--mh-bg); }
```

需要把颜色画进 Canvas / 图表时，监听 `window` 上的 `mh:theme` 事件；也可用 `window.MicaTheme.get() / set() / toggle()`。

完整规范见开发文档 `/docs/theme` 的《主题设计指南》。若要把这套设计语言带到其他系统，直接使用单文件规范 [docs/MICA-UI.md](docs/MICA-UI.md)（内含完整令牌表、组件 CSS、规则与自检清单）。

## 配置

| 环境变量 | 默认值 | 说明 |
|---|---|---|
| `MICA_ADDR` | `:8080` | 监听地址 |
| `MICA_MODULES_DIR` | `modules` | 模块目录 |
| `MICA_DATA_DIR` | `data` | 运行数据目录 |

## 路线图

- **P0（当前）**：宿主内核、模块扫描、前端挂载、Docker 分发。
- **P1**：OCI 模块仓库索引与在线安装、签名校验、PostgreSQL（schema-per-module）。
- **P2**：WASM 后端运行时（wazero）、依赖解析、`/__host/*` 双 token 管理通道。

模块后端契约已在 manifest 中预留（`backend: { type: "wasm", entry: "plugin.wasm" }`），P0 不执行。
