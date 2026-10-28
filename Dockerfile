# ---- 构建阶段：容器内编译，本地无需安装 Go ----
FROM golang:1.24-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod ./
COPY main.go ./
COPY internal ./internal
COPY web ./web
RUN go build -trimpath -ldflags="-s -w" -o /out/mica-home .

# ---- 运行阶段 ----
FROM alpine:3.19
RUN adduser -D -u 10001 mica
WORKDIR /app
COPY --from=build /out/mica-home /app/mica-home
COPY modules /app/modules
RUN mkdir -p /data && chown -R mica:mica /data /app
USER mica
ENV MICA_ADDR=:8080 \
    MICA_MODULES_DIR=/app/modules \
    MICA_DATA_DIR=/data
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/app/mica-home"]
