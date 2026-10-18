# Multi-stage build: Vue client → Go binary → minimal runtime image.
# docker build -t messario:latest .

FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/internal/web/dist ./internal/web/dist
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/messario ./cmd/messario

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 messario
WORKDIR /app
COPY --from=build /out/messario /app/messario
RUN mkdir -p /app/data/files && chown -R messario:messario /app/data
USER messario
ENV MESSARIO_ADDR=:8080 MESSARIO_FILES_DIR=/app/data/files
EXPOSE 8080
VOLUME ["/app/data"]
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/app/messario"]
CMD ["serve"]
