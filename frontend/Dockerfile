# docker build -f frontend/Dockerfile -t academico/web:dev .   (contexto = raíz del repo)
FROM oven/bun:1-alpine AS build
WORKDIR /app/frontend

COPY frontend/package.json frontend/bun.lock ./
RUN bun install --frozen-lockfile

COPY backend/proto /app/backend/proto
COPY frontend/ ./
RUN bun run build

FROM nginxinc/nginx-unprivileged:alpine AS runtime
COPY --from=build /app/frontend/dist /usr/share/nginx/html
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 8080
