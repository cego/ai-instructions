FROM golang:1.25.0-alpine3.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /ai-instructions ./cmd/ai-instructions

FROM alpine:3.22.1
RUN apk add --no-cache git ca-certificates
COPY --from=builder /ai-instructions /usr/local/bin/ai-instructions
ENTRYPOINT ["ai-instructions"]
