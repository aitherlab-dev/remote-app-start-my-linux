# RemoteLauncher — прогресс

Снимок состояния для новой сессии Claude Code. Перед стартом прочитай:
1. Этот файл — короткая картина «где мы»
2. `/home/sasha/.claude/plans/bubbly-greeting-knuth.md` — полный план
3. `docs/ТЗ.md` — исходное техническое задание
4. `CLAUDE.md` в корне

## Текущее состояние: Фаза 4 завершена

19 коммитов в `main`. Последний: `131b673 feat(auth): rate limit /api/pair against brute force`.

**Сервер полностью функциональный и защищённый**, готов к `make install`-этапу (который ещё не сделан).

### Что сервер умеет прямо сейчас

- HTTPS на `:8443` (ECDSA P-256, self-signed, срок 10 лет)
- Сертификат живёт в `$XDG_CONFIG_HOME/remotelauncher/cert.pem` + `key.pem`
- Парсит все `.desktop` файлы (XDG-aware: user перекрывает system)
- Отдаёт список приложений и иконки с поддержкой наследования тем и KDE-layout
- Запускает программы через `POST /api/apps/{id}/launch` с отсоединением от сервера
- Reaps zombie процессы, `running` в `/api/apps` корректно отражает статус
- PIN-pairing при старте (stdout + log), одноразовый, TTL 10 мин
- Bearer tokens 32 байта base64url, хранится только SHA-256 hash
- Токены персистятся в `tokens.json`, живут между перезапусками
- Rate limit на `/api/pair`: 5 попыток на IP / 20 глобально / 10 минут, 429 + Retry-After
- `/api/status` возвращает `cert_fingerprint` для TLS-pinning на клиенте

### Реальный e2e проверен

Сервер реально запускает Chromium через `curl -X POST https://localhost:8443/api/apps/chromium/launch`, `running` переключается на `true`, после закрытия — обратно `false`.

## Структура коммитов по фазам

### Фаза 0 — Инфра
- `50f4835 chore: initial server scaffolding` (S0.1)
- `0f4fe44 docs: add project brief and Claude Code guidance`
- `805386f chore: add lint and coverage tooling` (S0.2)

### Фаза 1 — Ядро сервера
- `11e418d feat(desktop): parse single .desktop entry file` (S1.1)
- `830c26c feat(desktop): parse Exec field with quoting and placeholders` (S1.2)
- `e253332 feat(desktop): scan XDG .desktop directories with user-over-system priority` (S1.3)
- `5db1cfa feat(catalog): in-memory application catalog with safe reload` (S1.4)
- `be17cee feat(httpapi): /api/status and /api/apps with graceful shutdown` (S1.5)

### Фаза 2 — Иконки
- `18592cc feat(icons): XDG icon lookup by name with size and theme fallback` (S2.1)
- `c9dd43b feat(httpapi): GET /api/apps/{id}/icon with size and theme fallback` (S2.2)
- `b051f23 feat(icons): theme inheritance and KDE-style size dirs` (S2.3)

### Фаза 3 — Запуск
- `2d83db2 feat(launcher): start applications detached from the server` (S3.1)
- `3b936f1 feat(launcher): track running PIDs per application` (S3.2)
- `aac2e8f feat(httpapi): POST /api/apps/{id}/launch + running status` (S3.3)
- `41afefc fix(launcher): reap child processes to clear stale running status` (S3.4)

### Фаза 4 — Безопасность
- `159c0ab feat(tls): self-signed ECDSA cert and HTTPS on :8443` (S4.1)
- `537d044 feat(auth): PIN pairing + in-memory Bearer tokens on protected routes` (S4.2a)
- `d439a00 feat(auth): persist tokens to tokens.json` (S4.2b)
- `131b673 feat(auth): rate limit /api/pair against brute force` (S4.3)

## Что осталось

### Фаза 5 — Упаковка сервера (оставшиеся 4 этапа)

- **S5.1 — Флаги + конфиг-файл**: `internal/config` с TOML (единственная внешняя зависимость — `github.com/BurntSushi/toml`), порядок приоритетов defaults→file→env→flags, все хардкоды из main.go (порт 8443, пути серта, TTL PIN, лимиты rate-limit) в конфиг. Плотность M.
- **S5.2 — `log/slog` уровень из конфига**: slog уже используется, нужен только конфигурируемый level (debug/info/warn/error) и формат (json/text). Плотность S.
- **S5.3 — systemd user-unit + install.sh**: `packaging/remotelauncher.service`, `install.sh` копирует бинарь в `~/.local/bin/`, unit в `~/.config/systemd/user/`, enable-now. `make install/uninstall/package`. Плотность M.
- **S5.4 — README + финальный прогон**: README сервера, version через `-ldflags`, финальные тесты. Плотность S.

После S5.4 сервер считается готовым к MVP и ставится на машину через `make install`.

### Фаза A — Android-клиент

Это отдельный большой блок, начинать с установки JDK/Android Studio/SDK. Полная декомпозиция — в плане, раздел «A0…A6».

## Важные факты для новой сессии

- **Репо**: `/home/sasha/WORK/REMOTE-MY-LINUX/`
- **Go-код**: `server/`, `go.mod` имя `github.com/sasha/remotelauncher`, Go 1.26.1
- **git user**: локально `Sasha Aither <ceo@aitherlab.org>` (не трогать глобальный config)
- **Линтеры**: `golangci-lint`, `gofumpt`, `goimports` через `go install`
- **Тесты**: `make test`, `go test -race`, покрытие ядра ≥85%, HTTP/auth ≥80%
- **Интеграционный тест**: `make integration` за билд-тегом, парсит PIN из stdout, проходит pair flow, реально поднимает сервер
- **Manual-тесты**: `make scan-debug` (реальные .desktop), `make icons-debug` (реальные иконки) — за билд-тегом `manual`
- **Реальная проверка**: `XDG_CONFIG_HOME=/tmp/rl-test ./bin/remotelauncher` + curl через HTTPS с `-k`
- **Пользователь не программист** — объяснять в терминах «что сервер умеет», не листингами кода
- **Commits в Conventional Commits** с обязательным `Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>` trailer
- **Команды агентов через TeamCreate**, не через Agent — это требование из глобальных инструкций пользователя

## Как стартовать новую сессию над S5.1

```
/init   # уже запускался, CLAUDE.md есть
# прочитать docs/PROGRESS.md, /home/sasha/.claude/plans/bubbly-greeting-knuth.md
# собрать команду TeamCreate remotelauncher-s5-1 с developer + reviewer
# этапы 5.1 → 5.2 → 5.3 → 5.4 последовательно
```

После S5.4 — готов MVP-сервер. Решать, идём ли мы во Фазу A (Android) — это длинный блок с установкой Android SDK.
