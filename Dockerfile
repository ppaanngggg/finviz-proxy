FROM golang:1.21-alpine3.18 AS builder

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /app

ADD go.mod go.sum /app/
RUN go mod download

ADD *.go /app/
RUN go build -o main .

FROM alpine:3.18 AS final

WORKDIR /app
COPY --from=builder /app/main /app/main

CMD ["/app/main"]