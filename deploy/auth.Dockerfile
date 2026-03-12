FROM golang:1.24.11-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/authd ./cmd/authd
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM alpine:3.19
RUN addgroup -S app && adduser -S app -G app \
    && apk add --no-cache ca-certificates wget
WORKDIR /app
COPY --from=builder /out/authd /app/authd
COPY --from=builder /out/migrate /app/migrate
COPY --from=builder /app/db /app/db

EXPOSE 8080
USER app
HEALTHCHECK --interval=10s --timeout=3s --retries=3 CMD wget --spider -q http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/app/authd"]
