# L3.6: SalesTracker — CRUD с аналитикой и агрегированием

Сервис учёта финансовых записей (доходы / расходы) с CRUD-операциями и агрегированной аналитикой за период строго по ТЗ: сумма, среднее, количество, медиана и 90-й перцентиль. Дополнительно — экспорт в CSV.

## Архитектура

* **HTTP API (chi)**: CRUD по записям + агрегированный эндпоинт `/analytics` и CSV-экспорт.
* **PostgreSQL** через `github.com/wb-go/wbf/dbpg` (та же библиотека, что в L3.5): единственное хранилище. Агрегаты считаются на стороне БД (`SUM/AVG/COUNT` и `percentile_cont` для медианы и P90).
* **slog**: структурное логирование, кастомный access-middleware (RequestID → access-logger → Recoverer).
* **Веб-интерфейс**: одностраничное приложение на HTML + vanilla JS — форма создания, таблица записей с фильтрами и пагинацией, сводка по периоду, экспорт CSV.

```
L3.6/
├── cmd/
│   ├── server/main.go       # entrypoint HTTP-сервиса
│   └── migrate/main.go      # CLI-мигратор (-action up|down|step -n N)
├── config/
│   ├── local.yaml
│   └── docker.yaml
├── internal/
│   ├── config/              # cleanenv-конфиг
│   ├── handlers/            # один эндпоинт = один файл (verb-er интерфейсы)
│   ├── middleware/logger/   # access-logger на slog
│   ├── model/               # доменные сущности и фильтры
│   └── repo/postgres/       # реализация Repo на wbf/dbpg (db.Master.{Query|Exec}Context)
├── migrations/0001_init.{up,down}.sql
└── web/static/              # HTML + JS + CSS
```

## API

### CRUD

* **POST** `/items` — создать запись.
  ```json
  {
    "type":        "income",        // "income" | "expense"
    "amount":      1500.00,
    "category":    "salary",
    "note":        "Аванс",
    "occurred_at": "2026-05-20T10:00:00Z"
  }
  ```
  Валидация: `type ∈ {income, expense}`, `amount >= 0`, `category` обязательна и ≤ 64 символов.
  Дата — RFC3339 либо `YYYY-MM-DD`; пустая = `now()`. Все параметры передаются как `$N` placeholders — SQL-инъекции исключены.

* **GET** `/items?from=&to=&type=&category=&sort=&order=&limit=&offset=` — список с фильтрами.
  Сортировка whitelisted: `occurred_at` (default) | `amount` | `id`. Порядок: `asc` | `desc` (default).
  Пагинация: `limit` (1..500, default 50), `offset` (≥ 0). В ответе всегда `items`, `limit`, `offset`, `has_more` (через `limit + 1` трюк).

* **GET** `/items/{id}` — получить одну запись.
* **PUT** `/items/{id}` — обновить запись (тело — как у POST).
* **DELETE** `/items/{id}` — удалить запись. Возвращает `204 No Content`.

### Аналитика

* **GET** `/analytics?from=&to=` — агрегированная сводка за период строго по ТЗ.
  ```json
  {
    "from":   "2026-05-01T00:00:00Z",
    "to":     "2026-05-31T23:59:59Z",
    "count":  42,
    "sum":    158430.00,
    "avg":    3772.14,
    "median": 2500.00,
    "p90":    12000.00
  }
  ```
  Возвращаются ровно те 5 метрик, что требует ТЗ: `count`, `sum`, `avg`, `median`, `p90`.
  Медиана и P90 считаются на стороне БД через `percentile_cont(0.5 / 0.9) WITHIN GROUP (ORDER BY amount)`.
  Параметры `from`/`to` принимаются в RFC3339 либо `YYYY-MM-DD`; оба не обязательны.

### Экспорт

* **GET** `/items.csv?from=&to=&type=&category=` — CSV со всеми отфильтрованными записями (постранично, по 1000 строк за итерацию).

### Web UI

`/` → редирект на `/static/index.html`: форма добавления, фильтры, аналитические виджеты, таблица с inline-редактированием/удалением, кнопка «Скачать CSV».

## Запуск

### Docker Compose

```bash
cd L3.6
docker compose up --build
```

Поднимет:
1. **PostgreSQL 17** (порт `5436` снаружи).
2. **Миграции** (`golang-migrate`, источник `file://migrations`).
3. **Сервер** на `:8080`.

UI: <http://localhost:8080>.

### Локальный запуск (Postgres в Docker + Go нативно)

```bash
cd L3.6

# 1. БД
docker compose up postgres -d

# 2. Миграции
go run ./cmd/migrate -action up

# 3. Сервер
go run ./cmd/server
```

После запуска сервер слушает адрес из `config/local.yaml` (по умолчанию `:8080`).

## Конфиг

`config/local.yaml` (схема совпадает с L3.5):

```yaml
env: "local"

server:
  addr: ":8080"
db:
  host: "localhost"
  port: 5436
  user: "postgres"
  password: "postgres"
  dbname: "salestracker"
  sslmode: "disable"
```

`config/docker.yaml` отличается только адресом хоста PG (`postgres`, порт `5432`).
