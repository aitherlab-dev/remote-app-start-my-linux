# RemoteLauncher — прогресс

Снимок состояния для новой сессии Claude Code. Перед стартом прочитай:
1. Этот файл — короткая картина «где мы»
2. `/home/sasha/.claude/plans/bubbly-greeting-knuth.md` — полный план
3. `docs/ТЗ.md` — исходное техническое задание
4. `CLAUDE.md` в корне

## Текущее состояние: Фаза 5 завершена — сервер MVP-готов

26 коммитов в `main`. Последний feat-коммит: `c99908f feat(build): README, ldflags version, final green run`.

**Сервер полностью функциональный, защищённый и упакованный**. Ставится на машину одной командой `make install`: бинарь уезжает в `~/.local/bin/`, systemd user-unit в `~/.config/systemd/user/`, сервис поднимается через `enable --now`. Можно раздавать как MVP.

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

### Фаза 5 — Упаковка
- `393f2d7 feat(config): TOML config with defaults→file→env→flags precedence` (S5.1)
- `8c1a425 feat(log): slog level and format from config` (S5.2)
- `b2028e2 feat(packaging): systemd user unit and make install/uninstall/package` (S5.3)
- `c99908f feat(build): README, ldflags version, final green run` (S5.4)

### Финальные цифры сервера

- Покрытие ядра: catalog 100%, desktop 91.9%, icons 94.0%, launcher 98.2%
- Покрытие API/auth: httpapi 90.4%, auth 88.0%
- Config/tls: config 92.0%, tlsutil 86.4%
- Суммарно: 87.2% statements
- Бинарь: 10.3 MiB, одна внешняя зависимость (`BurntSushi/toml`)
- `version` в `/api/status` вшивается через `-ldflags -X main.Version=$(git describe)`

## Что осталось

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

## Как стартовать новую сессию над Android-клиентом

```
/init   # уже запускался, CLAUDE.md есть
# прочитать docs/PROGRESS.md, /home/sasha/.claude/plans/bubbly-greeting-knuth.md
# первый шаг — A0.1: установить JDK, Android Studio, SDK Platform 35,
# настроить ANDROID_HOME и PATH, подключить телефон с USB-debug
```

Сервер — закрытая история на уровне MVP. Следующая длинная секция — Android: установка инструментов, скелет проекта, Connect/Pairing/Apps/Launch/HTTPS/Release. Полная декомпозиция — в плане, раздел «A0…A6».
