# Mica Home 模块 Sidecar 契约（定稿）

> 状态：宿主方与产品方已确认  
> 版本：v1.0 · 2026-10-28  
> 适用：Mihomo 控制台模块及后续需要容器内常驻进程的模块  
> 用途：模块方按本文修改需求与实现；宿主按本文实现安装器、supervisor 与反代

## 0. 一句话结论

**Sidecar 是模块自带的 `mihomo-watch` 监控程序，不是 mihomo 本体。**  
宿主只负责 `mihomo-watch` 的安装、启动、停止、日志、状态与 HTTP 反代；`mihomo-watch` 自己负责 mihomo 的进程、配置、订阅与控制口。

所有进程都运行在 **Mica Home 的 Docker Linux 容器**内。mihomo 的代理端口只在容器内回环监听，**不发布到宿主机，也不对局域网开放**。

### 术语

| 名词 | 含义 |
|---|---|
| Mica Home / 宿主 | 容器内的 Go 主进程（PID 1） |
| sidecar | 一种机制/角色：模块附带的常驻程序，不是某个固定程序 |
| `mihomo-watch` | 本模块的 sidecar 程序，由模块包提供 |
| sidecar supervisor | 宿主进程内部的管理代码，不是独立程序 |
| mihomo | 被 watch 管理的内核，不是 sidecar |

```
浏览器里的模块前端
        │  /api/m/<id>/*（宿主反代 + token）
Mica Home 宿主（容器内）
        │  启动 / 停止 / 重启 / 日志 / 状态
mihomo-watch（模块 sidecar）
        │  进程 / 配置 / 订阅 / 控制口
mihomo（容器内内核）
```

## 1. 职责边界

| 层 | 负责 | 不负责 |
|---|---|---|
| Mica Home | sidecar 安装与校验、生命周期、日志、`__status`/`__logs`、反代、启用确认 | 不认识 mihomo，不解析 mihomo 配置，不分配 mihomo 端口 |
| mihomo-watch | 启动/重启/停止 mihomo、写配置、拉订阅、暴露控制 API、管理 mihomo 端口 | 模块安装、前端鉴权、宿主级权限 |
| 模块前端 | 通过 `/api/m/<id>/*` 展示与控制 | 不直连容器、不接触 token/端口 |

## 2. 模块包与安装布局

### 2.1 zip 结构

```
manifest.json
frontend/
sidecar/
  mihomo-watch        # 宿主执行的入口
  mihomo              # 由 watch 拉起，宿主不直接执行
```

- `manifest.json`、`frontend/` 位于包根，不能多套文件夹。
- `sidecar/` 下可以放多个文件；只有 `executables` 声明的文件会被宿主赋予可执行位。
- **sidecar 文件不得进入 `modules/<id>/`**。

### 2.2 安装后的目录

```
modules/<id>/                 # 模块前端
/data/sidecar/<id>/           # watch + mihomo 二进制（卸载时删除）
/data/modules/<id>/           # 模块用户数据（配置、订阅缓存、规则；卸载默认保留）
/data/run/<id>.sock           # watch 控制通道（可选，Unix socket）
/data/logs/<id>.log           # watch/mihomo 日志（宿主滚动 5MB × 3）
```

### 2.3 体积与平台

| 项 | 限制 |
|---|---|
| 前端包（manifest + frontend） | 20MB |
| sidecar 包（sidecar/） | 128MB |
| 单次上传 zip | 160MB |
| 平台 | 首版仅 `linux/amd64`，必须静态编译（`CGO_ENABLED=0`） |

宿主会校验平台与 `sha256`；不匹配直接拒绝安装。

## 3. manifest 契约

```json
{
  "id": "mihomo",
  "name": "Mihomo 控制台",
  "version": "1.0.0",
  "frontend": { "entry": "index.js", "card": "card.js" },
  "sidecar": {
    "binary": "sidecar/mihomo-watch",
    "executables": ["sidecar/mihomo-watch", "sidecar/mihomo"],
    "platform": "linux/amd64",
    "sha256": {
      "sidecar/mihomo-watch": "<64 位 hex>",
      "sidecar/mihomo": "<64 位 hex>"
    },
    "args": [],
    "env": {},
    "healthPath": "/health",
    "socketName": "mihomo-watch.sock",
    "permissions": ["process.run", "fs.write:/data/modules/mihomo", "net.listen"]
  }
}
```

| 字段 | 必填 | 说明 |
|---|---|---|
| `binary` | 是 | 宿主执行的入口，必须在 `sidecar/` 下 |
| `executables` | 否 | 需要 `0755` 的文件；默认只包含 `binary` |
| `platform` | 是 | 首版只接受 `linux/amd64` |
| `sha256` | 是 | 文件相对路径 → SHA256；至少覆盖 `executables` 中的文件 |
| `args` | 否 | 参数数组，不经过 shell；支持 `${MICA_*}` 占位符 |
| `env` | 否 | 额外环境变量；不得包含宿主机密钥 |
| `healthPath` | 否 | 默认 `/health`（管理通道） |
| `socketName` | 否 | 指定后宿主使用 `/data/run/<socketName>`；缺省时宿主分配回环端口 |
| `permissions` | 是 | 权限声明，安装展示、启用确认使用 |

**不再包含**：`ports`、`autoStart`、`stopOnDisable`。  
启用即启动、停用即停止，由宿主固定执行；mihomo 端口由 watch 自己管理。

### 3.1 宿主注入的环境变量

| 变量 | 值 |
|---|---|
| `MICA_DATA_DIR` | `/data` |
| `MICA_MODULE_ID` | 模块 id |
| `MICA_MODULE_DIR` | `/app/modules/<id>` |
| `MICA_MODULE_DATA_DIR` | `/data/modules/<id>` |
| `MICA_SIDECAR_DIR` | `/data/sidecar/<id>` |
| `MICA_SIDECAR_ADDR` | `unix:///data/run/<socket>` 或 `tcp://127.0.0.1:<port>` |
| `MICA_SIDECAR_TOKEN` | 每次启动随机生成；宿主反代模块请求时注入 `X-Mica-Token` |
| `MICA_HOST_TOKEN` | 每次启动随机生成；仅宿主管理请求使用 `X-Mica-Host-Token` |

`args` 中可以使用 `${MICA_SIDECAR_ADDR}` 等占位符，宿主启动前替换。

## 4. mihomo-watch 接口契约

> 首版实现：宿主直接调用 manifest 的 `healthPath`（默认 `/health`），统一带 `X-Mica-Token`；`/__host/*` 预留给后续双 token 管理通道，当前被反代拒绝。

watch 必须把接口分成**两条通道**，不能把所有能力都暴露给模块前端。

| 通道 | 调用方 | 用途 | 鉴权 |
|---|---|---|---|
| 管理通道 `healthPath`（默认 `/health`） | 仅宿主 | 健康检查 | `X-Mica-Token` |
| 模块通道 `/api/m/<id>/*` → `/` | 模块前端（经宿主反代） | 配置、订阅、节点、规则、mihomo 控制 | `X-Mica-Token` |

宿主反代**必须拒绝** `/__host/*`，模块前端永远访问不到管理口。

### 4.1 宿主保留路径（不会转发）

- `GET /api/m/<id>/info`
- `GET /api/m/<id>/ui/*`
- `GET /api/m/<id>/__status`
- `GET /api/m/<id>/__logs`
- 任何 `/__host/*`

watch 不要使用 `info`、`ui`、`__` 开头的路径作为模块业务接口。

### 4.2 管理通道（host-only）

| 方法 | 路径 | 返回 |
|---|---|---|
| GET | `healthPath`（默认 `/health`） | `200 {"ok":true,"version":"x.y.z"}`，watch 自身存活即可 |

- 宿主启动 watch 后轮询 `healthPath`，通过后才把模块标记为 `running`。
- 管理请求带 `X-Mica-Token`，缺失或不匹配返回 401。
- `/__host/*` 预留给后续双 token 管理通道，当前被宿主反代拒绝。

### 4.3 模块通道

宿主把 `/api/m/<id>/<path>` 反代到 watch 的同一路径，保留方法、请求体、查询参数、状态码、`Content-Type`。

watch 在模块通道上自行定义业务接口，宿主不解析、不限制具体路径。例如：

- `GET /status`：mihomo 状态
- `GET /version`
- `GET /proxies`
- `GET /configs`、`PUT /configs`
- `POST /config/inject`
- `POST /fetch`

`/status` 示例：

```json
{
  "mihomo": "running",
  "pid": 123,
  "version": "v1.18.0",
  "startedAt": "2026-10-28T12:00:00Z",
  "ports": { "mixed": 7890 },
  "lastError": ""
}
```

### 4.4 网络与鉴权

- watch 只监听 `127.0.0.1` 或 Unix socket，**不得监听 `0.0.0.0`**。
- 管理请求校验 `X-Mica-Host-Token`；模块反代请求校验 `X-Mica-Token`。
- 不向模块前端暴露 mihomo 控制口的 secret。
- 宿主反代只允许已启用模块访问，否则 403。

## 5. 生命周期

> 宿主侧实现细节（安装、supervisor、反代、状态接口、实施顺序）见开发文档 `/docs/modules` 第 10 节《宿主侧实现》。

| 事件 | 宿主行为 | watch 行为 |
|---|---|---|
| 安装 | 解压前端与 sidecar、校验平台/哈希、创建 `/data/modules/<id>/` | — |
| 启用 | 启动 watch，等待 `/__host/health` 通过（≤5 秒） | 启动 mihomo，准备控制口 |
| 停用 | SIGTERM watch，10 秒后杀整个进程组 | 停止 mihomo，确保子进程退出 |
| 卸载 | 停 watch → 删 `/data/sidecar/<id>` → 保留用户数据 | 退出前清理 mihomo |
| 升级 | 停旧 watch → 备份 → 替换 → 启动 → 失败回滚 | 按新版本启动 |
| watch 崩溃 | 1/2/5/10 秒退避重启，最多 5 次；稳定 60 秒重置 | 启动时清理残留 mihomo |
| mihomo 崩溃 | 不介入 | 自行重启 mihomo |
| 宿主退出 | 停 HTTP → 停全部 watch | 停止 mihomo |

- 宿主使用进程组 + `Pdeathsig`，防止孤儿进程。
- 启用后 watch 未就绪：模块保持启用，状态标记 `degraded`，并记录原因。
- 宿主重启后，已启用模块的 watch 自动拉起。

### 5.1 状态枚举

宿主 `__status` 返回：

```json
{
  "state": "running",
  "pid": 456,
  "startedAt": "2026-10-28T12:00:00Z",
  "restarts": 0,
  "exitCode": 0,
  "lastError": ""
}
```

`state`：`stopped | starting | running | stopping | degraded`

`/api/modules` 每项增加 `sidecar: { "state": "running", "error": "" }`。

## 6. 数据面（代理端口）

- mihomo 的 `mixed-port`（如 7890）**只在容器内 `127.0.0.1` 监听**。
- `docker-compose.yml` **不发布**该端口；宿主机与局域网均不可见。
- `allow-lan: false`，不得监听 `0.0.0.0`。
- 浏览器模块不能把 7890 当 HTTP 代理使用；它只能通过 `/api/m/<id>/*` 做控制。
- 容器内其他进程（宿主、未来的 WASM 后端模块）可以连接 `127.0.0.1:7890`。

## 7. 权限与安全

1. 安装页展示：sidecar 文件名、平台、SHA256、体积、权限声明。
2. 启用时弹窗确认；同意记录为 `id + sha256`，升级后重新确认。
3. 宿主只执行 `binary` 声明的文件，参数数组直传，不经过 shell。
4. watch 必须用进程组管理 mihomo，退出时清理子进程。
5. 反代只允许已启用模块访问，否则 403。
6. 宿主注入随机 token；watch 校验 `X-Mica-Token`。
7. 不向 sidecar 传递宿主机/Docker 密钥。
8. 宿主记录生命周期与代理访问审计日志。

> 诚实说明：宿主能限制“执行哪个二进制、给哪些环境变量、反代哪些请求”，但无法在操作系统层完全限制原生进程读写 `/data`。sidecar 仍应视为受信任代码。

## 8. 安装、升级与卸载

- **同 ID 升级**：版本不同允许升级；先停 watch、备份旧 `modules/<id>` 与 `/data/sidecar/<id>`、替换、启动；失败回滚旧版本。
- **同版本重装**：拒绝（409）。
- **卸载**：停止 watch，删除 `/data/sidecar/<id>`，保留 `/data/modules/<id>`；用户可另行清除数据。
- **日志**：宿主捕获 watch stdout/stderr 到 `/data/logs/<id>.log`，5MB × 3 滚动；watch 需把 mihomo 输出并入自己的 stdout/stderr。

## 9. 与旧需求文档的差异（模块方修改清单）

| 旧文档 | 需要改成 |
|---|---|
| sidecar = mihomo 本体 | sidecar = `mihomo-watch`；mihomo 由 watch 管理 |
| manifest 有 `ports` | 删除；mihomo 端口由 watch 管理，宿主不分配、不发布 |
| manifest 有 `autoStart` / `stopOnDisable` | 删除；启用即启动、停用即停止 |
| `/api/m/<id>/*` 直接反代 mihomo | 反代到 watch；watch 再代理 mihomo 控制接口 |
| 状态只有一层 | 两层：`__status` = watch；模块通道 `/status` = mihomo |
| 所有接口都暴露给模块前端 | 管理通道 `/__host/*` 仅宿主；模块业务接口才走反代 |
| “其他模块用 `127.0.0.1:7890`” | 浏览器模块不可直连；容器内进程才可 |
| 宿主分配 mihomo 端口 | watch 管理；宿主只管理 watch 控制通道 |
| 权限“启用时确认”未落地 | 保留：安装展示 + 启用确认 |
| 验收只测 mihomo | 同时测 watch 与 mihomo 的启停、崩溃恢复、孤儿清理 |

## 10. 验收清单

- [ ] 安装后 `/data/sidecar/<id>/` 存在 watch 与 mihomo；`modules/<id>/` 内无二进制。
- [ ] 启用后 ≤5 秒 `__status` 为 `running`。
- [ ] `/status` 显示 mihomo `running`、PID、版本、实际端口。
- [ ] `/api/m/<id>/version` 返回 mihomo 版本，前端无需知道端口与 secret。
- [ ] 停用后 `ps` 中 watch 与 mihomo **都消失**。
- [ ] `kill mihomo` → watch 自动拉起；`kill watch` → 宿主拉起并清理旧 mihomo。
- [ ] 宿主重启后，已启用模块自动恢复。
- [ ] 卸载后 `/data/sidecar/<id>` 删除，`/data/modules/<id>` 保留。
- [ ] `docker port mica-home` 看不到 7890；宿主机访问 7890 失败。
- [ ] 未启用模块访问 `/api/m/<id>/*` 返回 403。
- [ ] 模块前端访问 `/api/m/<id>/__host/*` 返回 403。
- [ ] 无 `X-Mica-Token` 的模块请求返回 401；无 `X-Mica-Host-Token` 的管理请求返回 401。
- [ ] 宿主能通过 `/__host/health` 判断就绪，模块无关接口不受影响。
- [ ] 错误平台或 SHA256 不匹配的包安装失败。
- [ ] 升级失败能回滚到旧版本并恢复运行。
- [ ] 日志文件存在且按 5MB × 3 滚动。

## 11. 待模块方确认

1. mihomo 是否支持用启动参数指定控制地址（Unix socket 或回环端口）？若不支持，watch 需要生成/注入配置片段。
2. watch 的控制 API 是“根路径代理 mihomo”（推荐），还是统一放在 `/mihomo/*` 下？
3. watch 如何识别并清理上一次残留的 mihomo（pid 文件 / 进程名校验 / 端口探测）？
4. mihomo 的具体版本与构建参数（静态、linux/amd64）是否随模块版本固定？
5. `sidecar/` 内除 watch、mihomo 外是否还需要其他资源文件？
