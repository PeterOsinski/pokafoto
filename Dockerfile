# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS go-builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -ldflags="-s -w" -o /app/bin/drive ./cmd/drive

FROM node:22-alpine AS web-builder

WORKDIR /app
COPY web/package*.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci
COPY web/ .
RUN --mount=type=cache,target=/root/.npm \
    npm run build

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata ffmpeg exiftool

COPY --from=go-builder /app/bin/drive /usr/local/bin/drive
COPY --from=web-builder /app/dist /app/web/dist

ENV DRIVE_STORAGE_PATH=/data
ENV DRIVE_DB_PATH=/data/drive.db

RUN mkdir -p /data /tmp/kilo

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

ENTRYPOINT ["/usr/local/bin/drive"]
