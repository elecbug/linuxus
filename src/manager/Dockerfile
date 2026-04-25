FROM golang:1.25 AS builder

WORKDIR /app

COPY pkg ./pkg
COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o linuxus-manager cmd/main.go

FROM ubuntu:22.04

WORKDIR /app

COPY --from=builder /app/linuxus-manager /app/linuxus-manager

CMD ["/app/linuxus-manager"]