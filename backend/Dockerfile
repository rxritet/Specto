# Этап сборки backend
FROM golang:1.26.0-alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o specto ./cmd/specto

# Финальный образ
FROM alpine:3.21
WORKDIR /root/
COPY --from=backend-builder /app/specto .
# Копируем миграции, если они нужны в рантайме
COPY --from=backend-builder /app/internal/database/migrations ./internal/database/migrations
EXPOSE 8080
CMD ["./specto", "server"]
