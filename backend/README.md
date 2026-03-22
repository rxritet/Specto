# Specto

> Монолитный backend-проект на Go для управления задачами.
> 
> Текущий фокус: чистый Go backend, HTTP API, аутентификация, простая эксплуатация и предсказуемая архитектура.

[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

---

## Что это за проект

Specto — это backend-приложение на Go с монолитной архитектурой. Оно предоставляет:

- HTTP API для пользователей и задач
- cookie-based аутентификацию
- два варианта хранения данных: PostgreSQL и BoltDB
- CLI для запуска сервера и заполнения базы тестовыми данными
- unit, fuzz и integration tests

Проект intentionally простой: стандартная библиотека используется везде, где это оправдано, а внешние зависимости добавлены только для конкретных задач вроде CLI, bcrypt и integration testing.

---

## Технологический стек

| Слой | Решение |
|---|---|
| Язык | Go 1.26 |
| HTTP | `net/http` + `http.ServeMux` с pattern routing |
| CLI | [Cobra](https://github.com/spf13/cobra) |
| БД (prod) | PostgreSQL (`lib/pq`) |
| БД (lite/dev) | BoltDB |
| Auth | signed HTTP-only cookie + `bcrypt` |
| Логирование | `slog` + `otelslog` |
| Сборка | [Mage](https://magefile.org) |
| Integration tests | `testcontainers-go` |

---

## Структура проекта

```text
.
├── cmd/specto/              # CLI entrypoint: server, seed
├── deploy/
│   └── specto.service       # systemd unit for deployment
├── internal/
│   ├── config/              # ENV-based configuration
│   ├── database/            # PostgreSQL, BoltDB, migrations, tx helpers
│   ├── domain/              # Domain models and typed errors
│   ├── logging/             # slog fanout handler
│   ├── server/              # HTTP server bootstrap + graceful shutdown
│   ├── service/             # Business logic for users, tasks, stats, auth
│   └── web/                 # HTTP handlers, middleware, auth, router, fuzz tests
├── tests/
│   └── integration/         # Docker-based integration tests for Postgres
├── magefile.go              # dev, prod, test, seed, deploy tasks
├── go.mod
└── README.md
```

---

## Возможности на текущий момент

- Регистрация пользователя
- Логин и логаут
- Проверка текущей сессии через `/auth/me`
- CRUD для задач
- Статистика задач по статусам
- Защита task endpoints через middleware аутентификации
- Привязка задач к текущему пользователю
- Graceful shutdown сервера
- Поддержка PostgreSQL и BoltDB через общий repository contract

---

## HTTP API

### Public endpoints

| Метод | Путь | Описание |
|---|---|---|
| `GET` | `/health` | Проверка, что сервер жив |
| `POST` | `/auth/register` | Регистрация пользователя |
| `POST` | `/auth/login` | Логин и установка сессионной cookie |
| `POST` | `/auth/logout` | Очистка сессии |

### Protected endpoints

| Метод | Путь | Описание |
|---|---|---|
| `GET` | `/auth/me` | Текущий пользователь |
| `GET` | `/tasks` | Список задач текущего пользователя |
| `POST` | `/tasks` | Создание задачи |
| `GET` | `/tasks/{id}` | Получение задачи по id |
| `PUT` | `/tasks/{id}` | Обновление задачи |
| `DELETE` | `/tasks/{id}` | Удаление задачи |
| `GET` | `/tasks/stats` | Статистика задач текущего пользователя |

### Пример регистрации

```bash
curl -i -X POST http://localhost:8080/auth/register \
	-H "Content-Type: application/json" \
	-d '{
		"email": "alice@example.com",
		"name": "Alice",
		"password": "password123"
	}'
```

### Пример создания задачи после логина

```bash
curl -i -X POST http://localhost:8080/tasks \
	-H "Content-Type: application/json" \
	-b cookies.txt \
	-d '{
		"title": "Write report",
		"description": "Q4 summary",
		"status": "in_progress"
	}'
```

`user_id` в защищённых task endpoints не является источником истины: сервер использует id текущего аутентифицированного пользователя из сессии.

---

## Конфигурация

### Базовая

| Переменная | Значение по умолчанию | Описание |
|---|---|---|
| `SPECTO_HOST` | `0.0.0.0` | Host сервера |
| `SPECTO_PORT` | `8080` | Port сервера |
| `SPECTO_LOG_LEVEL` | `info` | Уровень логирования |
| `SPECTO_DB_PROVIDER` | `bolt` | `bolt` или `postgres` |
| `SPECTO_DB_DSN` | пусто | DSN для PostgreSQL |
| `SPECTO_DB_PATH` | `specto.db` | Путь к BoltDB файлу |

### Аутентификация

| Переменная | Значение по умолчанию | Описание |
|---|---|---|
| `SPECTO_AUTH_SECRET` | `specto-dev-secret-change-me` | Секрет для подписи cookie |
| `SPECTO_AUTH_SESSION_TTL` | `24h` | TTL сессии |
| `SPECTO_AUTH_SECURE_COOKIES` | `false` | Выставлять `Secure` для cookie |

Для production `SPECTO_AUTH_SECRET` нужно обязательно переопределять.

---

## Быстрый старт

**Требования:** Go 1.26+, опционально Docker для integration tests, опционально Mage.

```bash
git clone https://github.com/rxritet/Specto.git
cd Specto
```

### Вариант 1: быстрый локальный запуск через BoltDB

```bash
go run ./cmd/specto server
```

Или через Mage:

```bash
go install github.com/magefile/mage@latest
mage dev
```

Сервер будет доступен на `http://localhost:8080`.

### Вариант 2: запуск с PostgreSQL

```bash
export SPECTO_DB_PROVIDER=postgres
export SPECTO_DB_DSN='postgres://user:pass@localhost:5432/specto?sslmode=disable'
go run ./cmd/specto server
```

### Seed тестовых данных

```bash
go run ./cmd/specto seed
```

Или:

```bash
mage seed
```

---

## Сборка

### Production build

```bash
mage prod
```

Артефакт будет лежать в:

- `bin/specto` на Linux/macOS
- `bin/specto.exe` на Windows

Также можно собрать напрямую:

```bash
go build -o ./bin/specto ./cmd/specto
```

---

## Тесты

### Unit и обычные package tests

```bash
go test ./...
```

или:

```bash
mage test
```

### Integration tests с PostgreSQL в Docker

Integration tests теперь opt-in и запускаются только если явно включить их через переменную окружения:

```bash
SPECTO_RUN_INTEGRATION=1 go test ./tests/integration/...
```

Или вместе со всем проектом:

```bash
SPECTO_RUN_INTEGRATION=1 go test ./...
```

### Fuzz tests

```bash
go test -fuzz=FuzzDecodeTaskJSON ./internal/web
go test -fuzz=FuzzDecodeTaskForm ./internal/web
```

---

## Архитектурные решения

### Context-aware backend

Repository и service layer принимают `context.Context`, поэтому проект уже готов к:

- request cancellation
- timeouts
- transaction propagation
- middleware-driven cross-cutting logic

### Tx-in-context

Транзакция может пробрасываться через `context.Context`, при этом репозиторные интерфейсы остаются единообразными.

### Два storage backend'а

- BoltDB удобен для локальной разработки и небольших инсталляций
- PostgreSQL подходит для production и integration testing

Оба backend'а реализуют один и тот же набор repository interfaces.

### Cookie auth без внешнего session store

Сессия хранится в подписанной cookie:

- `HttpOnly`
- `SameSite=Lax`
- HMAC-подпись на `SHA-256`
- TTL управляется через конфиг

Это keeps the system simple, пока не потребуется полноценное server-side session invalidation.

### SIMD с fallback

Подсчёт task statistics ускорен на amd64 через asm-реализацию, при этом для остальных платформ остаётся pure-Go fallback.

---

## CLI команды

| Команда | Описание |
|---|---|
| `specto server` | Запустить HTTP-сервер |
| `specto seed` | Наполнить базу стартовыми данными |
| `specto --help` | Показать список команд |

---

## Деплой

Проект содержит `mage deploy`, который:

1. cross-compiles бинарник под `linux/amd64`
2. отправляет бинарник и `deploy/specto.service` на сервер через `rsync`
3. выполняет `systemctl daemon-reload`, `enable` и `restart`

Переменные для деплоя:

- `DEPLOY_HOST`
- `DEPLOY_USER`
- `DEPLOY_DIR`
- `DEPLOY_UNIT`

---

## Лицензия

MIT © 2026 Radmir Abraev
