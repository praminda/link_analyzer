FROM golang:1.26.2-alpine AS builder
WORKDIR /src

# Cache dependencies.
COPY go.mod ./
RUN go mod download

# Copy source and build.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/link-analyzer ./cmd

FROM alpine:3.21
WORKDIR /app
RUN addgroup -S app && adduser -S app -G app

COPY --from=builder /out/link-analyzer /app/link-analyzer

# SQLite job store: writable by non-root user; mount a volume here to persist across restarts.
RUN mkdir -p /app/data && chown -R app:app /app
ENV JOB_DB_PATH=/app/data/jobs.sqlite
VOLUME ["/app/data"]

EXPOSE 8080
USER app
ENTRYPOINT ["/app/link-analyzer"]
