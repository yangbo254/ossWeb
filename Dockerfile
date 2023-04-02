# 打包依赖阶段使用golang作为基础镜像
FROM golang:1.20 as builder

# 启用go module
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct

WORKDIR /app

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build .

FROM alpine

WORKDIR /app
COPY --from=builder /app/config.json.sample .
ENV GIN_MODE=release

ENTRYPOINT ["./ossweb"]