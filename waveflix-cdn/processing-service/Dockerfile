FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o processing-service ./cmd

FROM alpine:3.18

RUN apk add --no-cache ffmpeg ca-certificates

WORKDIR /app

COPY --from=builder /app/processing-service .

RUN mkdir -p /tmp/processing && chmod 755 /tmp/processing

ENV PORT=8081
ENV WORK_DIR=/tmp/processing
ENV FFMPEG_PATH=/usr/bin/ffmpeg
ENV MAX_CONCURRENT_JOBS=5

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

CMD ["./processing-service"]
