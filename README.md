# PR Reviewer Assignment Service

Микросервис для автоматического назначения ревьюеров на Pull Request'ы.

## Описание

Сервис автоматически назначает до двух активных ревьюверов из команды автора PR, позволяет переназначать ревьюверов и управлять командами и пользователями.

### Основные возможности

- Создание команд с участниками
- Управление активностью пользователей
- Автоматическое назначение до 2 ревьюверов при создании PR
- Переназначение ревьюверов из команды заменяемого участника
- Идемпотентная операция merge PR
- Получение списка PR для конкретного ревьювера

## Быстрый старт

### Запуск с помощью docker-compose

```bash
docker-compose up
```

Сервис будет доступен на `http://localhost:8080`

### Остановка сервиса

```bash
docker-compose down
```

### Остановка с удалением данных

```bash
docker-compose down -v
```

## API Endpoints

Полная спецификация API доступна в файле `openapi.yml`.

### Teams

- `POST /team/add` - Создать команду с участниками
- `GET /team/get?team_name=<name>` - Получить информацию о команде

### Users

- `POST /users/setIsActive` - Изменить статус активности пользователя
- `GET /users/getReview?user_id=<id>` - Получить PR'ы, где пользователь назначен ревьювером

### Pull Requests

- `POST /pullRequest/create` - Создать PR с автоназначением ревьюверов
- `POST /pullRequest/merge` - Отметить PR как merged (идемпотентная операция)
- `POST /pullRequest/reassign` - Переназначить ревьювера

## Примеры использования

### Создание команды

```bash
curl -X POST http://localhost:8080/team/add \
  -H "Content-Type: application/json" \
  -d '{
    "team_name": "backend",
    "members": [
      {"user_id": "u1", "username": "Alice", "is_active": true},
      {"user_id": "u2", "username": "Bob", "is_active": true},
      {"user_id": "u3", "username": "Charlie", "is_active": true}
    ]
  }'
```

### Создание Pull Request

```bash
curl -X POST http://localhost:8080/pullRequest/create \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001",
    "pull_request_name": "Add feature",
    "author_id": "u1"
  }'
```

### Получение PR для ревьювера

```bash
curl http://localhost:8080/users/getReview?user_id=u2
```

### Merge PR

```bash
curl -X POST http://localhost:8080/pullRequest/merge \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001"
  }'
```

### Переназначение ревьювера

```bash
curl -X POST http://localhost:8080/pullRequest/reassign \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001",
    "old_user_id": "u2"
  }'
```

## Архитектура

Проект использует трехслойную архитектуру:

- **Handler** - HTTP эндпоинты (маршрутизация, валидация)
- **Service** - бизнес-логика (назначение ревьюверов, переназначение)
- **Repository** - работа с PostgreSQL

### Структура проекта

```
.
├── cmd/
│   └── server/
│       └── main.go              # Точка входа
├── internal/
│   ├── handler/                 # HTTP handlers
│   ├── models/                  # Структуры данных
│   ├── repository/              # Работа с БД
│   └── service/                 # Бизнес-логика
├── migrations/
│   └── 001_init.sql            # SQL миграции
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── openapi.yml
└── README.md
```

## Конфигурация

Сервис настраивается через переменные окружения:

- `DB_HOST` - хост PostgreSQL (по умолчанию: `localhost`)
- `DB_PORT` - порт PostgreSQL (по умолчанию: `5432`)
- `DB_USER` - пользователь БД (по умолчанию: `postgres`)
- `DB_PASSWORD` - пароль БД (по умолчанию: `password`)
- `DB_NAME` - имя БД (по умолчанию: `postgres`)
- `SERVER_PORT` - порт сервера (по умолчанию: `8080`)

## Бизнес-логика

### Назначение ревьюверов

При создании PR:
1. Определяется команда автора
2. Выбирается до 2 активных участников команды (исключая автора)
3. Выбор происходит случайным образом
4. Если доступных кандидатов меньше двух, назначается доступное количество (0/1)

### Переназначение ревьювера

При переназначении:
1. Проверяется, что PR не в статусе MERGED
2. Проверяется, что указанный пользователь назначен ревьювером
3. Находится команда заменяемого пользователя
4. Выбирается случайный активный участник команды (исключая автора PR и текущих ревьюверов)
5. Происходит замена ревьювера

### Merge PR

Операция merge идемпотентна - повторный вызов не вызывает ошибку и возвращает текущее состояние PR.

## Допущения и решения

1. Случайный выбор ревьюверов: Используется math.rand

2. Применение миграций: Миграции применяются автоматически при старте сервиса

3. Graceful shutdown: Реализована корректная остановка HTTP сервера с таймаутом

4. Connection pool: Настроен пул соединений с БД для оптимальной производительности

5. Retry logic: Реализована логика повторных подключений к БД при старте

## Технологический стек

- **Go 1.21**
- **PostgreSQL 15**
- **Docker & Docker Compose**
- **lib/pq** - PostgreSQL драйвер

