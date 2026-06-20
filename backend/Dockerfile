# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Cache dependencies to avoid downloading them on every build
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
  go mod download

COPY . .

# Cache compiler artifacts to only recompile files that changed
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s" -o /app/api ./cmd/api


# Development stage (Install specific tools for development)
FROM golang:1.26-alpine AS development
WORKDIR /app
RUN go install github.com/air-verse/air@1.65.3
EXPOSE 8080
CMD ["air", "-c", ".air.toml"]

# Production stage (Minimal and secure distroless image)
FROM gcr.io/distroless/static-debian12:nonroot@sha256:d093aa3e30dbadd3efe1310db061a14da60299baff8450a17fe0ccc514a16639

WORKDIR /

COPY --from=builder /app/api /app/api

CMD ["/app/api"]
