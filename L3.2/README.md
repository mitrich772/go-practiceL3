# URL Shortener + Analytics

Монорепозиторий с двумя Go-сервисами:

- **shortener** — сервис сокращения ссылок
- **analytics** — сервис сбора и агрегации статистики переходов

Дополнительно используются:
- **PostgreSQL** — хранение ссылок и аналитики
- **Redis** — кэширование

## Запуск

### PostgreSQL
---
```bash
cd docker/postgres
docker compose up -d
```
### Redis
```bash
cd docker/redis
docker compose up -d
```
### Миграции базы данных
---
Миграции запускаются отдельно для каждого сервиса.

Shortener — миграции
```bash
cd services/shortener
go run .\cmd\migrate\migrations.go -action up
```
Analytics — миграции
```bash
cd services/analytics
go run .\cmd\migrate\migrations.go -action up
```
### Запуск сервисов
---
Запуск shortener
```bash
cd services/shortener
go run .\cmd\main.go
```

Запуск analytics
```bash
cd services/analytics
go run .\cmd\main.go
```