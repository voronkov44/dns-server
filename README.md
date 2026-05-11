# dns-manager

`dns-manager` — клиент-серверное приложение на Go для управления DNS-серверами на удалённой машине.

Серверное приложение предоставляет REST API и управляет DNS через файл `resolv.conf`. CLI-клиент `dnsctl` не работает с файлом напрямую: он отправляет HTTP-запросы на сервер и получает результат через REST API.

Для безопасной локальной разработки можно использовать локальный файл:

```text
./resolv.conf
```

Для реального управления DNS на машине можно указать системный файл:

```text
/etc/resolv.conf
```

> [!WARNING]
> При работе с `/etc/resolv.conf` приложение действительно будет изменять DNS-настройки машины, на которой запущен server.

---

## Стек

- Go 1.26
- REST API на стандартном `net/http`
- CLI на [`github.com/spf13/cobra`](https://github.com/spf13/cobra)
- Конфигурация через [`github.com/ilyakaznacheev/cleanenv`](https://github.com/ilyakaznacheev/cleanenv)
- Логирование через `log/slog`
- Makefile для автоматизации
- Unit-тесты и smoke-тесты
- Работа с `resolv.conf` через файловый repository

---

## Установка и запуск проекта

### 1. Клонирование

```bash
git clone https://github.com/voronkov44/dns-manager.git
cd dns-manager
```

---

### 2. Настройка окружения

Создайте локальный `.env` на основе примера:

```bash
cp .env.example .env
```

В `.env.example` используется безопасный локальный вариант:

```env
DNS_MANAGER_RESOLV_CONF_PATH=./resolv.conf
```

Для локальной проверки лучше использовать именно `./resolv.conf`, чтобы не менять системные DNS-настройки.

Для реального управления DNS на машине можно указать:

```env
DNS_MANAGER_RESOLV_CONF_PATH=/etc/resolv.conf
```

---

### 3. Запуск сервера через Makefile

```bash
make run-server
```

Makefile подхватывает `.env`, если он есть. Если `.env` нет, используются значения по умолчанию из Makefile.

Для локальной разработки Makefile по умолчанию использует:

```env
DNS_MANAGER_RESOLV_CONF_PATH=./resolv.conf
```

---

### 4. Ручной запуск сервера через Go

```bash
go run ./cmd/server
```

При ручном запуске приложение читает конфигурацию через `config.LoadConfig()`:

- если рядом есть `.env`, значения берутся из него;
- если `.env` нет, используются переменные окружения и значения по умолчанию из `config.go`.

В `config.go` дефолтный путь к `resolv.conf`:

```env
DNS_MANAGER_RESOLV_CONF_PATH=/etc/resolv.conf
```

Поэтому для безопасного ручного запуска лучше явно указать локальный файл:

```bash
DNS_MANAGER_RESOLV_CONF_PATH=./resolv.conf go run ./cmd/server
```

---

## Использование REST API

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/healthz` | Проверка работоспособности сервера |
| `GET` | `/dns` | Получить список DNS-серверов |
| `POST` | `/dns` | Добавить DNS-сервер |
| `DELETE` | `/dns?server=<ip>` | Удалить DNS-сервер |

---

### Healthcheck

```bash
curl http://localhost:8080/healthz
```

Ответ:

```json
{"status":"ok"}
```

---

### Получить список DNS

```bash
curl http://localhost:8080/dns
```

Ответ:

```json
{
  "servers": ["8.8.8.8", "1.1.1.1"]
}
```

---

### Добавить DNS

```bash
curl -X POST http://localhost:8080/dns \
  -H "Content-Type: application/json" \
  -d '{"server":"8.8.8.8"}'
```

Пример ответа:

```json
{
  "servers": ["8.8.8.8"]
}
```

---

### Удалить DNS

Основной вариант удаления — через query parameter:

```bash
curl -X DELETE "http://localhost:8080/dns?server=8.8.8.8"
```

Также поддерживается fallback через JSON body:

```bash
curl -X DELETE http://localhost:8080/dns \
  -H "Content-Type: application/json" \
  -d '{"server":"8.8.8.8"}'
```

---

### HTTP-статусы

| Status | Когда используется |
| --- | --- |
| `200 OK` | Успешный healthcheck, получение списка или удаление DNS |
| `201 Created` | DNS-сервер успешно добавлен |
| `400 Bad Request` | Невалидный JSON, пустой body, неизвестные поля или некорректный IP |
| `404 Not Found` | DNS-сервер не найден при удалении или запрошен неизвестный endpoint |
| `409 Conflict` | DNS-сервер уже существует |
| `500 Internal Server Error` | Внутренняя ошибка сервера |

---

## Использование CLI

CLI называется `dnsctl`.

CLI не изменяет `resolv.conf` напрямую. Он вызывает REST API сервера, а уже сервер выполняет работу с файлом.

---

### Команды через `go run`

```bash
go run ./cmd/dnsctl --help
go run ./cmd/dnsctl list
go run ./cmd/dnsctl add 8.8.8.8
go run ./cmd/dnsctl delete 8.8.8.8
```

Для команды удаления также доступны alias-команды:

```bash
go run ./cmd/dnsctl remove 8.8.8.8
go run ./cmd/dnsctl rm 8.8.8.8
```

---

### Команды через Makefile

```bash
make run-cli
make cli-list
make cli-add DNS=8.8.8.8
make cli-delete DNS=8.8.8.8
```

---

### Сборка бинарников

```bash
make build
```

После сборки бинарники появятся в директории `bin/`:

```bash
./bin/dns-manager-server
./bin/dnsctl list
./bin/dnsctl add 8.8.8.8
./bin/dnsctl delete 8.8.8.8
```

---

### Установка CLI в GOPATH/bin

```bash
make install-cli
```

После этого, если `$(go env GOPATH)/bin` есть в `PATH`, можно использовать CLI как обычную команду:

```bash
dnsctl list
dnsctl add 8.8.8.8
dnsctl delete 8.8.8.8
```

---

### Подключение CLI к серверу

Адрес сервера можно передать флагом:

```bash
dnsctl --server http://localhost:8080 list
```

Или через переменную окружения:

```env
DNS_MANAGER_SERVER_URL=http://localhost:8080
```

Важно различать две переменные:

| Variable | Назначение |
| --- | --- |
| `DNS_MANAGER_HTTP_ADDR` | Адрес, на котором server слушает входящие HTTP-запросы |
| `DNS_MANAGER_SERVER_URL` | URL, по которому CLI подключается к server |

Пример:

```env
DNS_MANAGER_HTTP_ADDR=:8080
DNS_MANAGER_SERVER_URL=http://localhost:8080
```

---

## Makefile

| Command | Description |
| --- | --- |
| `make run-server` | Запустить сервер |
| `make run-cli` | Показать help CLI |
| `make cli-list` | Получить список DNS |
| `make cli-add DNS=8.8.8.8` | Добавить DNS |
| `make cli-delete DNS=8.8.8.8` | Удалить DNS |
| `make build` | Собрать server и CLI |
| `make build-server` | Собрать только server |
| `make build-cli` | Собрать только CLI |
| `make install-cli` | Установить `dnsctl` в `GOPATH/bin` |
| `make test` | Запустить все тесты |
| `make test-unit` | Запустить unit-тесты |
| `make test-smoke` | Запустить smoke-тесты |
| `make test-verbose` | Запустить все тесты с подробным выводом |
| `make test-unit-verbose` | Запустить unit-тесты с подробным выводом |
| `make test-smoke-verbose` | Запустить smoke-тесты с подробным выводом |
| `make lint` | Запустить `golangci-lint` |
| `make check` | Запустить `fmt`, `tidy`, `lint` и `test` |
| `make fmt` | Форматировать Go-код |
| `make tidy` | Обновить `go.mod` и `go.sum` |
| `make clean` | Удалить `bin/` |

---

## Конфигурация

Пример конфигурации находится в `.env.example`.

| Variable | Default | Description |
| --- | --- | --- |
| `DNS_MANAGER_HTTP_ADDR` | `:8080` | Адрес HTTP-сервера |
| `DNS_MANAGER_RESOLV_CONF_PATH` | `/etc/resolv.conf` в `config`, `./resolv.conf` в `.env.example` и Makefile | Путь к управляемому `resolv.conf` |
| `DNS_MANAGER_READ_HEADER_TIMEOUT` | `5s` | Таймаут чтения HTTP-заголовков |
| `DNS_MANAGER_SHUTDOWN_TIMEOUT` | `10s` | Таймаут graceful shutdown |
| `DNS_MANAGER_LOG_LEVEL` | `info` | Уровень логирования |
| `DNS_MANAGER_LOG_FILE_PATH` | `logs/dns-server.log` | Файл логов |
| `DNS_MANAGER_SERVER_URL` | `http://localhost:8080` | URL сервера для CLI |

Конфигурация читается через `cleanenv`.

Если `.env` есть, приложение читает значения из него. Если `.env` нет, применяются значения из переменных окружения и `env-default` tags в `config.go`.

Файл `.env` не коммитится, потому что это локальная конфигурация окружения. Файл `.env.example` коммитится как безопасный пример.

---

## Структура проекта

```text
dns-manager/
├── cmd
│   ├── dnsctl
│   │   └── main.go
│   └── server
│       └── main.go
├── config
│   └── config.go
├── internal
│   ├── cli
│   │   ├── add.go
│   │   ├── delete.go
│   │   ├── list.go
│   │   └── root.go
│   ├── client
│   │   ├── client.go
│   │   └── client_test.go
│   └── dns
│       ├── handler.go
│       ├── handler_test.go
│       ├── model.go
│       ├── payload.go
│       ├── repository.go
│       ├── repository_test.go
│       ├── service.go
│       └── service_test.go
├── pkg
│   ├── logger
│   │   └── logger.go
│   ├── middleware
│   │   ├── chain.go
│   │   ├── common.go
│   │   ├── cors.go
│   │   ├── logs.go
│   │   └── recover.go
│   ├── req
│   │   ├── decode.go
│   │   ├── decode_test.go
│   │   └── handle.go
│   └── res
│       ├── res.go
│       └── res_test.go
├── tests
│   └── smoke
│       └── smoke_test.go
├── .env.example
├── .gitignore
├── Makefile
├── go.mod
└── go.sum
```

`bin/`, `logs/` и локальный `resolv.conf` являются runtime/build artifacts и игнорируются через `.gitignore`.

---

## Архитектура

Проект разделён на несколько слоёв:

```text
cmd/server       — запуск HTTP-сервера
cmd/dnsctl       — запуск CLI
config           — чтение конфигурации
internal/dns     — основная доменная логика DNS manager
internal/client  — HTTP client для CLI
internal/cli     — команды Cobra CLI
pkg/logger       — инициализация slog logger
pkg/middleware   — HTTP middleware
pkg/req          — JSON decode helpers
pkg/res          — JSON response helpers
tests/smoke      — smoke/integration tests
```

Основной модуль доменной логики находится в `internal/dns`:

| File | Responsibility |
| --- | --- |
| `handler.go` | HTTP-слой: принимает запросы, вызывает service и отдаёт JSON |
| `service.go` | Бизнес-логика и валидация IP через `net/netip` |
| `repository.go` | Работа с файлом `resolv.conf` |
| `model.go` | Внутренняя модель DNS-сервера |
| `payload.go` | Request/response DTO для REST API |

Основной flow приложения:

```text
CLI / curl
  -> REST API
  -> handler
  -> service
  -> repository
  -> resolv.conf
```

---

## Особенности реализации

### Работа с resolv.conf

DNS-сервер хранится в файле как строка:

```text
nameserver 8.8.8.8
```

`FileRepository` читает только строки, начинающиеся с `nameserver`. Остальные строки файла сохраняются: комментарии, `search`, `options` и другие настройки.

Пример файла:

```text
# local resolver config
search example.local
nameserver 1.1.1.1
options rotate
```

При добавлении нового DNS-сервера repository дописывает новую строку `nameserver ...`, не удаляя существующие комментарии и настройки.

Невалидные `nameserver`-адреса игнорируются при парсинге списка. Если файла нет, repository создаёт его автоматически.

---

### Валидация IP

Валидация IP-адресов сделана через `net/netip`, а не через regex.

Поддерживаются:

- IPv4
- IPv6
- нормализация IPv6

Например, эти две записи означают один и тот же адрес:

```text
2001:db8::1
2001:0db8:0000:0000:0000:0000:0000:0001
```

После нормализации они сравниваются как одинаковые. Это позволяет корректно находить дубли IPv6 даже в разных текстовых формах.

---

### Mutex и конкурентные запросы

HTTP handler-и могут выполняться параллельно.

Операции `AddServer` и `DeleteServer` являются read-modify-write операциями:

```text
прочитать файл -> изменить список -> записать файл
```

Без синхронизации два параллельных запроса могут прочитать старую версию файла и перетереть изменения друг друга.

Поэтому `FileRepository` использует `sync.Mutex`. Mutex защищает операции внутри одного процесса сервера.

> [!NOTE]
> Если запустить несколько процессов сервера на один и тот же `resolv.conf`, одного `sync.Mutex` уже недостаточно. Для такого сценария нужен file lock. В текущем решении предполагается один экземпляр server-приложения.

---

### Atomic write

Простая запись через `os.WriteFile` может повредить файл, если процесс завершится во время записи. Например, можно получить пустой или частично записанный `resolv.conf`.

Поэтому repository использует более безопасный подход:

1. создаёт временный файл рядом с целевым;
2. записывает новое содержимое во временный файл;
3. вызывает `Sync`;
4. закрывает файл;
5. делает `rename` временного файла в целевой;
6. синхронизирует директорию.

Также учитывается symlink через `filepath.EvalSymlinks`, потому что `/etc/resolv.conf` на Linux часто может быть ссылкой.

---

### Graceful shutdown

Сервер слушает `os.Interrupt` и `SIGTERM`.

При завершении он:

- перестаёт принимать новые запросы;
- даёт активным запросам время завершиться;
- использует таймаут из `DNS_MANAGER_SHUTDOWN_TIMEOUT`.

Это хорошая практика даже для небольшого HTTP-сервера.

---

### Logging

Логирование реализовано через `log/slog`.

Логи выводятся:

- в терминал;
- в файл `logs/dns-server.log`.

Уровень логирования настраивается через:

```env
DNS_MANAGER_LOG_LEVEL=info
```

Файл логов полезен для диагностики, если сервер работал неправильно.

Логи не пишутся атомарно, потому что это диагностическая информация, а не состояние системы. Для состояния системы атомарность реализована именно в записи `resolv.conf`.

---

### Middleware

В проекте есть набор HTTP middleware:

| Middleware | Description |
| --- | --- |
| `Logging` | Логирует HTTP-запросы: статус, метод, путь, bytes, duration и query |
| `Recover` | Ловит panic, пишет stack trace в лог и возвращает 500 |
| `CORS` | Добавляет базовую CORS-поддержку |
| `Chain` | Собирает middleware в цепочку |
| `WrapperWriter` | Позволяет получить status code, bytes и отследить вызов `WriteHeader` |

---

### req и res helpers

Пакет `pkg/req` отвечает за строгий decode JSON body.

Он:

- запрещает unknown fields;
- запрещает несколько JSON-объектов в одном body;
- возвращает ошибку на пустой body.

Пакет `pkg/res` унифицирует JSON-ответы:

```text
res.Json(w, payload, http.StatusOK)
```

Он выставляет `Content-Type: application/json`, статус ответа и кодирует body в JSON.

---

### CLI

CLI сделан через Cobra.

Основные команды:

```bash
dnsctl list
dnsctl add 8.8.8.8
dnsctl delete 8.8.8.8
```

CLI не работает с `resolv.conf` напрямую. Он отправляет запросы на REST API сервера.

`--help` генерируется Cobra:

```bash
dnsctl --help
dnsctl add --help
dnsctl delete --help
```

---

## Тестирование

Основные команды:

```bash
make test
make test-unit
make test-smoke
make test-verbose
make test-smoke-verbose
make check
```

---

### Unit-тесты

Unit-тесты проверяют отдельные компоненты приложения.

#### `repository_test.go`

Проверяет файловый repository:

- чтение строк `nameserver`;
- добавление DNS-сервера;
- удаление DNS-сервера;
- отклонение дублей IPv4;
- отклонение дублей IPv6 в разных текстовых формах;
- сохранение комментариев, `search`, `options` и других строк;
- создание отсутствующего `resolv.conf`.

#### `service_test.go`

Проверяет бизнес-логику:

- валидацию IPv4;
- валидацию IPv6;
- нормализацию IPv6;
- отклонение невалидных адресов;
- вызов repository только после успешной валидации.

#### `handler_test.go`

Проверяет HTTP-слой:

- healthcheck;
- получение списка DNS;
- добавление DNS;
- удаление DNS через query parameter;
- fallback удаления через JSON body;
- invalid JSON;
- unknown fields;
- duplicate/not found/bad request сценарии;
- корректные HTTP-статусы.

#### `client_test.go`

Проверяет HTTP client для CLI:

- `ListServers`;
- `AddServer`;
- `DeleteServer`;
- использование query parameter для delete;
- обработку API errors;
- обработку malformed response.

#### `pkg/req` и `pkg/res`

Проверяют общие helpers:

- ошибку на пустой body;
- запрет unknown fields;
- запрет нескольких JSON-объектов в body;
- единый JSON response helper.

---

### Smoke-тесты

Smoke-тесты проверяют полный путь приложения

Важные особенности smoke-тестов:

- используется `httptest.Server`;
- используется временный `resolv.conf` через `t.TempDir()`;
- настоящий `/etc/resolv.conf` никогда не читается и не изменяется;
- проверяется не одна функция, а связка компонентов целиком.

| Test | Что проверяет |
| --- | --- |
| `TestSmokeDNSLifecycle` | Полный цикл `list -> add -> list -> delete` |
| `TestSmokeDuplicateAddReturnsConflict` | Дубликат возвращает `409 Conflict` |
| `TestSmokeDeleteMissingServerReturnsNotFound` | Удаление отсутствующего DNS возвращает `404 Not Found` |
| `TestSmokeInvalidIPReturnsBadRequest` | Невалидные IPv4/IPv6 возвращают `400 Bad Request` |
| `TestSmokeMissingResolvConfIsCreated` | Отсутствующий `resolv.conf` создаётся автоматически |
| `TestSmokeParallelAddRequestsDoNotOverwriteEachOther` | Параллельные add не перетирают изменения |
| `TestSmokeParallelDuplicateAddOnlyCreatesOneEntry` | Параллельное добавление дубля создаёт только одну запись |
| `TestSmokeParallelDeleteRequestsDoNotCorruptFile` | Параллельные delete не повреждают файл |
| `TestSmokeIPv6DuplicateNormalization` | IPv6-дубли ловятся после нормализации |

---

## Линт и проверка качества

Запуск линтера:

```bash
make lint
```

Полная проверка проекта:

```bash
make check
```

`make check` запускает:

```text
go fmt ./...
go mod tidy
golangci-lint run ./...
go test ./...
```

Для `make lint` должен быть установлен `golangci-lint`.

---

## Важные замечания

Не запускайте сервер с `DNS_MANAGER_RESOLV_CONF_PATH=/etc/resolv.conf`, если не хотите менять системный DNS.

Для локальной проверки используйте:

```env
DNS_MANAGER_RESOLV_CONF_PATH=./resolv.conf
```

Для записи в `/etc/resolv.conf` могут потребоваться повышенные права.

В некоторых Linux-системах `/etc/resolv.conf` может управляться `systemd-resolved`, `NetworkManager` или другим системным сервисом. В таком случае внешняя система может перезаписывать файл.

В рамках задания предполагается, что никто кроме приложения не пишет в управляемый `resolv.conf`.

---

## Зависимости

Для работы проекта нужны:

- Go 1.26+
- `golangci-lint` для `make lint`
- зависимости из `go.mod`, включая:
    - `github.com/spf13/cobra`
    - `github.com/ilyakaznacheev/cleanenv`

Установка Go-зависимостей:

```bash
go mod download
```

Пример установки `golangci-lint` на macOS:

```bash
brew install golangci-lint
```

На других ОС способ установки зависит от окружения.

---

## Итог

Проект покрывает основные требования задания:

- REST API для управления DNS-серверами;
- CLI-клиент `dnsctl`;
- работа с `resolv.conf`;
- корректная обработка ошибок;
- healthcheck;
- логирование;
- конфигурация через .env;
- Makefile для запуска, сборки и тестирования;
- unit-тесты и smoke-тесты;
- безопасные тесты, которые работают только с временным `resolv.conf`.
