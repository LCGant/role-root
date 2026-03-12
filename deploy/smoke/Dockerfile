FROM golang:1.24.11-alpine AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/smoke ./main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/smoke /app/smoke
ENTRYPOINT ["/app/smoke"]
