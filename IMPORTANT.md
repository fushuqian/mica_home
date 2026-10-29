# IMPORTANT

## 项目
- 名称：Mica Home
- 仓库：https://github.com/fushuqian/mica_home
- 镜像：`mica-home:0.1.0`，容器名 `mica-home`，端口 8080
- Docker Hub：`kenzong/mica-home`（`0.1.0`、`latest`），账号 `kenzong`，凭据保存在 Docker Desktop credsStore，不落仓库

## 配置
| 变量 | 默认 | 说明 |
|---|---|---|
| `MICA_ADDR` | `:8080` | 监听地址 |
| `MICA_MODULES_DIR` | `modules` | 模块目录（compose 挂载 `./modules`） |
| `MICA_DATA_DIR` | `data` | 运行数据目录，安装状态存 `state.json` |
- Docker 卷：`mica-home_mica-data` → `/data`
- 首次启动（无 `state.json`）自动启用示例模块 `hello`

## 密钥
- 暂无。后续需记录：OCI 仓库凭证、模块签名密钥（cosign / ed25519）、PostgreSQL 连接串。

## 环境注意
- 本机 Docker Hub 直连超时，构建改用本机已缓存的基础镜像 tag（`golang:1.24-alpine`、`alpine:3.19`）；如换机器拉取失败，用镜像加速器 `docker.1ms.run/library/...`。
- 本机未安装 Go，所有编译在容器内完成。
