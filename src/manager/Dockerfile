FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

RUN mkdir -p pkg

COPY cmd ./cmd
COPY internal ./internal
COPY pkg/* ./pkg

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o linuxus-manager cmd/main.go

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/linuxus-manager /app/linuxus-manager

CMD ["/app/linuxus-manager"]