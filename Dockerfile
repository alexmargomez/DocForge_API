FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/docforge-server ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /out/docforge-server /app/docforge-server
EXPOSE 8086
ENTRYPOINT ["/app/docforge-server"]
