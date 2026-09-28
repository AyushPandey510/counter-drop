# Full production image: builds the React PWA and serves it from the Go API.

FROM node:20-alpine AS web-build
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.27.1-alpine AS api-build
WORKDIR /src/api
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/counter-drop-api ./cmd/api
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/cdadmin ./cmd/cdadmin

FROM alpine:3.22
RUN adduser -D -H counterdrop \
  && mkdir -p /app/web /app/migrations /data/files \
  && chown -R counterdrop:counterdrop /app /data
WORKDIR /app
COPY --from=api-build /src/api/migrations ./migrations
COPY --from=api-build /out/counter-drop-api /usr/local/bin/counter-drop-api
COPY --from=api-build /out/cdadmin /usr/local/bin/cdadmin
COPY --from=web-build /src/web/dist ./web
ENV CD_API_ADDR=:8080 \
    CD_WEB_DIR=/app/web \
    CD_MIGRATIONS_DIR=/app/migrations \
    CD_STORAGE_DIR=/data/files
USER counterdrop
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD wget -qO- http://127.0.0.1:8080/ready >/dev/null || exit 1
CMD ["counter-drop-api"]
