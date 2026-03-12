FROM golang:1.24.11-bookworm AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/notificationd ./cmd/notificationd

FROM alpine:3.19
RUN addgroup -S app && adduser -S app -G app \
    && apk add --no-cache ca-certificates wget

WORKDIR /app
COPY --from=builder /out/notificationd /usr/local/bin/notificationd

EXPOSE 8080
USER app
HEALTHCHECK --interval=10s --timeout=3s --retries=3 CMD wget --spider -q http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/notificationd"]
