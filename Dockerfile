# STAGE build
FROM golang:1.16.3 AS build

MAINTAINER Dengzhehang "darrendeng@yuninfy.com"

ENV GO111MODULE=on \
    CGO_ENABLED="0" \
    GOOS=linux \
    GOARCH=amd64 \
    GOPROXY="https://goproxy.cn,direct"

WORKDIR /app

COPY . .

RUN mkdir conf

RUN go build -o vsphere_api .

# STAGE deploy
FROM scratch

WORKDIR /

COPY --from=build /app /app

EXPOSE 8829

CMD  ["/app/vsphere_api", "-config=/app/conf"]