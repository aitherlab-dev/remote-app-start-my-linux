# RemoteLauncher Server

Go-сервер для RemoteLauncher — запускает Linux-приложения по запросу с Android-клиента.

## Команды

| Команда           | Что делает                                       |
|-------------------|--------------------------------------------------|
| `make build`      | сборка бинаря в `bin/remotelauncher`             |
| `make run`        | сборка и запуск                                  |
| `make test`       | прогон всех тестов                               |
| `make fmt`        | форматирование через `gofumpt` + `goimports`     |
| `make vet`        | `go vet ./...`                                   |
| `make lint`       | `golangci-lint run ./...`                        |
| `make cover`      | тесты с покрытием + сводка в консоль             |
| `make cover-html` | HTML-отчёт покрытия в `coverage.html`            |
| `make scan-debug` | сканирует ваши реально установленные приложения и печатает количество найденных. Полезно убедиться что парсер видит вашу систему. |
| `make clean`      | удалить артефакты сборки и покрытия              |

## Dev-инструменты

Линтеры и форматтеры — dev-зависимости, в бинарь сервера не входят. Установка в `~/go/bin`:

```sh
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install mvdan.cc/gofumpt@latest
go install golang.org/x/tools/cmd/goimports@latest
```

Если `~/go/bin` не в `PATH`, Makefile вызывает их через `$(GOBIN)`.

## Структура

```
server/
├── cmd/
│   └── remotelauncher/   # точка входа (main)
└── internal/             # внутренние пакеты (появятся в следующих этапах)
```
