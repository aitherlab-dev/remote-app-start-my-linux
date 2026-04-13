# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Текущее состояние

Сервер готов и прошёл end-to-end руками: поднимается, пара по PIN, список приложений, реальный запуск Obsidian с пользовательским vault'ом — всё работает. Фаза 5 закрыта локально (не запушено). Осталось приклеить Android-клиент — это Фаза A из плана.

- Полный план: `/home/sasha/.claude/plans/bubbly-greeting-knuth.md` (секции A0..A6)
- Снимок прогресса: `docs/PROGRESS.md`
- Источник требований: `docs/ТЗ.md`

Перед написанием кода сверяться с планом и ТЗ.

### Что сервер умеет прямо сейчас (Фаза 5 завершена)

- HTTPS на `:8443`, self-signed ECDSA-сертификат живёт в `~/.config/remotelauncher/`
- Парсит `.desktop` файлы (XDG-aware), отдаёт список приложений и иконки
- `POST /api/apps/{id}/launch` запускает реальную программу, отвязанную от сервера
- `running` статус через reaping SIGCHLD
- PIN-pairing при старте, Bearer-tokens (SHA-256 hash в `tokens.json`)
- Rate-limit на `/api/pair` (5 попыток/IP/10 минут, 429 + Retry-After)
- `/api/status` отдаёт `version` (через ldflags) и `cert_fingerprint` для SPKI-pinning
- Конфиг: TOML в `~/.config/remotelauncher/config.toml`, порядок defaults→file→env→flags, пример в `server/config.example.toml`
- Structured logging через `log/slog`, уровень и формат (text/json) из конфига
- systemd user-unit + `make install/uninstall/package`, release-тарбол собирается через `make package`
- `make fmt vet lint test build integration` зелёное, покрытие ядра 92–100%, http/auth ≥88%

### Что осталось

**Фаза A — Android-клиент** (Kotlin, Compose, minSdk 26, targetSdk 35):

- **A0** — инфра: JDK + Android Studio + SDK, скелет проекта, Hello World APK на реальном устройстве
- **A1** — Ktor API слой + экран Connect + DataStore настроек
- **A2** — Pairing-экран + EncryptedSharedPreferences для токена
- **A3** — Сетка приложений + иконки через Coil 3
- **A4** — Tap → launch → snackbar (первый момент когда продукт оправдывает себя)
- **A5** — **Критичный этап:** SPKI-pinning, TOFU, сверка fingerprint с `/api/status` (без этого релиз нельзя)
- **A6** — Release-сборка, подпись, keystore вне репо, APK для GitHub Releases

## Что за проект

**RemoteLauncher** — лаунчер программ на Linux-десктопе с Android-смартфона через интернет или локальную сеть. Не VNC/remote desktop: пользователь видит сетку приложений с иконками, нажимает — программа запускается на ПК. Две части:

- **Сервер** (Go, Linux) — один бинарник без зависимостей, парсит `.desktop` файлы из `/usr/share/applications/` и `~/.local/share/applications/`, отдаёт список и иконки по REST, запускает/останавливает процессы, статус через WebSocket, опционально mDNS/Avahi.
- **Клиент** (Kotlin, Android) — Material Design, подключение по адресу (домен/IP + порт), автообнаружение в локалке через mDNS (бонус), APK через GitHub Releases.

## Структура

```
REMOTE-MY-LINUX/
├── server/      # Go-сервер (готов, MVP)
├── packaging/   # systemd unit + install.sh/uninstall.sh
├── android/     # Kotlin-клиент (ещё не создан, появится в A0.2)
└── docs/        # ТЗ, PROGRESS, архитектурные заметки
```

## Ключевые архитектурные решения (из ТЗ)

- **Безопасность**: TLS обязателен, первое подключение — pairing через PIN-код, далее токен. Rate limiting на авторизацию, опциональный whitelist IP.
- **Подключение снаружи** — выбор пользователя: проброс порта + DDNS, reverse proxy, Tailscale/ZeroTier, Cloudflare Tunnel. Код сервера не должен завязываться на конкретный способ.
- **mDNS** — только бонус для локалки, основной режим — прямое подключение по адресу.
- **API** — REST для списка/запуска/остановки/иконок, WebSocket `/ws` для статуса в реальном времени. Полный список эндпоинтов в `docs/ТЗ.md`.

## MVP vs будущие версии

MVP (v1): список приложений с иконками, подключение по адресу, запуск по нажатию, PIN-pairing, HTTPS. WebSocket-статус, остановка, категории, mDNS, виджеты — всё это v2+. При работе над MVP не тащить фичи из следующих версий без явного запроса.

## Рабочий процесс

- **Этапы идут по плану**, каждый — отдельная команда через `TeamCreate` (developer + reviewer, опционально researcher). Не через `Agent` tool с несколькими вызовами.
- **Никаких half-done** — каждый этап оставляет зелёные тесты и работоспособный артефакт (собираемый бинарь/APK).
- **Правки по ревью делаются в том же этапе**, не переносятся в следующий.
- **Коммиты в Conventional Commits** (`feat(scope): ...`) с обязательным trailer'ом `Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>`.
- **Ничего не пушить в remote** без явного разрешения пользователя.

## Про пользователя

Пользователь не программист — не показывать листинги кода в ответах, объяснять на уровне «что происходит» и «что делать», а не «вот такая функция». Общение на русском, неформальное.
