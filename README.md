# 🎯 Specto

> Производительное веб-приложение для управления продуктивностью на Go.  
> Создаём первую версию приложения — не двадцатую.

[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)]()

---

## Философия

Specto построен на принципах **Simple Software**, вдохновлённых Джоном Кэлхуном.
Мы пишем код для людей, двигаемся итеративно и сознательно отказываемся от Hype Driven Development.

| Принцип | Как это выглядит на практике |
|---|---|
| Никаких преждевременных микросервисов | Один бинарник, монолитный layout — до тех пор, пока сложность не потребует иного |
| Сначала stdlib | `net/http`, `html/template`, `slog` — внешние зависимости добавляются только тогда, когда они оправданы |
| Толстый домен, тонкий транспорт | HTTP-хендлеры не содержат бизнес-логики |
| Директория = пакет | Новая директория создаётся только при необходимости нового пакета |

---

## Технологический стек

### Ядро — Go 1.26

- **GC «Green Tea»** — новый сборщик мусора с отключаемой паузой; снижение накладных расходов памяти на 10–40% в production
- `errors.AsType` — типобезопасные проверки ошибок вместо type switch
- `new(T(v))` — встроенный хелпер для указателей на литералы (не нужен `common.Ptr`)
- **SIMD (экспериментально, amd64)** — векторизованные агрегации в горячих путях с обязательным pure-Go fallback

### Слой данных — двойная стратегия хранения

| Движок | Окружение | Причина |
|---|---|---|
| **PostgreSQL** | Production | ACID-транзакции, проверенная надёжность |
| **BoltDB** | Малые инсталляции / разработка | Pure Go, ноль внешних зависимостей, один бинарник + один `.db` файл |

Транзакции передаются через `context.Context` — сигнатуры методов репозиториев остаются чистыми.

### Транспорт и UI

- `net/http` с улучшенным роутингом Go 1.22+ — сторонний роутер не нужен
- `html/template` — Server-Side Rendering, минимум клиентского JavaScript
- **Паттерн Decorator** — `html.Task` встраивает `app.Task` для UI-логики форматирования, не загрязняя домен
- `embed` — шаблоны и статика компилируются прямо в бинарный файл

### Инструментарий

- **[Mage](https://magefile.org)** — автоматизация сборки на Go (без Makefile)
- **[Cobra](https://github.com/spf13/cobra)** — CLI для управления сервером и административных задач
- **[Testcontainers-Go](https://golang.testcontainers.org)** — эфемерные Docker-контейнеры для интеграционных тестов
- **`slog` + `otelslog`** — структурированные логи с OpenTelemetry трейсами

---

## Структура проекта

Следуем стандарту **Server Project Layout** Алекса Эдвардса.

> Правило: создавай директорию только тогда, когда действительно нужен новый пакет.

```text
.
├── cmd/
│   └── specto/              # Точка входа, инициализация Cobra CLI
│       └── main.go
├── internal/                # Приватный код — граница импорта на уровне компилятора
│   ├── database/            # Реализации PostgreSQL + BoltDB, обёртки tx-in-context
│   ├── service/             # Вся бизнес-логика (Service Objects)
│   └── web/                 # HTTP-хендлеры и рендеринг html/template
├── pkg/                     # Пакеты, доступные для внешнего импорта
├── tests/
│   └── integration/         # Тесты на Testcontainers-Go + fixtures.sql
├── assets/                  # Шаблоны и статика (встраиваются через go:embed)
├── magefile.go              # Автоматизация: dev, prod, test, seed
└── go.mod
```

---

## Ключевые архитектурные паттерны

### Транзакции через Context

```go
// Сигнатуры репозитория остаются чистыми — tx извлекается из ctx внутри
func (r *TaskRepo) Create(ctx context.Context, t *Task) error {
    tx := txFromContext(ctx) // *sql.Tx не попадает в сигнатуру
    ...
}
```

### UI Decorator

```go
// internal/web/html/task.go
type Task struct {
    app.Task // встраиваем доменную модель
}

func (t Task) FormattedDeadline() string {
    return t.Deadline.Format("02 Jan 2006")
}
```

### SIMD с Fallback

```go
func aggregateDurations(tasks []Task) time.Duration {
    if cpu.X86.HasAVX2 {
        return aggregateSIMD(tasks)    // экспериментальный путь Go 1.26
    }
    return aggregateFallback(tasks)    // pure Go, всегда корректен
}
```

---

## Быстрый старт

**Требования:** Go 1.26+, Docker (для интеграционных тестов), Mage

```bash
# Установить Mage
go install github.com/magefile/mage@latest

# Клонировать
git clone https://github.com/rxritet/Specto.git
cd Specto
```

### Локальная разработка

```bash
# Запуск с чтением шаблонов с диска (быстрая правка без перекомпиляции)
mage dev
```

### Production сборка

```bash
# Компиляция с встроенными активами под целевую ОС
mage prod

# Бинарник полностью самодостаточен
./bin/specto server --db /var/lib/specto/data.db
```

### Наполнение базы данных

```bash
# Стартовые данные из кода/JSON — без сторонних CMS
# Seed-данные версионируются вместе с кодом
./bin/specto seed
```

---

## Тестирование

Следуем **TDD** по методологии *Learn Go with Tests* — тесты формируют контракты, а не только проверяют.

```bash
# Полный набор тестов (unit + integration + fuzz corpus)
mage test
```

### Интеграционные тесты

Эфемерный PostgreSQL через Testcontainers-Go. Каждый запуск — чистое состояние:

```go
func TestTaskService_Create(t *testing.T) {
    ctx := context.Background()
    pg, _ := postgres.RunContainer(ctx, ...)
    defer pg.Terminate(ctx)

    // fixtures.sql загружается перед каждым тестом для воспроизводимости
    applyFixtures(t, pg.ConnectionString())
    ...
}
```

### Фаззинг

Критичные парсеры пользовательского ввода (описания задач, строки длительности) покрыты встроенным фаззингом Go:

```bash
go test -fuzz=FuzzParseDuration ./internal/service/...
```

---

## Развёртывание

Деплой без лишних движений: **Build → Rsync → Restart**

```bash
mage prod
rsync -avz ./bin/specto user@server:/usr/local/bin/specto
ssh user@server "sudo systemctl restart specto"
```

### Systemd Unit

```ini
# /etc/systemd/system/specto.service
[Unit]
Description=Specto Productivity Hub
After=network.target

[Service]
ExecStart=/usr/local/bin/specto server --db /var/lib/specto/data.db
Restart=always
User=www-data
Environment=PORT=8080

[Install]
WantedBy=multi-user.target
```

---

## Наблюдаемость (Observability)

Specto использует **`slog`** как основной структурированный логгер с мостом **`otelslog`** для единых логов + трейсов — согласно стабильному OpenTelemetry Logs API (2025+).

```go
// main.go — логгер инициализируется один раз, распространяется через context
logger := slog.New(otelslog.NewHandler("specto"))
slog.SetDefault(logger)
```

Runtime-метрики (горутины, GC, память) собираются автоматически в opt-out режиме Go 1.26.

---

## CLI — справочник команд

| Команда | Описание |
|---|---|
| `specto server` | Запустить HTTP-сервер |
| `specto seed` | Наполнить БД стартовыми данными из кода/JSON |
| `specto migrate` | Применить миграции базы данных |
| `specto --help` | Полный список команд |

---

## 🗺 Roadmap

Разработка разбита на шесть последовательных фаз. Каждая фаза — самодостаточный вертикальный срез, который можно проверить и протестировать до перехода к следующей.

### Фаза 1 — Фундамент проекта

- [ ] Инициализация модуля: `go mod init github.com/rxritet/specto`
- [ ] Настройка структуры каталогов по Server Project Layout (Алекс Эдвардс)
- [ ] Написание `magefile.go` с таргетами `dev`, `prod`, `test`, `seed`
- [ ] Настройка Cobra CLI: корневая команда + команда `server`
- [ ] Конфигурация приложения через переменные окружения (без сторонних библиотек)
- [ ] Базовый HTTP-сервер через `net/http` — `GET /health` возвращает `200 OK`
- [ ] Настройка `slog` + `otelslog` как единой системы логирования в `main.go`
- [ ] Критерий готовности: `mage dev` запускает сервер, `/health` отвечает, логи структурированы

### Фаза 2 — Слой данных

- [ ] Определение доменных моделей: `User`, `Task` с JSON-аннотациями
- [ ] Написание интерфейсов репозиториев (`UserRepository`, `TaskRepository`)
- [ ] Реализация механизма `txFromContext` — передача `*sql.Tx` через `context.Context`
- [ ] Реализация BoltDB-провайдера (Pure Go, для локальной разработки)
- [ ] Реализация PostgreSQL-провайдера (для production)
- [ ] Написание SQL-миграций и команды `specto migrate`
- [ ] Написание `fixtures.sql` для воспроизводимого состояния БД в тестах
- [ ] Критерий готовности: оба провайдера проходят один и тот же набор unit-тестов через интерфейс

### Фаза 3 — Бизнес-логика (Service Objects)

- [ ] Реализация `TaskService` с методами: `Create`, `GetByID`, `List`, `Update`, `Delete`
- [ ] Реализация `UserService` с методами: `Register`, `Authenticate`
- [ ] Внедрение зависимостей через интерфейсы (не конкретные типы)
- [ ] Определение кастомных типов ошибок (`ErrNotFound`, `ErrConflict`, `ErrUnauthorized`)
- [ ] Использование `errors.AsType` для типобезопасных проверок во всех хендлерах
- [ ] Реализация функции агрегации с SIMD-путём и pure-Go fallback
- [ ] Написание unit-тестов для каждого метода сервиса (TDD)
- [ ] Критерий готовности: 100% покрытие методов сервисов unit-тестами; никакой HTTP-зависимости в пакете `service`

### Фаза 4 — Транспортный слой и UI

- [ ] Реализация HTTP-роутера через `net/http` (Go 1.22+ pattern matching)
- [ ] Написание хендлеров: декодирование запроса → вызов сервиса → ответ (без логики)
- [ ] Написание базовых HTML-шаблонов (`layout.html`, `tasks/index.html`, `tasks/show.html`)
- [ ] Реализация пакета `html` с Decorator-типами (`html.Task`, `html.User`)
- [ ] Настройка `go:embed` для шаблонов и статики
- [ ] Реализация разделения `dev`/`prod` сборки в `magefile.go` (диск vs embed)
- [ ] Middleware: логирование запросов, обработка паники, заголовки безопасности
- [ ] Критерий готовности: полный CRUD по задачам через браузер; шаблоны обновляются без перекомпиляции в `mage dev`

### Фаза 5 — Тестирование и качество

- [ ] Написание интеграционных тестов через Testcontainers-Go (PostgreSQL)
- [ ] Загрузка `fixtures.sql` перед каждым интеграционным тестом
- [ ] Написание fuzz-тестов для парсеров пользовательского ввода (`FuzzParseDuration`, `FuzzParseTaskDescription`)
- [ ] Настройка линтера `golangci-lint` с конфигом `.golangci.yml`
- [ ] Написание GitHub Actions workflow: `lint` + `test` + `build` на каждый push
- [ ] Настройка таргета `mage test` для запуска unit + integration тестов
- [ ] Критерий готовности: CI зелёный на всех коммитах; fuzz corpus без краш-кейсов

### Фаза 6 — Деплой и эксплуатация

- [ ] Написание команды `specto seed` для наполнения БД из кода/JSON (Cobra)
- [ ] Написание `systemd` unit-файла (`/etc/systemd/system/specto.service`)
- [ ] Настройка деплой-скрипта: `mage prod` → `rsync` → `systemctl restart`
- [ ] Настройка OpenTelemetry: экспорт трейсов и метрик в OTLP-эндпоинт
- [ ] Проверка автоматического сбора runtime-метрик Go 1.26 (opt-out модель)
- [ ] Документирование процедуры деплоя в `docs/deployment.md`
- [ ] Критерий готовности: бинарник разворачивается на чистом сервере одной командой; метрики видны в Grafana/Jaeger

---

## Участие в разработке

1. Форкни репозиторий
2. Создай ветку фичи: `git checkout -b feat/your-feature`
3. Пиши тесты первыми (TDD)
4. Убедись, что `mage test` проходит без ошибок
5. Открой Pull Request

---

## Лицензия

MIT © 2026 Radmir Abraev
