# ============================================================
# 多阶段构建：gitlink-cli 子赛题四网页终端
# ============================================================
# 阶段1 builder —— Go 静态编译
# ============================================================
FROM golang:1.26-alpine AS builder

ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /src

# 先拷依赖清单，利用 Docker 层缓存
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# modernc.org/sqlite 是 pure-Go，CGO_ENABLED=0 即可编译纯静态二进制
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/gitlink-cli .

# ============================================================
# 阶段2 runtime —— Python + Go 二进制
# ============================================================
FROM python:3.12-slim

# Go CLI 放入 PATH
COPY --from=builder /out/gitlink-cli /usr/local/bin/gitlink-cli

# 科研算法层：先拷 requirements.txt 安装依赖（利用层缓存），再拷源码
# pip 先升级自身，再用清华镜像（带 retry + trusted-host 防证书/网络抖动）
COPY scripts/research/requirements.txt /app/scripts/research/requirements.txt
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir --default-timeout=300 --retries 5 \
    -i https://pypi.tuna.tsinghua.edu.cn/simple \
    --trusted-host pypi.tuna.tsinghua.edu.cn \
    -r /app/scripts/research/requirements.txt
COPY scripts/research/ /app/scripts/research/

WORKDIR /app

# 子赛题四网页终端 HTTP 服务
EXPOSE 8080
ENTRYPOINT ["gitlink-cli", "server", "--port", "8080", "--research-dir", "/app/scripts/research", "--work-dir", "/app/research-output"]
