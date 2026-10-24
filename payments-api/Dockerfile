FROM golang:alpine AS builder

WORKDIR /app

# Download dependencies first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and migrations
COPY . .

# Compile static binary
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/payments-api ./cmd/api

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/payments-api /app/payments-api

ENV PORT=8080
EXPOSE 8080

USER guest:users

ENTRYPOINT ["/app/payments-api"]
