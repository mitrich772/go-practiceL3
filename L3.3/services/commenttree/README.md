
# CommentTree — древовидные комментарии
````md
Сервис для работы с древовидными комментариями (неограниченная вложенность):

- создание комментариев и ответов
- получение поддерева комментариев
- удаление комментария вместе с поддеревом
- пагинация корневых комментариев
- полнотекстовый поиск по комментариям

````
## API

### Создание комментария
**POST** `/comments`

Корневой комментарий:
```json
{
  "body": "Привет!"
}
````

Ответ на комментарий:

```json
{
  "parent_id": 1,
  "body": "Ответ на комментарий #1"
}
```

---

### Получение поддерева

**GET** `/comments?parent={id}&depth={n}`

Пример:

```
/comments?parent=1&depth=2
```
---

### Удаление комментария

**DELETE** `/comments/{id}`

Пример:

```
DELETE /comments/10
```

---

### Получение корневых комментариев (пагинация)

**GET** `/roots?limit=20&offset=0&sort=created_at&order=desc`

---

### Полнотекстовый поиск

**GET** `/search?q=слово&limit=20&offset=0&sort=rank&order=desc`

---

## Запуск

### 1) PostgreSQL в Docker

В `.\docker\posgres` выполнить:

```bash
docker compose up -d
```



---

### 2) Миграции базы 

Мигратор вынесен в `cmd\migrate`:

```bash
cd services\commenttree
go run .\cmd\migrate\main.go -action up
```

---

### 3) Запуск сервиса

```bash
cd services\commenttree
go run .\cmd\commenttree\main.go
```

После запуска сервер будет слушать адрес из конфига, например:

```
localhost:8080
```

## Конфиг

Файл конфига:

```
config/local.yaml
```

Пример:

```yaml
storage:
  host: localhost
  port: 5433
  user: postgres
  password: postgres
  dbname: postgres

http_server:
  address: localhost:8080
  timeout: 4s
  idle_timeout: 60s
```
