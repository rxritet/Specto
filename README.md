# 🎯 Specto

> Производительное веб-приложение для управления задачами на Go.  
> Монолитная архитектура, минимум зависимостей, максимум предсказуемости.

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Build](https://github.com/rxritet/Specto/actions/workflows/ci.yml/badge.svg)](https://github.com/rxritet/Specto/actions)

---

## Технологический стек

| Слой | Решение |
|---|---|
| Язык | Go 1.25 |
| CLI | [Cobra](https://github.com/spf13/cobra) |
| БД (prod) | PostgreSQL (`lib/pq`) |
| БД (dev/lite) | BoltDB (pure Go, zero deps) |
| Сборка | [Mage](https://magefile.org) |
| Логирование | `slog` + `otelslog` (OpenTelemetry) |
| Тесты | `testcontainers-go` (ephemeral Postgres) |
| Линтер | `golangci-lint` |

---

## Структура проекта

```text
.
├── cmd/specto/          # Точка входа + Cobra команды (server, seed)
├── internal/
│   ├── config/          # Конфигурация через env-переменные
│   ├── database/        # PostgreSQL + BoltDB провайдеры, tx-in-context, миграции
│   ├── domain/          # Доменные модели (Task, User) и типы ошибок
│   ├── logging/         # Настройка slog/otelslog
│   ├── service/         # Бизнес-логика: TaskService, UserService, StatsService
│   │   ├── count_amd64.s    # SIMD-агрегация (AVX2, amd64)
│   │   └── count_generic.go # Pure-Go fallback
│   └── web/
│       ├── handlers.go  # HTTP-хендлеры (тонкий транспорт)
│       ├── middleware.go # Логирование запросов, panic recovery, security headers
│       ├── router.go    # net/http роутинг (Go 1.22+ pattern matching)
│       └── html/        # Decorator-типы + html/template + go:embed assets
├── deploy/
│   └── specto.service   # systemd unit
├── tests/               # Интеграционные тесты (Testcontainers)
├── magefile.go          # Автоматизация: dev, prod, test, seed, deploy
└── go.mod
```

---

## Быстрый старт

**Требования:** Go 1.25+, Docker (для интеграционных тестов), [Mage](https://magefile.org)

```bash
go install github.com/magefile/mage@latest

git clone https://github.com/rxritet/Specto.git
cd Specto
```

### Локальная разработка (BoltDB, шаблоны с диска)

```bash
mage dev
# → http://localhost:8080
```

### Production сборка

```bash
mage prod
# Бинарник: ./bin/specto  (stripped, trimpath, go:embed assets внутри)

./bin/specto server --db /var/lib/specto/data.db
```

### Seed данных

```bash
mage seed
# или: ./bin/specto seed
```

---

## Тесты

```bash
# Unit + интеграционные (поднимает ephemeral Postgres через Docker)
mage test

# Фаззинг парсеров пользовательского ввода
go test -fuzz=FuzzParseTaskTitle ./internal/web/...
```

---

## Деплой

```bash
# Один шаг: cross-compile linux/amd64 → rsync → systemctl restart
mage deploy

# Переменные окружения деплоя (опционально переопределить)
export DEPLOY_HOST=prod.example.com
export DEPLOY_USER=specto
export DEPLOY_DIR=/opt/specto
export DEPLOY_UNIT=specto
```

Systemd unit лежит в `deploy/specto.service` и копируется на сервер автоматически.

---

## Ключевые архитектурные решения

### Tx-in-context
`*sql.Tx` передаётся через `context.Context` — сигнатуры репозиториев остаются чистыми, транзакция не просачивается в интерфейсы.

### UI Decorator
`internal/web/html` содержит типы-обёртки над доменными моделями (`html.Task` embeds `domain.Task`) — UI-форматирование изолировано от домена.

### SIMD с fallback
`count_amd64.s` реализует AVX2-агрегацию для горячих путей на amd64; `count_generic.go` — pure-Go fallback, работающий на любой платформе.

### Двойная стратегия БД
BoltDB для локальной разработки и малых инсталляций (один `.db` файл), PostgreSQL для production. Оба провайдера реализуют один интерфейс репозитория.

---

## CLI команды

| Команда | Описание |
|---|---|
| `specto server` | Запустить HTTP-сервер |
| `specto seed` | Наполнить БД стартовыми данными |
| `specto migrate` | Применить SQL-миграции |
| `specto --help` | Полный список команд |

---

## Лицензия

MIT © 2026 Radmir Abraev
