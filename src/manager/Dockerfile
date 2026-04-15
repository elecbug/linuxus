FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o linuxus-manager cmd/main.go

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y tzdata && rm -rf /var/lib/apt/lists/*

RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime \
    && echo $TZ > /etc/timezone

WORKDIR /app

COPY --from=builder /app/linuxus-manager /app/linuxus-manager

EXPOSE 8080

CMD ["/app/linuxus-manager"]