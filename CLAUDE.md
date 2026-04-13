# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Текущее состояние

MVP работает end-to-end на реальном железе: телефон → подтверждение SPKI fingerprint → pairing по PIN → сетка приложений → тап → программа стартует на ПК. Фазы 1–5 (сервер) и A0–A5 (Android) закрыты и закоммичены в `main` локально, не запушены.

- Полный план: `/home/sasha/.claude/plans/bubbly-greeting-knuth.md` (секции A0..A6)
- Снимок прогресса: `docs/PROGRESS.md`
- Источник требований: `docs/ТЗ.md`

Перед написанием кода сверяться с планом и ТЗ.

### Сервер (Go, MVP готов)

- HTTPS на `:8443`, self-signed ECDSA-сертификат в `~/.config/remotelauncher/`
- `cert_fingerprint` в `/api/status` — это **SHA-256 SPKI hash** (не cert hash), для пиннинга на клиенте
- Парсит `.desktop`, отдаёт список и иконки, запускает программы (отвязанно), reaping SIGCHLD
- PIN-pairing → Bearer-tokens (SHA-256 в `tokens.json`), rate-limit на `/api/pair`
- TOML-конфиг в `~/.config/remotelauncher/config.toml`, structured logging через `log/slog`
- systemd user-unit (`make install`), release-тарбол (`make package`)
- Покрытие ядра 92–100%, http/auth ≥88%

### Android-клиент (Kotlin, Compose, A0–A5 готовы)

- Compose, minSdk 26, targetSdk 35, namespace `com.remotelauncher`
- Ktor 3 CIO + kotlinx-serialization, Coil 3 для иконок, navigation-compose
- DataStore preferences для URL, EncryptedSharedPreferences для токена и pin
- SPKI-pinning через `PinnedTrustManager` + `PinHolder` (singleton AtomicReference)
- TOFU при первом подключении: диалог с fingerprint, кнопка «Доверять»
- `usesCleartextTraffic=false`, dev-trust удалён

### Что осталось — A6 (release)

- `assembleRelease` с подписью release-keystore (вне репо)
- Инструкция по созданию keystore (gitignore + где хранить пароль)
- Готовый APK для GitHub Releases

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

## Ключевые архитектурные решения (из ТЗ)

- **Безопасность**: TLS обязателен, первое подключение — pairing через PIN-код, далее токен. Rate limiting на авторизацию, опциональный whitelist IP.
- **Подключение снаружи** — выбор пользователя: проброс порта + DDNS, reverse proxy, Tailscale/ZeroTier, Cloudflare Tunnel. Сервер не завязывается на конкретный способ.
- **mDNS** — только бонус для локалки, основной режим — прямое подключение по адресу.
- **API** — REST для списка/запуска/остановки/иконок, WebSocket `/ws` для статуса в реальном времени. Полный список в `docs/ТЗ.md`.

## Гочи Android (важно для будущих сессий)

- **`X509ExtendedTrustManager`, не `X509TrustManager`**: Android Conscrypt вызывает `Socket`/`SSLEngine` варианты `checkServerTrusted`. Если их не реализовать — handshake падает молча, без понятной ошибки. Должны быть все 6 методов.
- **SPKI ≠ cert hash**: сервер считает `sha256(cert.RawSubjectPublicKeyInfo)`, Android — `sha256(cert.publicKey.encoded)`. Раньше сервер считал sha256 от всего сертификата — это сломано, исправлено в `feat(tls): server fingerprint computes SHA-256 SPKI`.
- **`adb shell input text` плохо ест спецсимволы** (`:`, `/`, `https://`). Для тестов вводить адрес без схемы (`192.168.1.248:8443`) — `parseServerUrl` сам добавит `https://`.
- **Координаты тапов через `adb shell uiautomator dump`**, иначе попадаешь в соседний элемент.

## MVP vs будущие версии

MVP (v1): список приложений с иконками, подключение по адресу, запуск по нажатию, PIN-pairing, HTTPS, SPKI-pinning. WebSocket-статус, остановка, категории, mDNS, виджеты — всё это v2+. При работе над MVP не тащить фичи из следующих версий без явного запроса.

## Рабочий процесс

- **Этапы идут по плану**, для крупных задач — отдельная команда через `TeamCreate` (developer + reviewer). Для мелких правок можно вручную, если проще и быстрее.
- **Никаких half-done** — каждый этап оставляет зелёные тесты и работоспособный артефакт (собираемый бинарь/APK).
- **Правки по ревью делаются в том же этапе**, не переносятся в следующий.
- **Коммиты в Conventional Commits** (`feat(scope): ...`) с обязательным trailer'ом `Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>`.
- **Ничего не пушить в remote** без явного разрешения пользователя.

## Про пользователя

Пользователь не программист — не показывать листинги кода в ответах, объяснять на уровне «что происходит» и «что делать», а не «вот такая функция». Общение на русском, неформальное.
