FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o hls-service ./cmd

FROM alpine:3.18

RUN apk add --no-cache ffmpeg ca-certificates

WORKDIR /app

COPY --from=builder /app/hls-service .

RUN mkdir -p /tmp/hls && chmod 755 /tmp/hls

ENV PORT=8080
ENV HLS_WORK_DIR=/tmp/hls
ENV FFMPEG_PATH=/usr/bin/ffmpeg

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./hls-service"]
