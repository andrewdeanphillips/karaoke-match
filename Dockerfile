# syntax=docker/dockerfile:1

FROM node:24-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
# frontend/.env is gitignored and excluded from the build context, so without
# this the build inlines `undefined` into every API call. Empty string makes
# them relative paths, correct for serving frontend and API from one origin.
ENV VITE_API_BASE_URL=""
RUN npm run build

FROM golang:1.26-alpine AS backend
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# Overwrite the committed placeholder with the real build so cmd/api/web.go's
# //go:embed directive bundles the actual frontend, not the placeholder.
COPY --from=frontend /src/frontend/dist/ ./cmd/api/web/dist/
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

FROM alpine:3
RUN apk add --no-cache ca-certificates
COPY --from=backend /out/api /usr/local/bin/api
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
