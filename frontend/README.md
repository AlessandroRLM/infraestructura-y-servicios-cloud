# Frontend — SPA

SPA del sistema de gestión académica en React. Consume la API Connect del backend con tipos generados desde los mismos `.proto`. Se compila estática y la sirve Nginx.

## Stack

| Tema | Tecnología |
|------|-----------|
| Runtime / package manager | Bun |
| Build | Vite |
| Lenguaje | TypeScript |
| Cliente API | connect-web (`@connectrpc/connect-web`) |
| Estado servidor | TanStack Query + connect-query |
| Routing | TanStack Router (route tree generado, tipado) |
| Schemas | Zod |
| Estilos | CSS Modules |
| Lint / formato | Biome |
| Tests | Vitest + React Testing Library |

## Estructura

```
frontend/src/
├── features/       # enrollments, grades, reports, users, auth
│   └── <feature>/  # componentes + hooks (connect-query) del dominio
├── components/ui/  # primitivos reutilizables
├── lib/            # cliente connect, router, config
└── gen/            # código generado de los .proto (buf)
```

Organización feature-first, espejo de los dominios del backend. El patrón container/presentational se aplica donde separa lógica de presentación.

## API y transporte

El cliente usa el protocolo **Connect** (`createConnectTransport`). En producción Nginx sirve la SPA y hace reverse-proxy de las rutas RPC hacia la API, por lo que frontend y backend comparten origen: la sesión viaja en una cookie `HttpOnly` `SameSite=Lax` (first-party, inaccesible desde JavaScript). En desarrollo, el proxy de Vite cumple el mismo rol que Nginx.

El formato de serialización es condicional por entorno (`useBinaryFormat: import.meta.env.PROD`): JSON en desarrollo para mantener el panel de red legible, y protobuf binario en producción por eficiencia de payload. La elección del protocolo (Connect frente a gRPC nativo) y su impacto en el balanceo de carga se documentan en [`docs/arquitectura`](../docs/arquitectura/README.md#16-decisiones-y-alternativas).

## Uso

Requisitos: [Bun](https://bun.sh), [`buf`](https://buf.build).

```bash
# instalación determinista; Bun no corre lifecycle scripts salvo trustedDependencies
bun install --frozen-lockfile

# generar el cliente type-safe desde los .proto
buf generate

# desarrollo
bun run dev        # vite

# build de producción (estático, lo sirve Nginx)
bun run build      # vite build

# tests
bun run test       # vitest

# lint + formato
bunx biome check .
```

## Seguridad de dependencias

Bun no ejecuta los `postinstall` (ni otros lifecycle scripts) de las dependencias por defecto — el vector más común de supply-chain en npm. Solo los paquetes listados en `trustedDependencies` del `package.json` pueden correr scripts. El lockfile se fija con `--frozen-lockfile`.

## Configuración

| Variable | Descripción |
|----------|-------------|
| `VITE_API_URL` | URL base de la API Connect. |

La sesión viaja en una cookie `httpOnly`; el cliente no maneja el token. El gating de UI por rol es solo presentación: el control real es del backend.
