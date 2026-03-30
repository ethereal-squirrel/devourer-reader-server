FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o devourer-server ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /build/devourer-server .
COPY migrations/ ./migrations/
COPY plugins/     ./plugins/
VOLUME ["/app/data", "/app/assets"]
EXPOSE 9024
ENV DATABASE_PATH=/app/data/devourer.db \
    ASSETS_PATH=/app/assets \
    MIGRATIONS_DIR=/app/migrations \
    PLUGINS_PATH=/app/plugins \
    PORT=9024
CMD ["./devourer-server", "serve"]
