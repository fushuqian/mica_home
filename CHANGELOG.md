# CHANGELOG

## 2026-10-29

### 文档 · 补 Docker Hub 与 GitHub 两种安装说明
- README 新增「安装与运行」：Docker Hub 直接 `docker run`（数据/模块命名卷）、Compose 镜像版、GitHub 源码构建、本地 Go 调试、验证与升级步骤。
- 明确镜像不内置 mihomo 等第三方模块，需在 `MODULES` 页上传安装。

### 新增 · Sidecar 首版收尾
- 崩溃自动重启：1/2/5/10/10 秒退避，最多 5 次；稳定运行 60 秒后重置计数；watch 崩溃后回收被容器 init 收养的僵尸子进程。
- 日志轮转：`/data/logs/<id>.log` 超过 5MB 滚动为 `.1` / `.2`。
- 同 ID 版本升级：停旧 watch → 备份模块与 sidecar → 替换 → 重新扫描；失败自动回滚旧版本。同版本重装仍返回 409。
- 上传改流式：`multipart` 直接写入 `/data/tmp`，不再整包读入内存。
- 权限确认：sidecar 声明了 `permissions` 的模块安装后不自动启用；`enable` 返回 428，`POST /api/modules/{id}/confirm` 记录 `id + 指纹`，升级或权限变化后需重新确认。
- `/api/modules` 增加 `sidecarState` / `sidecarPid` / `sidecarError` / `needConfirm`；管理页展示 sidecar 状态并提供权限确认弹窗。
- 卸载清理 sidecar 目录、日志、socket 与空用户数据目录；非空用户数据保留。
- 补充 `internal/module` 单元测试：安装/升级、同版本 409、升级失败回滚、卸载清理与用户数据保留。

### 调整 · 首页搜索框移入模块仓库标题区
- 顶栏移除搜索框，只保留品牌、导航与主题按钮；顶栏不再有居中的搜索位。
- 搜索框移到「模块仓库」标题行左侧，替换原来的统计文字「已启用 N 个模块 · 已安装 N 个」，并删除对应样式与 `shelf-summary` 统计更新逻辑。

### 调整 · 移除顶栏 ONLINE 胶囊
- 删除首页搜索栏右侧的 `ONLINE · x/y` 状态胶囊及其样式与 JS 更新逻辑。
- STATUS 页顶栏同步移除该胶囊，页面内仍保留 `LIVE` 状态与自动刷新。

### 新增 · STATUS 独立页
- 新增 `/status` 页面：模块总数、已启用、运行时长、宿主版本、运行时、模块目录、扫描告警，每 5 秒自动刷新。
- 顶部导航统一为 `DASHBOARD / MODULES / STATUS / DEVGUIDES`，STATUS 位于 MODULES 与 DEVGUIDES 之间。
- 首页移除 Host Status 面板，只保留模块摘要、筛选与卡片。

### 调整 · 顶部导航统一
- 三个一级入口统一为 `DASHBOARD / MODULES / DEVGUIDES`，中英文写法保持一致。
- 修复开发文档页顶部导航缺少「模块管理」项的问题；两个文档页现在也显示完整三项，`DEVGUIDES` 标记当前页。
- 首页模块区入口同步改为 `MODULES`。

### 修复 · 大模块包上传被截断
- 上传上限从 20MB 提到 160MB；前端/解压限制拆成前端 20MB、sidecar 128MB、单文件 128MB，解决 `mihomo-2.0.1.zip`（25.9MB）被 `MaxBytesReader` 断连的问题。
- `/api/m/<id>/*` 改由统一前缀路由处理，避免 Go 1.24 ServeMux 方法/通配符优先级冲突导致宿主启动 panic。

### 新增 · 宿主 Sidecar 首版
- manifest 新增 `sidecar` 解析与校验：平台、SHA256、executables、healthPath、socketName。
- 安装器拆分 `frontend/` 与 `sidecar/` 两个目标：模块进 `modules/<id>`，二进制进 `/data/sidecar/<id>`，用户数据进 `/data/modules/<id>`。
- 新增 `internal/sidecar`：启动/停止、Unix/TCP 地址、token、日志、`Pid` 与状态；停用时 SIGTERM→10s→SIGKILL 进程组。
- 新增 `/api/m/{id}/__status`、`__logs` 与 `/api/m/{id}/*` 反代；未启用 403。
- 宿主启动时自动拉起已启用模块的 sidecar；SIGTERM 时先停 sidecar 再关 HTTP。
- 已实测 mihomo 2.0.1：安装、启动、`/version`、`/status`、停用、再启用全部通过。

### 调整 · 首页改为 DASHBOARD
- 移除首屏的 eyebrow、大标题、导语与独立统计区块。
- `Host Status` 面板扩展为整行，顶部显示模块总数、正在运行、运行时长，底部两列显示宿主版本、运行时、已启用模块、模块目录。
- 删除「模块仓库」标题，仅保留模块摘要与筛选/管理入口；顶栏及文档、管理页的返回入口统一改名为 `DASHBOARD`。
- 首页标题改为 `DASHBOARD · Mica Home`。

### 修复 · 模块卡片展开弹层被按钮类名污染
- 卡片「打开」按钮的 `.open` 与弹层容器的 `.stage-wrap.open` 状态类重名；弹层打开时按钮样式（深色背景 + 999px 圆角）套到全屏容器上，出现巨大黑色圆形遮罩。
- 按钮类改名为 `.mod-open`，弹层状态类保持不变；已用内置浏览器点开 Hello 与 Mihomo 卡片验证恢复正常。
- UI 设计规范补 R-13：状态类统一 `.is-*` 并绑定组件根，禁止裸 `.open`/`.active`/`.show`；自检清单增加状态类冲突检查。

### 文档 · 宿主侧 Sidecar 实现
- `/docs/modules` 新增第 10 节「宿主侧实现」：规划代码结构、安装事务、运行时监督、反代、状态接口、权限审计、实施顺序与首版边界。
- 补充术语表：sidecar 是机制/角色，`mihomo-watch` 是具体程序，supervisor 是宿主内部代码。
- `docs/SIDECAR-CONTRACT.md` 增加宿主实现指引，并明确双通道 token 与验收。

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

### 新增 · 面向 Agent 的 UI 设计规范
- 新增 `docs/MICA-UI.md`：单文件自包含的设计规范，供其他系统/Agent 直接复刻本项目界面语言；含设计定位、12 条硬性规则（R-01～R-12）、完整明暗令牌表、可直接落盘的 `mica-ui.css`（基础层 + 排版 + 布局 + 15 类组件）、排版与间距体系、动效约定、可访问性要求与 8 项交付自检。
- 该文件是 v0.1.0 设计语言的快照，令牌与组件提炼自 `web/theme.css`、`web/index.html`、`web/docs.html` 与示例模块；宿主页面本身未改动。
- 已用浅色/深色两套渲染验证全部组件，令牌值与 `web/theme.css` 逐条一致。

### 调整 · 开发文档补 Sidecar 契约
- `/docs/modules` 新增第 9 节「Sidecar 契约」：三层职责、包与目录结构、manifest 字段、管理通道 `/__host/*` 与模块通道 `/api/m/{id}/*`、生命周期、数据面、限额、权限与验收。
- `docs/SIDECAR-CONTRACT.md` 同步双通道设计：管理口仅宿主、独立 token、反代拒绝 `/__host/*`；验收补 403/401 与健康检查。
- 明确浏览器模块不能直连 7890，容器内进程才可；代理端口不发布。

### 新增 · 模块 Sidecar 契约定稿
- 新增 `docs/SIDECAR-CONTRACT.md`：明确 sidecar 是模块自带的 `mihomo-watch`，宿主只管理 watch，watch 管理 mihomo。
- 契约覆盖模块包结构、manifest 字段、宿主注入环境变量、watch 接口、生命周期与状态、数据面（7890 仅容器内回环、不发布）、安装/升级/卸载、权限确认与验收清单。
- 与旧需求文档的差异单独列出，模块方可按清单修改。

### 调整 · 明确宿主运行形态
- `/docs/modules` 第 1 节新增醒目说明：宿主运行在 Docker Linux 容器，模块前端在浏览器执行；容器内不能操作 Windows 进程、命名管道与用户目录，也不负责拉起宿主机进程。
- 第 7 节「能力边界」补「本机进程 / 文件 / 命名管道」未实现说明；README 快速开始同步。

### 新增 · 模块三态管理与自渲染卡片
- 模块状态拆成「已安装 / 已启用 / 已停用」：扫描到的目录算已安装，启用状态单独持久化（`state.json` 的 `enabled`，兼容旧 `installed`）。
- 新增 `POST /api/modules/{id}/enable|disable`、`DELETE /api/modules/{id}`；卸载会删除 `modules/<id>` 目录。旧 install/uninstall 路径保留为兼容别名。
- 新增 `frontend.card` 契约：模块导出 `renderCard(el, api)` 填充首页卡片内容区；宿主提供外壳、1.5 秒超时与失败降级。`/api/modules` 新增 `enabled` 字段，`/api/health` 新增 `enabled` 计数。
- 首页只渲染已启用模块，卡片改为「打开」按钮 + 模块自定义内容；顶栏新增「模块管理」。
- 新增 `/manage` 模块管理页：上传 zip、启用/停用、卸载删除，展示所有已安装模块。
- 开发文档补卡片契约、enable/disable/delete 接口与验收步骤；README 同步。

### 调整 · 模块文档补交付与验收
- `/docs/modules` 新增第 8 节「交付与验收」：zip 包结构、同 ID 覆盖规则、卸载语义、P0 无后端/数据库、纯 ESM、样式隔离、图标名、入口契约，以及 6 条验收步骤。
- 明确 AI 生成模块时需同时阅读「模块开发」与「主题设计」两页，并按验收步骤自测。

### 新增 · 模块 zip 包安装
- 新增 `POST /api/modules/install`：接收 multipart 字段 `file`，校验并解压 zip 到 `modules/<id>/`，写入启用状态后重新扫描。
- 安装包要求 `manifest.json` 与 `frontend/` 位于包根；同 ID 已存在返回 409，拒绝覆盖。
- 安全限制：上传 20MB、解压总量 20MB、单文件 8MB、条目 500 个；拒绝绝对路径、`..` 穿越、符号链接与盘符/ADS 路径。
- 仓库页右上角新增「上传安装」按钮，安装成功后自动刷新列表并提示。
- 开发文档补「上传安装模块包」说明与 curl 示例；README 补 API。

### 调整 · 正文字体改为 HarmonyOS Sans SC
- 新增 `web/fonts/HarmonyOS_Sans_SC_Regular.woff2` 与 `Bold.woff2`（由系统 TTF 转码，合计约 8.7MB），在 `web/theme.css` 中用 `@font-face` 自托管，并优先 `local()` 命中本机字体。
- `--mh-font-d` / `--mh-font-b` 统一为 `"HarmonyOS Sans SC"` + 中文回退栈；`--mh-font-m` 保留 IBM Plex Mono。
- 三个页面移除 Bricolage Grotesque / IBM Plex Sans 的 Google Fonts 请求，只保留 IBM Plex Mono。
- `docs/MICA-UI.md` 同步字体令牌、`@font-face`、字体接入说明与自检清单。

### 调整 · 开发文档拆分为两册
- 新增 `/docs/modules`（模块开发）与 `/docs/theme`（主题设计）两个同级页面；`/docs` 302 跳转到模块开发册，旧链接不失效。
- 主题指南从原来的第 8 条独立成册，编号重排为 01 三条原则 → 08 验证方法，不再出现「8.x」。
- 抽出 `web/docs.css`（两页共用排版样式）与 `web/docs.js`（目录滚动高亮），删掉页内联样式与上一轮的目录分组。
- 两页顶部新增「模块开发 | 主题设计」同级切换器；顶栏仍保持「模块仓库 / 开发文档」两个一级项，窄屏也能切换。
- 旧 `web/docs.html` 移入 `toberemoved/`。

### 调整 · 顶部导航同级化
- 顶部导航样式移入 `web/theme.css`（`.topbar-nav`），两页共用同一套结构与交互，当前页以加粗 + 下划线高亮。
- `/docs` 顶栏改为与首页一致的「模块仓库 / 开发文档」同级导航，移除「← 返回模块仓库」按钮与「开发文档」小标签，不再呈现子页面观感。
- `/docs` 顶栏与内容区最大宽度由 1160px 对齐到 1240px，两页切换时顶栏宽度与间距一致。

### 修复
- 模块舞台打开后永久停留在"正在加载模块前端…"：`openModule` 先取 token 再调用会自增 token 的 `cleanupStage()`，导致守卫误判提前返回。改为先失效再取 token。
- 顶栏移除无实际用途的"模块 API"链接（直接打开原始 JSON，没有可用界面）。
- 模块卡片移除悬停时白光扫过效果，保留上浮、阴影与边框高亮。

### 早前
- 确认架构方向：Go 宿主 + WASM（wazero）后端模块 + 远程 ESM 前端 + PostgreSQL schema-per-module + OCI 模块仓库，Docker 单容器分发。
