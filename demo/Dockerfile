# demo/Dockerfile — GitLink CLI 演示网页后端
# 多阶段：Go 编译 Linux 二进制 → Python 运行时跑 server.py
# 构建上下文 = 仓库根（gitlink-cli/）：  docker build -f demo/Dockerfile -t gitlink-cli-demo .

# ---------- Stage 1: 编译 gitlink-cli（Linux） ----------
FROM golang:1.26-alpine AS builder
ENV GOPROXY=https://goproxy.cn,direct
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o /out/gitlink-cli .

# ---------- Stage 2: Python 运行时 ----------
FROM python:3.11-alpine
RUN apk add --no-cache git ca-certificates
WORKDIR /app
# 二进制
COPY --from=builder /out/gitlink-cli /usr/local/bin/gitlink-cli
# 演示网页 + Skills（server.py 要读 SKILL.md）
COPY demo/ /app/demo/
COPY skills/ /app/skills/
ENV HOST=0.0.0.0
ENV PORT=8000
ENV GITLINK_BIN=/usr/local/bin/gitlink-cli
EXPOSE 8000
WORKDIR /app/demo/web
CMD ["python3", "server.py"]
