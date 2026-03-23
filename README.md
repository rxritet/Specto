# Specto

Банковский монорепозиторий: Go backend + Angular frontend.

Проект находится в стадии миграции от legacy-функциональности задач к банковским модулям. Основной вектор развития: счета, карты и переводы.

## Что Это За Проект

Specto объединяет:

- Backend на Go с HTTP API и triple-store архитектурой: PostgreSQL + Redis + BoltDB.
- Frontend на Angular + TypeScript для пользовательского интерфейса.
- Локальную инфраструктуру (PostgreSQL + pgAdmin + Redis) через Docker Compose.

На текущий момент в кодовой базе присутствует legacy-логика задач (tasks) для совместимости. Новые изменения должны идти в банковские модули.

## Стек Технологий

- Backend: Go 1.26, net/http, ServeMux, Cobra, slog.
- Storage: PostgreSQL (ACID-ядро), Redis (сессии/rate-limit/кэш), BoltDB (audit log).
- Frontend: Angular 19, TypeScript, RxJS.
- CI/CD: GitHub Actions.
- Локальная инфраструктура: Docker Compose.

## Структура Репозитория

```text
.
├── backend/                    # Go backend
│   ├── cmd/specto/             # CLI команды (server, seed)
│   ├── internal/
│   │   ├── config/             # Конфигурация из env
│   │   ├── database/           # PostgreSQL/BoltDB, миграции, tx
│   │   ├── domain/             # Доменные модели и ошибки
│   │   ├── service/            # Бизнес-логика
│   │   └── web/                # HTTP handlers, middleware, router
│   ├── tests/integration/      # Интеграционные тесты
│   └── go.mod
├── frontend/                   # Angular frontend
├── .github/workflows/ci.yml    # CI pipeline
├── docker-compose.yml          # PostgreSQL + pgAdmin + Redis
└── README.md
```

## Быстрый Старт

Требования:

- Go 1.26+
- Node.js 22+
- Docker Desktop (для PostgreSQL/pgAdmin)

### 1. Поднять Базу Данных

```bash
docker compose up -d
```

После запуска:

- PostgreSQL: localhost:5432
- pgAdmin: http://localhost:5050
- Redis: localhost:6379

### 2. Запустить Backend

```bash
cd backend
go test -count=1 ./...
go run ./cmd/specto server
```

Backend по умолчанию слушает порт 8080.

### 3. Запустить Frontend

```bash
cd frontend
npm install
npm start
```

Frontend по умолчанию доступен на http://localhost:4200.

## Конфигурация Backend

Ключевые переменные окружения:

- SPECTO_HOST
- SPECTO_PORT
- SPECTO_LOG_LEVEL
- SPECTO_POSTGRES_DSN
- SPECTO_REDIS_ADDR
- SPECTO_REDIS_PASSWORD
- SPECTO_REDIS_DB
- SPECTO_BOLT_PATH
- SPECTO_RATE_LIMIT_PER_MINUTE
- SPECTO_BALANCE_CACHE_TTL
- SPECTO_AUTH_SECRET
- SPECTO_AUTH_SESSION_TTL
- SPECTO_AUTH_SECURE_COOKIES

Рекомендуется хранить секреты во внешнем env-файле и не коммитить их в репозиторий.

## Тестирование

### Backend

```bash
cd backend
go test -count=1 ./...
go test -race -count=1 -coverprofile=coverage.out ./...
```

Интеграционные тесты:

```bash
cd backend
SPECTO_RUN_INTEGRATION=1 go test -tags integration -race -count=1 -timeout 120s ./tests/integration/
```

### Frontend

```bash
cd frontend
npm test -- --watch=false --browsers=ChromeHeadless
npm run build
```

## CI

GitHub Actions настроен в [ .github/workflows/ci.yml ](.github/workflows/ci.yml):

- Backend lint/test/integration/build
- Frontend build/test

## Текущий Статус И План

- Реализована монорепозиторная структура backend/frontend.
- Настроены базовые пайплайны CI.
- В проекте еще присутствует legacy task-функциональность.
- Следующие шаги: выделение и развитие банковских модулей accounts, cards, transfers.

## Полезные Файлы

- Backend документация: [ backend/README.md ](backend/README.md)
- Frontend документация: [ frontend/README.md ](frontend/README.md)
- Backend доменные модели: [ backend/internal/domain/models.go ](backend/internal/domain/models.go)
- Backend роутер: [ backend/internal/web/router.go ](backend/internal/web/router.go)
- Миграции БД: [ backend/internal/database/migrations/001_init.up.sql ](backend/internal/database/migrations/001_init.up.sql)
- CI workflow: [ .github/workflows/ci.yml ](.github/workflows/ci.yml)
