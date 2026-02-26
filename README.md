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

## Участие в разработке

1. Форкни репозиторий
2. Создай ветку фичи: `git checkout -b feat/your-feature`
3. Пиши тесты первыми (TDD)
4. Убедись, что `mage test` проходит без ошибок
5. Открой Pull Request

---

## Лицензия

MIT © 2026 Radmir Abraev
