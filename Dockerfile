# STAGE build
FROM golang:1.18.2 AS build

MAINTAINER Dengzhehang "dengzhehang@outlook.com"

ENV GO111MODULE=on \
    CGO_ENABLED="0" \
    GOOS=linux \
    GOARCH=amd64 \
    GOPROXY="https://goproxy.cn,direct"

WORKDIR /app

COPY . .

RUN mkdir conf

RUN go build -o vsphere-facade .

# STAGE deploy
FROM scratch

WORKDIR /

COPY --from=build /app /app

EXPOSE 8829

CMD  ["/app/vsphere-facade", "-config=/app/conf"]