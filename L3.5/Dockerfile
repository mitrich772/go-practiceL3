# ===== Этап сборки =====
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Копируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем сервер
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# Собираем утилиту миграций
RUN CGO_ENABLED=0 GOOS=linux go build -o /migrate ./cmd/migrate

# ===== Финальный образ =====
FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Копируем бинарники
COPY --from=builder /server /app/server
COPY --from=builder /migrate /app/migrate

# Копируем конфиг, миграции и веб-файлы
COPY config/ /app/config/
COPY migrations/ /app/migrations/
COPY web/ /app/web/

EXPOSE 8080

CMD ["/app/server"]
