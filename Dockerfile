# ---- 构建阶段：容器内编译，本地无需安装 Go ----
FROM golang:1.24-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod ./
COPY main.go ./
COPY internal ./internal
COPY web ./web
RUN go build -trimpath -ldflags="-s -w" -o /out/mica-home .

# ---- 运行阶段：glibc 运行层，供含 Python / C 扩展的 sidecar 使用 ----
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates tzdata procps \
    && rm -rf /var/lib/apt/lists/*
RUN useradd -u 10001 -M -s /usr/sbin/nologin mica
WORKDIR /app
COPY --from=build /out/mica-home /app/mica-home
COPY modules /app/modules
COPY docker/entrypoint.sh /app/entrypoint.sh
RUN chmod 755 /app/entrypoint.sh \
    && mkdir -p /data && chown -R mica:mica /data /app
# 以 root 进入入口脚本，先修正挂载目录属主（命名卷默认 root 所有），
# 再用 setpriv 降权到 mica(10001) 运行；MICA_NO_CHOWN=1 可跳过修正。
ENTRYPOINT ["/app/entrypoint.sh"]
ENV MICA_ADDR=:8080 \
    MICA_MODULES_DIR=/app/modules \
    MICA_DATA_DIR=/data
EXPOSE 8080
VOLUME ["/data"]
