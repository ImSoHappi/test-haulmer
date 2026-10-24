FROM golang:alpine AS builder

WORKDIR /app

# Download dependencies first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Compile static binary
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/acquirer-mock ./cmd/mock

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/acquirer-mock /app/acquirer-mock

ENV HTTP_PORT=8081
EXPOSE 8081

USER guest:users

ENTRYPOINT ["/app/acquirer-mock"]
