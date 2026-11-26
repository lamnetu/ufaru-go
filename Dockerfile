FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -o /parser ./cmd/parser

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -H -u 10001 app
WORKDIR /app
COPY --from=builder /parser .
USER app
EXPOSE 8081
ENTRYPOINT ["./parser"]