# Mica Home 模块开发说明（AI AGENT 版）

> 面向自动化编码 Agent。内容与 `/docs/modules`、`docs/SIDECAR-CONTRACT.md` 同步。
> 适用版本：Mica Home 0.1.0 · 最后更新：2026-10-30
> 读取方式：`GET /docs/agent.md`（`text/markdown`），无需渲染。

## 0. 先读结论

- 宿主是一个运行在 **Linux 容器**内的 Go 单进程（PID 1），运行层为 `debian:bookworm-slim`（glibc 2.36），模块前端在用户浏览器里执行。
- 模块 = `manifest.json` + 纯 ESM 前端；需要常驻进程时再加 `sidecar/`。
- 模块包是一个 zip，包根**直接**放 `manifest.json`、`frontend/`、可选 `sidecar/`，不能多套一层目录。
- 宿主只执行 manifest 的 `binary`，参数数组直传，不经过 shell。
- 模块前端**不能**直连容器、不能拿 token、不能访问 `__host`。

## 1. 包结构与限额

```
manifest.json
frontend/
  index.js          # 必须导出 mount(el, api)
  card.js           # 可选，导出 renderCard(el, api)
sidecar/
  <watch 二进制>     # 宿主执行的入口
  <其他文件>         # 只有 executables 声明的文件会被 chmod 0755
```

| 项 | 上限 |
|---|---|
| 单次上传 zip | 256MB |
| 前端部分（manifest + frontend/） | 20MB |
| sidecar 部分（sidecar/） | 512MB |
| 单个文件解压后 | 512MB |
| 包内文件数 | 30000 |
| 平台 | 仅 `linux/amd64` |

安装后目录：

```
modules/<id>/            模块前端（卸载时删除）
/data/sidecar/<id>/      sidecar 二进制（卸载时删除）
/data/modules/<id>/      模块用户数据（卸载时保留）
/data/run/<socketName>   sidecar 控制 socket
/data/logs/<id>.log      stdout/stderr，5MB 滚动，保留 .1/.2
```

## 2. manifest 契约

```json
{
  "id": "demo",
  "name": "Demo",
  "version": "1.0.0",
  "description": "一句话说明",
  "author": "Mica Home",
  "category": "网络",
  "icon": "radar",
  "tone": "tone-c",
  "permissions": [],
  "frontend": { "entry": "index.js", "card": "card.js" },
  "sidecar": {
    "binary": "sidecar/demo-watch",
    "executables": ["sidecar/demo-watch", "sidecar/demo-core"],
    "platform": "linux/amd64",
    "sha256": {
      "sidecar/demo-watch": "<64 位 hex>",
      "sidecar/demo-core": "<64 位 hex>"
    },
    "args": [],
    "env": {},
    "healthPath": "/health",
    "healthTimeoutSec": 5,
    "socketName": "demo-watch.sock",
    "permissions": ["process.run", "fs.write:/data/modules/demo", "net.listen"]
  }
}
```

字段规则：

- `id`：`^[a-z][a-z0-9-]{1,31}$`，全小写。
- `version`：语义化版本；同 ID 不同版本可升级，同版本重装返回 409。
- `frontend.entry` / `frontend.card`：相对 `frontend/` 的路径，必须存在。
- `sidecar.binary`：必须以 `sidecar/` 开头。
- `sidecar.platform`：只接受 `linux/amd64`；不匹配拒绝安装。
- `sidecar.sha256`：相对路径 → 小写 hex；至少覆盖 `executables`。不匹配拒绝安装。
- `sidecar.args`：数组，不经 shell；支持 `${MICA_*}` 占位符。
- `sidecar.env`：追加环境变量；不得放宿主机密钥。
- `sidecar.healthPath`：默认 `/health`。
- `sidecar.healthTimeoutSec`：默认 5，范围 1–120；超时只标记 `degraded`，进程不会被杀。
- `sidecar.socketName`：可选；有值时宿主用 Unix socket，否则分配 `127.0.0.1` 回环端口。
- `sidecar.permissions`：非空时，安装后**不会自动启用**，必须先确认。

## 3. 前端契约

```js
// frontend/index.js
export function mount(el, api) {
  el.innerHTML = '<p>Hello</p>';
  return () => { el.innerHTML = ''; }; // 可选卸载函数
}

// frontend/card.js（可选）
export function renderCard(el, api) {
  el.innerHTML = '<div>自定义卡片内容</div>';
}
```

`api` 注入字段：

| 字段 | 说明 |
|---|---|
| `api.id` / `api.version` | 当前模块标识 |
| `api.request(path, init?)` | 请求 `/api/m/<id>/<path>`，自动加前缀 |
| `api.toast(text)` | 宿主统一提示 |
| `api.close()` | 关闭模块面板 |

约束：

- 卡片渲染限时 1.5 秒；抛错/超时会回退 manifest 默认卡片。
- 卡片内按钮需阻止事件冒泡到“打开面板”或使用 `data-no-open`。
- 不要把宿主 CSS 类名当状态类复用（宿主状态类统一 `is-*`）。
- 主题跟随宿主：用 `var(--mh-*)` 令牌，不写死颜色。

## 4. sidecar 契约

### 4.1 宿主注入环境变量

| 变量 | 值 |
|---|---|
| `MICA_DATA_DIR` | `/data` |
| `MICA_MODULE_ID` | 模块 id |
| `MICA_MODULE_DIR` | `/app/modules/<id>` |
| `MICA_MODULE_DATA_DIR` | `/data/modules/<id>` |
| `MICA_SIDECAR_DIR` | `/data/sidecar/<id>` |
| `MICA_SIDECAR_ADDR` | `unix:///data/run/<socket>` 或 `tcp://127.0.0.1:<port>` |
| `MICA_SIDECAR_TOKEN` | 每次启动随机生成；宿主反代时注入 `X-Mica-Token` |
| `MICA_HOST_TOKEN` | 当前与 `MICA_SIDECAR_TOKEN` 相同（双 token 未实现） |

### 4.2 接口通道

| 通道 | 调用方 | 鉴权 | 用途 |
|---|---|---|---|
| `healthPath`（默认 `/health`） | 仅宿主 | `X-Mica-Token` | 就绪判断 |
| `/api/m/<id>/*` → watch 同路径 | 模块前端（经宿主） | `X-Mica-Token` | 业务接口 |

- 宿主保留前缀，不要使用：`info`、`ui/`、`__status`、`__logs`、`__host`。
- 反代使用 `httputil.ReverseProxy`，支持 WebSocket/Upgrade、流式响应、`Range`。
- 未启用模块 → 403；`__host` → 403；watch 未带 token → 401（watch 自己校验）。
- 模块前端永远拿不到 token，也不应尝试。

### 4.3 生命周期（宿主负责）

| 事件 | 行为 |
|---|---|
| 启用 | 启动 watch；轮询 `healthPath`，超时 `healthTimeoutSec`；失败标 `degraded` |
| 停用 | SIGTERM 整个进程组，12 秒后 SIGKILL |
| 崩溃 | 1/2/5/10/10 秒退避重启，最多 5 次；稳定 60 秒重置 |
| 宿主重启 | 已启用且已确认的模块自动拉起 |
| 升级 | 停旧 → 备份 → 替换 → 重新扫描；失败自动回滚 |
| 卸载 | 停 watch，删除 sidecar、日志、socket 与空数据目录，保留非空用户数据 |

watch 要求：

- 只监听 `127.0.0.1` 或 Unix socket，不得 `0.0.0.0`。
- 自己拉起的内核必须留在同一进程组（不要 setsid），否则停用会残留。
- stdout/stderr 直接输出，宿主负责写日志与滚动。
- 退出时清理自己拉起的子进程。

### 4.4 状态枚举

`GET /api/m/<id>/__status`：

```json
{
  "state": "running",
  "pid": 456,
  "startedAt": "2026-10-30T03:00:00+08:00",
  "restarts": 0,
  "exitCode": 0,
  "lastError": ""
}
```

`state`：`stopped | starting | running | stopping | degraded`

### 4.5 权限确认

- 安装响应返回 `needConfirm`、`permissions`、`fingerprint`。
- 未确认时 `POST /api/modules/<id>/enable` 返回 **428**，body 同安装响应。
- `POST /api/modules/<id>/confirm` 记录 `id + 指纹`；升级或权限变化后指纹失效，需要重新确认。

## 5. 宿主 HTTP API

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/health` | 版本、Go 版本、模块数、运行时长、扫描告警 |
| GET | `/api/system` | 限额、数据盘容量、`data.writable` |
| GET | `/api/modules` | 模块列表，含 `sidecarState` / `sidecarPid` / `sidecarError` / `needConfirm` |
| POST | `/api/modules/install` | multipart 字段 `file`（zip），流式落盘 |
| POST | `/api/modules/{id}/enable` | 启用并启动 sidecar；未确认返回 428 |
| POST | `/api/modules/{id}/confirm` | 确认 sidecar 权限 |
| POST | `/api/modules/{id}/disable` | 停用并停止 sidecar |
| DELETE | `/api/modules/{id}` | 卸载 |
| GET | `/api/m/{id}/info` | manifest |
| GET | `/api/m/{id}/ui/{path}` | 模块前端静态资源 |
| GET | `/api/m/{id}/__status` | sidecar 状态 |
| GET | `/api/m/{id}/__logs?tail=N` | sidecar 日志尾部 |
| ANY | `/api/m/{id}/{path}` | 反代到 sidecar |

## 6. 主题

- 令牌唯一定义在 `web/theme.css`，宿主在 `<html data-theme="light|dark">` 上切换。
- 最小用法：

```css
.mh-demo { color: var(--mh-text); background: var(--mh-surface); border: 1px solid var(--mh-line); }
.mh-demo button { background: var(--mh-text); color: var(--mh-bg); }
```

- Canvas/图表监听 `window` 的 `mh:theme` 事件，或 `window.MicaTheme.get()/set()/toggle()`。
- 完整令牌与组件规范见 `/docs/theme` 与 `docs/MICA-UI.md`。

## 7. 打包与校验

```bash
# 1) 生成 sidecar 文件哈希
sha256sum sidecar/*

# 2) zip 必须无顶层嵌套
zip -r demo.zip manifest.json frontend sidecar

# 3) 平台必须是 linux/amd64
file sidecar/demo-watch
```

注意：x86-64 指令集。若 sidecar 或内核二进制用 `GOAMD64=v3`（mihomo `-v3` 构建）编译，在较老的 NAS CPU（无 AVX2）上会启动失败。统一用 baseline/v1 或 `-compatible` 构建。

## 8. 验收清单（模块方自测）

- [ ] 安装成功；`/data/sidecar/<id>` 有二进制，`/app/modules/<id>` 只有前端。
- [ ] 启用后 `healthTimeoutSec` 内 `__status` 为 `running`。
- [ ] `/api/m/<id>/version` 或业务接口经反代可用；WS 接口可升级。
- [ ] 停用后 watch 与其子进程都消失。
- [ ] kill watch 后 1/2/5/10 秒内自动恢复；宿主重启后自动恢复。
- [ ] 未启用访问返回 403；`__host` 返回 403；无 token 的直连返回 401。
- [ ] 卸载后 sidecar 删除、用户数据保留。
- [ ] 同版本重装 409；不同版本升级成功。
- [ ] 错误平台/错误 SHA256 的包安装失败且不影响旧版本。

## 9. 常见错误

| 现象 | 原因 | 处理 |
|---|---|---|
| 上传后 `Failed to fetch`，宿主日志 `install 500 0s` | `/data` 或 `/app/modules` 对 UID 10001 不可写 | `chown -R 10001:10001 data modules`；`/api/system` 的 `data.writable` 会提前暴露 |
| 安装返回 428 | sidecar 声明了权限且未确认 | 先 `POST /confirm` 再 `enable` |
| 安装返回 409 | 同 ID 同版本 | 改版本号或先卸载 |
| 安装返回 `sidecar 解压后超过 512 MB` | 超出限额 | 精简运行时或拆分 |
| 安装返回 `模块包文件数超过 30000` | venv/node_modules 文件过多 | 裁剪依赖或减少文件 |
| 启用后 `degraded`，日志有 `v3 microarchitecture` | 二进制用 GOAMD64=v3 | 换 baseline 构建 |
| sidecar 启动即退出 | 入口不存在/动态库缺失/权限 | 看 `/__logs`；glibc 依赖用镜像自带 glibc |
| 反代 404 `sidecar 未运行` | 模块未启用或进程未就绪 | 先启用，看 `__status` |

## 10. 明确不支持

- 模块声明 volume 或额外挂载点（由 compose 层解决）。
- `/__host/*` 双 token 管理通道（当前被拒绝）。
- service / shared sidecar，一个模块最多一个 sidecar。
- 宿主内的本机进程、文件、命名管道能力。
- 业务端口（如 7890）发布到宿主机或局域网。
- 模块签名校验（仅 SHA256）。

## 11. 文档同步责任

以下三处必须一起改，发版前核对：

1. `web/agent-guide.md`（本文件，供 AI）
2. `web/docs-modules.html`（供人阅读）
3. `docs/SIDECAR-CONTRACT.md`（供模块方）

需要同步的具体项：包结构、限额、manifest 字段、环境变量、HTTP 路由、状态枚举、生命周期、验收清单。
