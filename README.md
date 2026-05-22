# SubAggregator

REST API для учёта подписок: создание, просмотр, обновление, удаление и расчёт суммарной стоимости за период.

## Возможности

- CRUDL операций над подписками
- Расчёт суммарной стоимости по пользователю, сервису и периоду
- PostgreSQL + миграции ([golang-migrate](https://github.com/golang-migrate/migrate))
- Swagger UI и OpenAPI-спека
- Структурированное логирование (`log/slog`)
- Docker Compose для локальной БД
- Слои по принципам чистой архитектуры: handlers → service → repository

## Стек

| Компонент | Технология |
|-----------|------------|
| Язык | Go 1.25+ |
| HTTP | [Gin](https://github.com/gin-gonic/gin) |
| БД | PostgreSQL 16, [pgx](https://github.com/jackc/pgx) |
| Миграции | golang-migrate |
| Конфиг | Viper (YAML) |
| Документация API | swaggo/swag |
| Тесты | testify |

## Архитектура

```
HTTP (Gin)
    ↓
handlers/      transport: парсинг запроса, коды ответов, логи
    ↓
service/       бизнес-логика: даты MM-YYYY, валидация, правила
    ↓
repository/    persistence: SQL, транзакции
    ↓
PostgreSQL
```

| Слой | Ответственность |
|------|----------------|
| `handlers` | HTTP, Swagger-аннотации, маппинг ошибок в статусы |
| `service` | Use cases, `ParseMonthYear`, доменные ошибки |
| `repository` | Запросы к БД, реализует `service.SubscriptionRepository` |
| `models` | DTO и сущности для JSON / БД |

## Структура проекта

```
SubAggregator/
├── cmd/
│   ├── app/                    # Точка входа HTTP-сервера
│   └── migrate/                # CLI миграций
├── docs/                       # Swagger (генерируется)
├── internal/
│   ├── config/                 # Конфигурация (YAML)
│   ├── database/               # Пул подключений pgx
│   ├── handlers/               # HTTP-хендлеры
│   │   ├── integration/        # Интеграционные тесты (build tag)
│   │   └── subscription.go
│   ├── middleware/             # Middleware (slog)
│   ├── migrations/             # SQL-миграции
│   ├── models/                 # Модели и DTO
│   ├── repository/             # Слой БД
│   └── service/                # Бизнес-логика
│       ├── mocks/              # Моки репозитория для тестов
│       ├── subscription.go   # интерфейс SubscriptionRepository
│       ├── date.go
│       └── errors.go
├── .env.example                # шаблон пароля (скопировать в .env)
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── README.md
```

## Требования

- Go 1.25+
- Docker и Docker Compose (для PostgreSQL)
- Make (опционально)

## Быстрый старт

### 1. Клонирование и зависимости

```bash
git clone https://github.com/LimeAsnet/SubAggregator.git
cd SubAggregator
go mod download
```

### 2. Конфигурация

**Пароль БД** — только в `.env` (не в yaml):

```bash
cp .env.example .env
cp internal/config/local.yaml.example internal/config/local.yaml
```

В `.env` задайте `POSTGRES_PASSWORD=...` — его же подхватит приложение (`DB_PASSWORD` или `POSTGRES_PASSWORD`).

**Остальное** (хост, порт, пользователь, `srvHost`) — в yaml:

| Файл | Когда |
|------|--------|
| `internal/config/local.yaml` | Локально: `make run`, тесты (`dbHost: localhost`) |
| `internal/config/docker.yaml` | В Docker: `CONFIG_NAME=docker` (`dbHost: postgres`) |

Пример `local.yaml` — см. `local.yaml.example` (поля `dbPassword` в yaml нет).

### 3. Запуск

**Весь стек в Docker** (PostgreSQL + миграции + API):

```bash
make docker-up
```

**Локальная разработка** (только БД в Docker):

```bash
make dev
```

Или по шагам:

```bash
make docker-up-db
make migrate-up
make run
```

Сервер: **http://localhost:8082**  
Swagger: **http://localhost:8082/swagger/index.html**

### Docker

| Команда | Описание |
|---------|----------|
| `make docker-up` | Собрать образ и запустить postgres → migrate → app |
| `make docker-up-db` | Только PostgreSQL |
| `make docker-down` | Остановить контейнеры |
| `make docker-logs` | Логи всех сервисов |
| `make docker-logs-app` | Логи API |

В Docker: `docker.yaml` + переменные из `.env`. Compose передаёт в `app` и `migrate`:

- `CONFIG_NAME=docker` → читается `docker.yaml`
- `DB_PASSWORD=${POSTGRES_PASSWORD}` → пароль для подключения к Postgres

Сервис `postgres` берёт `POSTGRES_PASSWORD` из того же `.env`. Без `.env` compose выдаст ошибку с подсказкой создать файл.

Хост БД внутри сети Compose: `postgres` (из `docker.yaml`).

Значения `POSTGRES_USER` / `POSTGRES_DB` в compose (дефолты `postgres`, `subHubdb`) должны совпадать с `dbUser` / `dbName` в `docker.yaml`.

Порты на хосте (опционально в `.env`): `POSTGRES_PORT` (дефолт `5432`), `APP_PORT` (дефолт `8082`).

## API

Базовый путь: `/api/v1`

| Метод | Путь | Описание |
|-------|------|----------|
| `POST` | `/subscriptions` | Создать подписку |
| `GET` | `/subscriptions?user_id={uuid}` | Список подписок пользователя |
| `GET` | `/subscriptions/total` | Суммарная стоимость за период |
| `PATCH` | `/subscriptions/{id}` | Обновить дату окончания (`end_date`) |
| `DELETE` | `/subscriptions/{id}` | Удалить подписку |

### Формат дат

В запросах даты передаются в формате **`MM-YYYY`** (например, `07-2025`). Парсинг выполняется в слое `service`.

### Примеры запросов

**Создать подписку**

```bash
curl -X POST http://localhost:8082/api/v1/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "service_name": "Netflix",
    "monthly_cost": 599,
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "start_date": "07-2025",
    "end_date": "12-2025"
  }'
```

**Список подписок**

```bash
curl "http://localhost:8082/api/v1/subscriptions?user_id=550e8400-e29b-41d4-a716-446655440000"
```

**Суммарная стоимость**

```bash
curl "http://localhost:8082/api/v1/subscriptions/total?user_id=550e8400-e29b-41d4-a716-446655440000&service_name=Netflix&start_date=01-2025&end_date=12-2025"
```

**Обновить дату окончания подписки**

Тело запроса содержит только `end_date` (формат `MM-YYYY`):

```bash
curl -X PATCH http://localhost:8082/api/v1/subscriptions/1 \
  -H "Content-Type: application/json" \
  -d '{"end_date": "12-2026"}'
```

**Удалить подписку**

```bash
curl -X DELETE http://localhost:8082/api/v1/subscriptions/1
```

## Swagger

| Ресурс | URL |
|--------|-----|
| Swagger UI | http://localhost:8082/swagger/index.html |
| OpenAPI JSON | http://localhost:8082/swagger/doc.json |

```bash
make swagger
```

## Тестирование

```bash
# Юнит-тесты (service, handlers)
make test

# Интеграционные (нужен Postgres + миграции)
make test-integration
```

Обязательно: тег `integration` (`go test -tags=integration ./internal/handlers/integration/...`). Без тега Go пишет `matched no packages`.

Перед интеграционными тестами: `.env` и `local.yaml` из example-файлов, `make docker-up-db`, `make migrate-up`.

| Пакет | Что тестируется |
|-------|-----------------|
| `internal/service` | Бизнес-логика, мок репозитория |
| `internal/handlers/integration` | Полный HTTP-стек + БД |

## Makefile

| Команда | Описание |
|---------|----------|
| `make help` | Список целей |
| `make docker-up` | Полный стек: postgres → migrate → app |
| `make docker-up-db` | Только PostgreSQL |
| `make docker-down` | Остановить контейнеры |
| `make migrate-up` | Применить миграции |
| `make migrate-down` | Откатить последнюю миграцию |
| `make migrate-version` | Версия схемы |
| `make swagger` | Сгенерировать Swagger |
| `make test` | Юнит-тесты |
| `make test-integration` | Интеграционные тесты |
| `make run` | Запустить API |
| `make build` | Собрать бинарники |
| `make dev` | docker-up-db + migrate-up + run (БД в Docker, app локально) |

## Миграции

Файлы: `internal/migrations/`

```bash
go run ./cmd/migrate up
go run ./cmd/migrate down
go run ./cmd/migrate version
```

## Схема БД

Таблица `subscriptions`:

| Поле | Тип | Описание |
|------|-----|----------|
| `id` | BIGSERIAL | Первичный ключ |
| `service_name` | VARCHAR(255) | Название сервиса |
| `monthly_cost` | BIGINT | Стоимость в месяц (≥ 0) |
| `user_id` | UUID | Пользователь |
| `start_date` | DATE | Дата начала |
| `end_date` | DATE | Дата окончания (`NULL` — активная подписка) |

## Лицензия

MIT
