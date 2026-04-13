# RemoteLauncher Server

Go-сервер для RemoteLauncher — запускает Linux-приложения на десктопе по
запросу с Android-клиента. Один бинарник без внешних зависимостей,
TLS-обязателен, pairing через PIN, токены.

## Быстрый старт

```sh
cd server
make build
./bin/remotelauncher
```

При первом запуске сервер:

1. создаёт TLS-сертификат в `$XDG_CONFIG_HOME/remotelauncher/` (или
   `~/.config/remotelauncher/`), если его ещё нет;
2. печатает в stdout pairing PIN вида `Pairing PIN: NNNNNN (valid for 10m0s)`;
3. слушает HTTPS на `:8443`.

PIN нужен только для первого сопряжения с клиентом — в обмен на правильный
PIN клиент получает Bearer-токен, которым подписывает все последующие
запросы.

## Файлы и каталоги

| Путь                                            | Что                                  |
|-------------------------------------------------|--------------------------------------|
| `$XDG_CONFIG_HOME/remotelauncher/config.toml`   | опциональный конфиг                  |
| `$XDG_CONFIG_HOME/remotelauncher/cert.pem`      | TLS-сертификат                       |
| `$XDG_CONFIG_HOME/remotelauncher/key.pem`       | приватный ключ                       |
| `$XDG_CONFIG_HOME/remotelauncher/tokens.json`   | выданные клиенту токены              |
| `$HOME/.local/bin/remotelauncher`               | бинарь (после `make install`)        |
| `$HOME/.config/systemd/user/remotelauncher.service` | systemd user unit                |

`$XDG_CONFIG_HOME` дефолтится в `~/.config` если переменная не задана.

## Конфигурация

Все настройки имеют компилированные дефолты — без конфига сервер работает.
Пример со всеми ключами и комментариями — [`config.example.toml`](config.example.toml).

Приоритет источников (позднее перекрывает раннее):

```
defaults  →  TOML-файл  →  переменные окружения (REMOTELAUNCHER_*)  →  флаги командной строки
```

Путь к TOML по умолчанию — `$XDG_CONFIG_HOME/remotelauncher/config.toml`.
Отсутствие файла — не ошибка.

## API (ключевое)

Все эндпоинты кроме `POST /api/pair` требуют заголовок `Authorization: Bearer <token>`.

| Метод и путь                  | Назначение                                   |
|-------------------------------|----------------------------------------------|
| `GET  /api/status`            | версия, uptime, fingerprint сертификата      |
| `GET  /api/apps`              | список `.desktop`-приложений                 |
| `GET  /api/apps/{id}/icon`    | PNG/SVG иконки приложения                    |
| `POST /api/apps/{id}/launch`  | запустить приложение                         |
| `POST /api/pair`              | обменять PIN на Bearer-токен (rate-limited)  |

Полный список — в `docs/ТЗ.md`.

## Установка как systemd user-сервис

```sh
cd server
make install          # собирает бинарь, ставит unit, enable --now
systemctl --user status remotelauncher
journalctl --user -u remotelauncher -f
```

`make install` и `make uninstall` дергают скрипты из `packaging/`. Скрипты
идемпотентны, повторный запуск ничего не ломает. `make uninstall` убирает
бинарь и unit, но **не трогает** `~/.config/remotelauncher/` — туда кладутся
сертификаты и токены, их пользователь удаляет вручную если нужна чистая
инсталляция.

## Сборка релизного архива

```sh
make package
```

Создаёт `dist/remotelauncher-<version>-linux-amd64.tar.gz` с бинарём,
systemd unit, install/uninstall-скриптами, `config.example.toml` и
`README.md`. `<version>` берётся из `git describe --tags --always --dirty`.

## Версия

Версия вшивается в бинарник через `-ldflags "-X main.Version=…"` на этапе
сборки. Посмотреть запущенную версию:

```sh
curl -sk https://localhost:8443/api/status | jq .version
```

В dev-сборке без git-тегов возвращается короткий SHA с суффиксом `-dirty`,
если есть незакоммиченные правки. Если git вообще недоступен — `dev`.

## Команды Makefile

| Команда           | Что делает                                               |
|-------------------|----------------------------------------------------------|
| `make build`      | сборка бинаря в `bin/remotelauncher` (с ldflags version) |
| `make run`        | сборка и запуск                                          |
| `make test`       | прогон всех юнит-тестов                                  |
| `make integration`| интеграционные тесты (build tag `integration`)           |
| `make fmt`        | форматирование через `gofumpt` + `goimports`             |
| `make vet`        | `go vet ./...`                                           |
| `make lint`       | `golangci-lint run ./...`                                |
| `make cover`      | тесты с покрытием + сводка                               |
| `make cover-html` | HTML-отчёт покрытия в `coverage.html`                    |
| `make scan-debug` | сканирует реальные `.desktop` на машине                  |
| `make icons-debug`| пробует найти иконки реально установленных приложений    |
| `make install`    | поставить под $HOME как systemd user-сервис              |
| `make uninstall`  | убрать бинарь и unit                                     |
| `make package`    | собрать релизный tar.gz в `dist/`                        |
| `make clean`      | удалить артефакты сборки                                 |

## Dev-зависимости

Линтеры и форматтеры в бинарь сервера не входят:

```sh
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install mvdan.cc/gofumpt@latest
go install golang.org/x/tools/cmd/goimports@latest
```
