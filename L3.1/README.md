# L3.1 Notification Service

Сервис для отправки уведомлений с использованием RabbitMQ и Telegram Bot API.  
Поддерживает повторные попытки (retry), задержку по TTL, error queue и отмену уведомлений.

---

## Запуск RabbitMQ в Docker

Для  запуска RabbitMQ  используйте:

```bash
docker compose up -d
```
## Запуск сервиса
```bash
go run ./cmd/app
```
