# L3.7: WarehouseControl — CRUD склада с историей и ролями

Сервис учёта товаров на складе: CRUD инвентаря, JWT-авторизация, разграничение прав по ролям и аудит изменений через PostgreSQL-триггеры.

## Архитектура

* **HTTP API (chi)**: CRUD по товарам, выдача JWT и просмотр истории.
* **PostgreSQL** через `github.com/wb-go/wbf/dbpg`: товары, демо-пользователи и таблица аудита.
* **DB triggers**: `trg_items_audit` пишет INSERT/UPDATE/DELETE в `item_history`; actor/role берутся из `set_config('app.actor', ...)` внутри транзакции.
* **JWT**: роль (`admin`, `manager`, `viewer`) передаётся в токене и проверяется middleware на каждом API-запросе.
* **Веб-интерфейс**: выбор роли, таблица товаров, добавление/редактирование/удаление по правам, история изменений для admin.

```
L3.7/
├── cmd/
│   ├── server/main.go       # entrypoint HTTP-сервиса
│   └── migrate/main.go      # CLI-мигратор (-action up|down|step -n N)
├── config/
│   ├── local.yaml
│   └── docker.yaml
├── internal/
│   ├── config/              # cleanenv-конфиг
│   ├── handlers/            # один эндпоинт = один файл + auth middleware
│   ├── middleware/logger/   # access-logger на slog
│   ├── model/               # доменные сущности и роли
│   └── repo/postgres/       # реализация Repo на wbf/dbpg
├── migrations/0001_init.{up,down}.sql
└── web/static/              # HTML + JS + CSS
```

## Роли

* `admin` — полный доступ: просмотр, создание, редактирование, удаление, история.
* `manager` — просмотр, создание и редактирование.
* `viewer` — только просмотр.

## API

### Авторизация

**POST** `/login`
```json
{
  "username": "ivan",
  "role": "admin"
}
```

Ответ:
```json
{
  "token": "jwt...",
  "user": {
    "username": "ivan",
    "role": "admin"
  }
}
```

Дальше API вызывается с заголовком:
```
Authorization: Bearer <token>
```

### CRUD

* **POST** `/items` — создать товар (`admin`, `manager`).
  ```json
  {
    "name": "Кабель USB-C",
    "sku": "USB-C-001",
    "quantity": 120,
    "location": "A-01",
    "description": "1 метр"
  }
  ```

* **GET** `/items?search=&sort=&order=&limit=&offset=` — список товаров (`admin`, `manager`, `viewer`).
  Сортировка whitelisted: `updated_at` (default) | `id` | `name` | `sku` | `quantity`.

* **GET** `/items/{id}` — получить товар (`admin`, `manager`, `viewer`).

* **PUT** `/items/{id}` — обновить товар (`admin`, `manager`).

* **DELETE** `/items/{id}` — удалить товар (`admin`).

### История

* **GET** `/items/{id}/history` — история изменений товара (`admin`).

История создаётся именно триггером PostgreSQL, а не приложением:

```sql
CREATE TRIGGER trg_items_audit
AFTER INSERT OR UPDATE OR DELETE ON items
FOR EACH ROW EXECUTE FUNCTION audit_item_changes();
```

## Запуск

### Docker Compose

```bash
cd L3.7
docker compose up --build
```

Поднимет:
1. **PostgreSQL 17** (порт `5437` снаружи).
2. **Миграции** (`golang-migrate`, источник `file://migrations`).
3. **Сервер** на `:8080`.

UI: <http://localhost:8080>.

### Локальный запуск (Postgres в Docker + Go нативно)

```bash
cd L3.7

# 1. БД
docker compose up postgres -d

# 2. Миграции
go run ./cmd/migrate -action up

# 3. Сервер
go run ./cmd/server
```

После запуска сервер слушает адрес из `config/local.yaml` (по умолчанию `:8080`).
