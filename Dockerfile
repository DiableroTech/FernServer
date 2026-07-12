# FernServer — production image (multi-stage).
# Stage 1 compiles the Go binary; stage 2 ships only that binary (~20MB vs ~1GB).
# Build:  docker build -t fern-server .
# Run alone (needs DATABASE_URL etc.): docker run --rm -p 8080:8080 --env-file .env fern-server

# --- Build ---
FROM golang:1.25-bookworm AS builder

WORKDIR /src

# Cache module downloads when only source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static binary — no CGO, runs in minimal Linux images.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /fern ./cmd/fern

# --- Run ---
FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /fern .

# Non-root — good habit for prod; app only binds :8080.
RUN useradd -r -u 10001 -g nogroup fern \
    && chown fern:nogroup /app/fern
USER fern

EXPOSE 8080

ENTRYPOINT ["./fern"]
