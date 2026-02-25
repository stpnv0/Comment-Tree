FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/commenttree ./cmd/commenttree

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/commenttree .
COPY --from=builder /app/config/config.yaml ./config/config.yaml
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/web ./web

EXPOSE 8080

ENTRYPOINT ["./commenttree"]