FROM golang:1.15-alpine AS builder
COPY . /dist
WORKDIR /dist
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
  apk add --no-cache ca-certificates tzdata
RUN go env -w GOPROXY=https://goproxy.cn,direct && \
 go mod tidy -v

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app main.go

FROM alpine

COPY --from=builder /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
COPY --from=builder /app /app

ENTRYPOINT [ "/app" ]