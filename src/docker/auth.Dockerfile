FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends tzdata ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY linuxusctl /app/linuxusctl

EXPOSE 8080
CMD ["/app/linuxusctl", "serve-auth"]
