FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o storage-service ./cmd

FROM alpine:3.18

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/storage-service .

RUN mkdir -p /storage/hot /storage/warm /storage/cold && \
    chmod 755 /storage/hot /storage/warm /storage/cold

ENV PORT=8082
ENV HOT_TIER_PATH=/storage/hot
ENV WARM_TIER_PATH=/storage/warm
ENV COLD_TIER_PATH=/storage/cold

EXPOSE 8082

HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8082/health || exit 1

CMD ["./storage-service"]
