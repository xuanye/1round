FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY apps/server/go.mod apps/server/go.sum ./
RUN go mod download

COPY apps/server/ ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/oneround-server ./cmd/oneround-server

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /out/oneround-server /app/oneround-server
COPY apps/server/config.example.yaml /app/config.example.yaml
COPY apps/server/migrations /app/migrations

RUN mkdir -p /app/data

EXPOSE 8080

ENTRYPOINT ["/app/oneround-server"]
