# L3.4: Image Processing Pipeline

Данный проект реализует распределенный пайплайн для обработки изображений с использованием микросервисной архитектуры на Go (API Gateway + Kafka + Worker).

## Архитектура
* **image-processor (API)**: Принимает HTTP-запросы на загрузку изображений (multipart/form-data), сохраняет оригинальный файл на диск, делает запись в PostgreSQL (статус `processing`) и отправляет задачу в Kafka. Отдает статус или готовую картинку по запросу.
* **Kafka**: Выступает надежной очередью сообщений, обеспечивающей асинхронную передачу задач на обработку с гарантией доставки (at-least-once) и устойчивостью к падению воркеров.
* **worker (Consumer)**: Читает задачи из Kafka, скачивает оригинальное изображение, применяет нужный режим обработки (через библиотеки `imaging` и `gg`), сохраняет результат и обновляет статус в БД на `ready` (или `failed`).
* **PostgreSQL**: Хранит метаданные о загруженных картинках (id, статусы, пути к файлам на диске).

## Режимы обработки картинок
Воркер поддерживает следующие значения параметра `mode`:
1. `resize`: Изменение размера с сохранением пропорций (укажите `width` или `height`).
2. `thumb`: Создание квадратной или прямоугольной миниатюры с обрезкой по центру (Smart Crop) под заданные `width` и `height`.
3. `watermark`: Наложение водяного знака. Текст можно передать в параметре `watermark_text` (по умолчанию: "WATERMARK").

## Быстрый старт (Локальный запуск)

1. **Запуск инфраструктуры** (БД + Kafka)
   ```bash
   # Запуск PostgreSQL
   docker compose -f docker/posgres/docker-compose.yml up -d
   # Запуск Kafka
   docker compose -f docker/kafka/docker-compose.yaml up -d
   ```
2. **Применение миграций PostgreSQL**
   Перейдите в директорию `image-processor` и запустите встроенный скрипт для миграций. Он автоматически подключится к БД, используя настройки из `config/local.yaml`, и применит файлы из папки `migrations`:
   ```bash
   cd services/image-processor
   go run cmd/migrate/migrations.go -action=up
   ```

3. **Запуск сервисов**
   Откройте две разные консоли.
   * Консоль 1: `cd services/image-processor && go run ./cmd/image-processor/main.go`
   * Консоль 2: `cd services/worker && go run ./cmd/worker/main.go`

---

## Примеры API (cURL)

### 1. Загрузка изображения (POST /upload)
*Загружает файл и ставит задачу на обработку. Возвращает ID задачи (UUID) и статус `processing`.*

**Пример 1: Водяной знак**
```bash
curl -X POST http://localhost:8080/upload \
  -F "mode=watermark" \
  -F "watermark_text=MY_SUPER_APP" \
  -F "file=@/path/to/your/image.jpg"
```

**Пример 2: Уменьшение размера с сохранением пропорций**
```bash
curl -X POST http://localhost:8080/upload \
  -F "mode=resize" \
  -F "width=300" \
  -F "file=@/path/to/your/image.jpg"
```

**Пример 3: Квадратная миниатюра (Crop Center)**
```bash
curl -X POST http://localhost:8080/upload \
  -F "mode=thumb" \
  -F "width=150" \
  -F "height=150" \
  -F "file=@/path/to/your/image.jpg"
```
**Ответ:** `{"id": "d290f1ee-6c54-4b01-90e6-d701748f0851", "status": "processing"}`

---

### 2. Получение статуса / картинки (GET /image/{id})
*Если картинка еще обрабатывается, вернет JSON 202 Accepted. Если готова — скачает саму картинку.*

```bash
curl -i http://localhost:8080/image/d290f1ee-6c54-4b01-90e6-d701748f0851
```
* **Ответ (еще в процессе):** `HTTP/1.1 202 Accepted` + `{"id":"...","status":"processing"}`
* **Ответ (готово):** `HTTP/1.1 200 OK` (Content-Type: image/jpeg) и потоковая передача байт картинки.

---

### 3. Удаление картинки (DELETE /image/{id})
*Удаляет метаданные из БД, а также физически стирает с диска оригинальный файл и обработанный результат.*

```bash
curl -i -X DELETE http://localhost:8080/image/d290f1ee-6c54-4b01-90e6-d701748f0851
```
* **Ответ:** `HTTP/1.1 204 No Content`
