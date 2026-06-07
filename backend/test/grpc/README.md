# Pruebas gRPC directas

Smoke tests data-driven contra los endpoints gRPC del backend, ejecutados con
`grpcurl` sobre el protocolo gRPC real (HTTP/2 en texto plano, h2c). No dependen
de Apidog: las herramientas de Apidog no operan gRPC de forma headless, por lo
que el contrato se lee directamente desde los `.proto` del repositorio.

Sirven como base para validar endpoints a mano o en bucle, de forma independiente
de la suite de integración en Go (que usa testcontainers).

## Requisitos

- `grpcurl` en el `PATH` (o exportar `GRPCURL` con la ruta).
- `jq`.
- El servidor y sus dependencias en ejecución. Desde `backend/`:

  ```sh
  docker compose up -d
  DATABASE_URL='postgres://app:app@127.0.0.1:5432/academico?sslmode=disable' \
  REDIS_URL='redis://127.0.0.1:6379' \
  SESSION_TTL=30m RESET_TOKEN_TTL=15m APP_ENV=dev COOKIE_SECURE=false \
  BCRYPT_COST=10 HTTP_ADDR=:8080 \
  go run ./cmd/api
  ```

  `APP_ENV=dev` es necesario para que el endpoint de reset devuelva el token en
  la respuesta (en producción no se expone).

## Uso

```sh
./run_auth_flow.sh          # una pasada
./run_auth_flow.sh 50       # 50 iteraciones de la tabla de login
```

Variables opcionales: `GRPCURL`, `ADDR` (default `127.0.0.1:8080`), `ADMIN`
(default `admin@dev.local`). El script termina con código distinto de cero si
algún caso falla, por lo que puede integrarse en CI o en un hook.

## Qué hace `run_auth_flow.sh`

1. Verifica alcance del servidor y herramientas.
2. Fija una contraseña conocida para el admin a través del flujo de reset (el
   hash sembrado en la migración es desconocido a propósito).
3. Recorre la tabla `data/login_cases.json` y valida que cada payload devuelva
   el código gRPC esperado (`OK`, `Unauthenticated`, `InvalidArgument`).
4. Ejercita el ciclo de sesión: login con cookie, logout, rechazo de cookie
   reutilizada y rechazo de logout sin cookie.

La sesión se transporta como **metadata** gRPC: el servidor responde con
`set-cookie` y el cliente reenvía `cookie: sid=<valor>`.

## Cómo extenderlo

- **Más casos de login:** agregar objetos a `data/login_cases.json`. El valor
  `__KNOWN__` en `password` se sustituye por la contraseña sembrada.
- **Otro servicio:** duplicar el patrón en un `run_<servicio>_flow.sh` con su
  propio archivo en `data/`, apuntando al `.proto` correspondiente en
  `backend/proto`.
