# Build stage
FROM oven/bun:1-alpine AS build
WORKDIR /app/frontend

COPY frontend/package.json frontend/bun.lock ./
RUN --mount=type=cache,target=/root/.bun/install/cache \
  bun install --frozen-lockfile

COPY backend/proto /app/backend/proto
COPY frontend/ ./
RUN bun run build

FROM oven/bun:1-alpine AS development

RUN mkdir -p /app/frontend && chown -R bun:bun /app
WORKDIR /app/frontend
USER bun
COPY --chown=bun:bun frontend/package.json frontend/bun.lock ./
RUN --mount=type=cache,target=/home/bun/.bun/install/cache,uid=1000,gid=1000 \
  bun install --frozen-lockfile
EXPOSE 5173
CMD ["bun", "run", "dev"]

# --- production (final stage = default build target) ---
FROM nginxinc/nginx-unprivileged:alpine AS runtime
COPY --from=build /app/frontend/dist /usr/share/nginx/html
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 8080
