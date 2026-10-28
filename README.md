# Mica Home

模块化 Web 宿主。宿主只做最薄的一层——路由、身份、存储与沙箱，其余能力全部由模块提供：**装进来即生效，卸载不留痕**。

当前为 **v0.1.0 P0 骨架**：单二进制 Go 宿主 + 模块扫描 + 模块前端动态挂载 + Docker 分发。

## 快速开始

```bash
docker compose up -d --build
```

浏览器打开 <http://localhost:8080>。模块目录挂载自仓库的 `./modules`，新增模块后刷新页面即可看到。

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
  "frontend": { "entry": "index.js" }
}
```

- `id`：小写字母开头，仅含小写字母、数字、短横线，最长 32 字符。
- `category`：用于仓库页筛选。
- `icon` / `tone`：外观提示，可选。
- `permissions`：模块声明的能力，P0 仅展示，不做运行时拦截。
- `frontend.entry`：相对 `frontend/` 的入口文件。

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
| GET | `/api/modules` | 全部模块清单（含 `installed` 状态） |
| POST | `/api/modules/{id}/install` | 安装（标记启用） |
| POST | `/api/modules/{id}/uninstall` | 卸载 |
| GET | `/api/m/{id}/info` | 单个模块的 manifest |
| GET | `/api/m/{id}/ui/{path}` | 模块前端静态资源（仅已安装模块可访问） |

安装状态持久化在 `MICA_DATA_DIR/state.json`。

## 配置

| 环境变量 | 默认值 | 说明 |
|---|---|---|
| `MICA_ADDR` | `:8080` | 监听地址 |
| `MICA_MODULES_DIR` | `modules` | 模块目录 |
| `MICA_DATA_DIR` | `data` | 运行数据目录 |

## 路线图

- **P0（当前）**：宿主内核、模块扫描、前端挂载、Docker 分发。
- **P1**：OCI 模块仓库索引与在线安装、签名校验、PostgreSQL（schema-per-module）。
- **P2**：WASM 后端运行时（wazero）、热升级与回滚、依赖解析、权限授权页。

模块后端契约已在 manifest 中预留（`backend: { type: "wasm", entry: "plugin.wasm" }`），P0 不执行。
