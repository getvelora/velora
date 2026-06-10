FROM node:26-alpine AS web
WORKDIR /src/apps/web
COPY apps/web/package*.json ./
RUN npm ci
COPY apps/web ./
RUN npm run build

FROM golang:1.26-alpine AS server
WORKDIR /src/apps/server
COPY apps/server/go.mod apps/server/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY apps/server ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 go build -o /out/velora ./cmd/velora

FROM alpine:3.23 AS server-runtime
RUN apk add --no-cache ffmpeg ca-certificates
WORKDIR /app
COPY --from=server /out/velora /app/velora
EXPOSE 8080
ENV PORT=8080
ENV WEB_DIST_DIR=/app/web
ENV VELORA_CONFIG_DIR=/config
ENV VELORA_CACHE_DIR=/cache
ENV VELORA_MEDIA_DIR=/media
ENV VELORA_DATABASE_DRIVER=sqlite
ENV VELORA_DATABASE_URL=/config/velora.db
CMD ["/app/velora"]

FROM server-runtime AS release
COPY --from=web /src/apps/web/dist /app/web
