FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /app/api ./cmd/api

FROM alpine:3.21

RUN addgroup -S app && adduser -S app -G app
USER app

COPY --from=builder /app/api /app/api

CMD ["/app/api"]
