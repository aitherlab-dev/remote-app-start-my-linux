# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Текущее состояние

MVP работает end-to-end на реальном железе + фаза B (post-MVP расширения) закрыта:
- **Фаза MVP** (сервер 1–5, Android A0–A5) — ✅ закрыта
- **B1** — веб-админка на `127.0.0.1:17843` + фильтр видимости — ✅ закрыта (commit `ecdea75`)
- **B2** — переверстка Android: читаемая тема, названия под иконками, фикс pair error — ✅ закрыта (commit `10a044d`)
- **B3** — кастомные ярлыки (запуск `claude` и прочего в терминале) с CRUD в веб-морде — ✅ закрыта (commit `25977d2`)
- **B4 (удалённый доступ)** — **отложен как отдельная будущая фаза**, план зафиксирован в `docs/FUTURE-REMOTE-ACCESS.md`. Текущий релиз работает только в локальной сети.
- **A6.1** — Release APK с подписью — следующий этап

Remote: `git@github.com:aitherlab-dev/remote-app-start-my-linux.git` (публичный). Тестовый стенд: Samsung Galaxy `RFCY818A5MT` + ПК `192.168.1.248:8443`.

- Снимок прогресса со всеми деталями: `docs/PROGRESS.md`
- Исходный план MVP: `/home/sasha/.claude/plans/bubbly-greeting-knuth.md` (секции A0..A6)
- Источник требований: `docs/ТЗ.md`

Перед написанием кода сверяться с `docs/PROGRESS.md` и этим файлом.

### Сервер (Go)

- HTTPS на `:8443`, self-signed ECDSA-сертификат в `~/.config/remotelauncher/`
- `cert_fingerprint` в `/api/status` — **SHA-256 SPKI hash** (не cert hash), для пиннинга на клиенте
- Парсит `.desktop`, отдаёт список и иконки, запускает программы (отвязанно), reaping SIGCHLD
- PIN-pairing → Bearer-tokens (SHA-256 в `tokens.json`), rate-limit на `/api/pair`
- TOML-конфиг в `~/.config/remotelauncher/config.toml`, structured logging через `log/slog`
- systemd user-unit (`make install`), release-тарбол (`make package`)
- **Второй HTTP-сервер на `127.0.0.1:17843`** (loopback, без TLS) — веб-админка со встроенной через `embed.FS` SPA на Tailwind + daisyUI + Alpine.js
- **`visibility.json`** рядом с `tokens.json` — скрытые от телефона приложения. `/api/apps` на 8443 фильтрует
- **`shortcuts.json`** — кастомные ярлыки (запуск произвольной команды в эмуляторе терминала с нужным cwd). Поддержаны: kitty, ghostty, gnome-terminal, konsole, alacritty, foot, xfce4-terminal, xterm. id с префиксом `custom:` в `/api/apps`/`/api/apps/{id}/launch`
- Покрытие: ядро 92–100%, http/auth ≥88%, shortcuts 82.2%, web 87.2%, visibility 83.7%

### Android-клиент (Kotlin, Compose)

- Compose, minSdk 26, targetSdk 35, namespace `com.remotelauncher`
- Ktor 3 CIO + kotlinx-serialization, Coil 3 для иконок, navigation-compose
- DataStore preferences для URL, EncryptedSharedPreferences для токена и pin
- SPKI-pinning через `PinnedTrustManager` + `PinHolder` (singleton AtomicReference)
- TOFU при первом подключении: диалог с fingerprint, кнопка «Доверять»
- `usesCleartextTraffic=false`, dev-trust удалён
- **Своя light-тема** (`RemoteLauncherTheme` с `darkTheme = false` **принудительно**): из-за `Theme.Material.Light.NoActionBar` в манифесте активити фон всегда белый, а `isSystemInDarkTheme()` на тёмной системе давал белый текст поверх. Сейчас forced light + `Surface`-обёртка над `AppNavHost`
- **`highContrastFieldColors()` + `fieldTextStyle()`** в `ui/theme/Fields.kt` — в OutlinedTextField текст прибит к `onBackground` через `MaterialTheme.colorScheme`
- **Сетка приложений**: `AppsScreen` — убран `aspectRatio(1f)`, minSize 104dp, иконка 56dp, `labelMedium` + `minLines=2` для читаемого имени
- **`HttpClientFactory.expectSuccess = true`** — сервер с 401 на pair с просроченным PIN больше не парсится как PairResponse; UI показывает «Неверный PIN»

### Следующий этап — A6.1 (release APK)

Release keystore, подпись v2+v3, ProGuard/R8 с keep-правилами для kotlinx-serialization, smoke-тест после ProGuard, README с fingerprint ключа. Подробности в `docs/PROGRESS.md`.

### Будущая фаза — удалённый доступ через платный VPS-туннель

Отложена. План и модель описаны в `docs/FUTURE-REMOTE-ACCESS.md`. Короткая суть: сервер держит исходящий туннель к VPS автора, телефон ходит на VPS. Работает за CGNAT, не требует UPnP/DDNS, модель — платная подписка. Изначальный B4 (UPnP + DDNS) отвергнут, потому что ломается на CGNAT и не даёт настоящего zero-config.

## Что за проект

**RemoteLauncher** — лаунчер программ на Linux-десктопе с Android-смартфона через интернет или локальную сеть. Не VNC/remote desktop: пользователь видит сетку приложений с иконками, нажимает — программа запускается на ПК.

- **Сервер** (Go, Linux) — один бинарник без зависимостей, парсит `.desktop` файлы из `/usr/share/applications/` и `~/.local/share/applications/`, отдаёт REST, запускает процессы.
- **Клиент** (Kotlin, Android) — Material 3, подключение по адресу + порту, APK через GitHub Releases.

## Структура

```
REMOTE-MY-LINUX/
├── server/      # Go-сервер (готов)
├── packaging/   # systemd unit + install.sh/uninstall.sh
├── android/     # Kotlin-клиент (готов до A5)
└── docs/        # ТЗ, PROGRESS, заметки
```

Android-исходники: `android/app/src/main/java/com/remotelauncher/{net,data,ui/{connect,pairing,apps}}`.

## Команды

**Сервер:**
```bash
make -C server build               # собрать ./bin/remotelauncher
make -C server test                # go test ./...
make -C server install             # systemd user-unit + бинарь в ~/.local/bin
make -C server package             # release-тарбол
systemctl --user status remotelauncher
journalctl --user -u remotelauncher -f
```

**Android:**
```bash
cd android
./gradlew :app:assembleDebug                                  # собрать debug APK
./gradlew :app:testDebugUnitTest                              # JVM unit-тесты
adb install -r app/build/outputs/apk/debug/app-debug.apk      # установить
adb shell pm clear com.remotelauncher                         # стереть данные (pin/токен)
adb shell am start -n com.remotelauncher/.MainActivity        # запустить
```

**Тестовый стенд:**
- сервер: `192.168.1.248:8443`
- телефон: Samsung Galaxy `RFCY818A5MT`

## Ключевые архитектурные решения (из ТЗ + решения пользователя по ходу разработки)

- **Безопасность**: TLS обязателен, первое подключение — pairing через PIN-код, далее токен. Rate limiting на авторизацию, опциональный whitelist IP.
- **Подключение снаружи** — в текущем релизе **не поддерживается**, только локальная сеть. Будущий платный туннель через VPS — см. `docs/FUTURE-REMOTE-ACCESS.md`.
- **mDNS** — только бонус для локалки, основной режим — прямое подключение по адресу.
- **API** — REST для списка/запуска/остановки/иконок, WebSocket `/ws` для статуса в реальном времени. Полный список в `docs/ТЗ.md`.
- **Веб-админка** — только на loopback (`127.0.0.1:17843`), без авторизации. Конфиг-валидация отбивает не-loopback адреса. Это десктопная часть, один раз настроил и забыл.
- **Кастомные ярлыки** — не реальные приложения, а shortcut-записи в `shortcuts.json`. id с префиксом `custom:` в API, чтобы не конфликтовать с .desktop приложениями. Запуск — через white-listed эмулятор терминала.

## Гочи Android (важно для будущих сессий)

- **`X509ExtendedTrustManager`, не `X509TrustManager`**: Android Conscrypt вызывает `Socket`/`SSLEngine` варианты `checkServerTrusted`. Если их не реализовать — handshake падает молча, без понятной ошибки. Должны быть все 6 методов.
- **SPKI ≠ cert hash**: сервер считает `sha256(cert.RawSubjectPublicKeyInfo)`, Android — `sha256(cert.publicKey.encoded)`. Раньше сервер считал sha256 от всего сертификата — это сломано, исправлено в `feat(tls): server fingerprint computes SHA-256 SPKI`.
- **`adb shell input text` плохо ест спецсимволы** (`:`, `/`, `https://`). Для тестов вводить адрес без схемы (`192.168.1.248:8443`) — `parseServerUrl` сам добавит `https://`.
- **Координаты тапов через `adb shell uiautomator dump`**, иначе попадаешь в соседний элемент.

## MVP vs будущие версии

MVP (v1): список приложений с иконками, подключение по адресу, запуск по нажатию, PIN-pairing, HTTPS, SPKI-pinning. WebSocket-статус, остановка, категории, mDNS, виджеты — всё это v2+. При работе над MVP не тащить фичи из следующих версий без явного запроса.

## Рабочий процесс

- **Никаких half-done** — каждый этап оставляет зелёные тесты и работоспособный артефакт (собираемый бинарь/APK).
- **Правки по ревью делаются в том же этапе**, не переносятся в следующий.
- **Коммиты в Conventional Commits** (`feat(scope): ...`) с обязательным trailer'ом `Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>`.
- **Push в remote `origin`** разрешён пользователем (репо публичный, github.com/aitherlab-dev/remote-app-start-my-linux).
- **Следующий этап — A6.1 (релизный APK)**. Удалённый доступ отложен в `docs/FUTURE-REMOTE-ACCESS.md`.

## Про пользователя

Пользователь не программист — не показывать листинги кода в ответах, объяснять на уровне «что происходит» и «что делать», а не «вот такая функция». Общение на русском, неформальное.
