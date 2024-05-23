FROM golang:1.21-alpine3.18 AS builder

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /app

ADD go.mod go.sum /app/
RUN go mod download

ADD /pkg /app/pkg
ADD /cmd /app/cmd

RUN go build -o main cmd/main/main.go

FROM alpine:3.18 AS final

WORKDIR /app
COPY --from=builder /app/main /app/main

CMD ["/app/main"]