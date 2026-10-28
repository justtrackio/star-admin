FROM oven/bun:1.4.3 AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/bun.lock ./
RUN bun install --frozen-lockfile
COPY frontend/ ./
RUN bun run build

FROM golang:1.27.2 AS backend-builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend-builder /app/frontend/dist ./public
RUN CGO_ENABLED=0 GOOS=linux go build -o star-admin .

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=backend-builder /app/star-admin .
COPY --from=backend-builder /app/build/migrations ./build/migrations
EXPOSE 8082
ENTRYPOINT ["./star-admin"]
