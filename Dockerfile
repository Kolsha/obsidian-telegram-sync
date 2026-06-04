FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /bin/obsidian-telegram-sync ./cmd/obsidian-telegram-sync/

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/obsidian-telegram-sync /usr/local/bin/

WORKDIR /app
VOLUME ["/app/vault"]

ENV OTS_CONFIG_PATH="/app/vault/config.yaml"

ENTRYPOINT ["obsidian-telegram-sync"]
