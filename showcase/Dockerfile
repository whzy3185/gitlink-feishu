# Stage 1: Build
FROM golang:1.26-alpine AS builder

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build gitlink-cli binary
RUN CGO_ENABLED=0 GOOS=linux go build -o gitlink-cli .

# Build showcase server binary
RUN CGO_ENABLED=0 GOOS=linux go build -o showcase-server ./showcase/

# Stage 2: Runtime
FROM alpine:latest

RUN apk add --no-cache ca-certificates git

WORKDIR /app

COPY --from=builder /build/gitlink-cli .
COPY --from=builder /build/showcase-server .

EXPOSE 9090

CMD ["./showcase-server"]
