# Backend — API

API del sistema de gestión académica en Go. Monolito modular organizado por dominio, sobre **Connect** (atiende gRPC, gRPC-Web y HTTP/JSON en un mismo puerto).

## Stack

| Tema | Tecnología |
|------|-----------|
| Lenguaje | Go |
| RPC | Connect-go (`protoc-gen-go`, codegen con `buf`) |
| Persistencia | `pgx` + `sqlc` sobre PostgreSQL |
| Migraciones | `golang-migrate` |
| Cache / sesiones | Redis |
| Tests | `go test` + `testify` |
| Lint | `golangci-lint` |

## Estructura

```
backend/
├── proto/          # contratos .proto por dominio (compartidos con el frontend)
├── cmd/api/        # entrypoint
├── internal/
│   ├── enrollments/ grades/ reports/ users/ auth/   # handler → service → repository
│   └── platform/   # db, cache, config, server, interceptors
└── migrations/     # migraciones SQL
```

Cada dominio expone una interface de repositorio (del lado del consumidor) que implementa el adaptador de Postgres. El dominio no conoce ni HTTP ni SQL.

## Uso

Requisitos: Go, [`buf`](https://buf.build), Docker (para Postgres y Redis locales).

```bash
# generar código desde los .proto
buf generate

# aplicar migraciones
migrate -path migrations -database "$DATABASE_URL" up

# correr la API
go run ./cmd/api

# tests y lint
go test ./...
golangci-lint run
```

## Configuración

Variables de entorno (en local vía `.env`; en el cluster vía ConfigMap/Secret):

| Variable | Descripción |
|----------|-------------|
| `DATABASE_URL` | Cadena de conexión a PostgreSQL. |
| `REDIS_URL` | Conexión a Redis (sesiones + cache). |
| `PORT` | Puerto de la API (por defecto 8080). |
