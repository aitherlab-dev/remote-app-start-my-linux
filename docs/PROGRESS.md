# RemoteLauncher — прогресс

Снимок состояния для новой сессии Claude Code. Перед стартом прочитай:
1. Этот файл — короткая картина «где мы»
2. `/home/sasha/.claude/plans/bubbly-greeting-knuth.md` — исходный план (секции A0..A6 для MVP)
3. `docs/ТЗ.md` — исходное техническое задание
4. `CLAUDE.md` в корне

## Текущее состояние — MVP работает + post-MVP фаза B идёт

Конвейер работает end-to-end на реальном железе: Samsung Galaxy `RFCY818A5MT` → HTTPS на ПК по `192.168.1.248:8443` → сетка приложений → тап → kitty/приложение стартует на Linux. Сервер крутится в systemd user-unit, автозапуск включён, Linger=yes (работает даже без логина).

## Что уже сделано

### Фаза MVP (commit до 7be9dd1)

- **Сервер (Go)** — фазы 1–5: парсинг `.desktop`, иконки, запуск, TLS+SPKI, pairing+токены, rate-limit, TOML конфиг, systemd
- **Android (Kotlin, Compose)** — фазы A0–A5: Connect / Pairing / Apps экраны, Ktor 3, Coil 3, DataStore, EncryptedSharedPreferences, SPKI-pinning через `PinnedTrustManager`+`PinHolder`, TOFU диалог

### Фаза B — post-MVP расширения

- **B1 (commit `ecdea75`)** — веб-админка + фильтрация видимости
  - Второй HTTP-сервер на `127.0.0.1:17843` (loopback, plain HTTP)
  - Встроенная в бинарь через `embed.FS` веб-морда на Tailwind + daisyUI + Alpine.js (через CDN)
  - `visibility.json` рядом с `tokens.json` — список скрытых app.id
  - `/api/apps` на :8443 фильтрует скрытые — телефон их не видит
  - Блок `[web]` в config.toml, валидация refuse не-loopback адреса
  - Coverage: visibility 83.7%, web 87.2%

- **B2 (commit `10a044d`)** — переверстка Android: читаемая тема, названия под иконками, pair error handling
  - Своя light/dark палитра вместо дефолтного Purple template
  - `RemoteLauncherTheme` **принудительно light** (darkTheme = false) потому что активити в манифесте забит на `Theme.Material.Light.NoActionBar`, а `isSystemInDarkTheme()` брал системную тёмную → белый текст на белом фоне. Сейчас фиксировано light, выглядит хорошо
  - `Surface` обёртка поверх `AppNavHost` чтобы Compose-фон закрывал активити темой
  - `highContrastFieldColors()` + `fieldTextStyle()` в `ui/theme/Fields.kt` — в OutlinedTextField текст прибит к `onBackground` через `MaterialTheme.colorScheme`, плюс явный sp/weight
  - `AppsScreen`: убран `aspectRatio(1f)`, minSize сетки 104dp, иконка 56dp, `labelMedium` для названия с `minLines=2` — длинные имена видны
  - `HttpClientFactory`: `expectSuccess = true` — сервер, вернувший 401 при протухшем PIN, больше не парсится как PairResponse, UI показывает «Неверный PIN»

- **B3 (commit `25977d2`)** — кастомные ярлыки с CRUD в веб-админке
  - Новый пакет `internal/shortcuts` — хранилище `shortcuts.json` рядом с `tokens.json`/`visibility.json`, атомарный persist (tmp+rename), валидация
  - Поля ярлыка: `id` (латиница, без пробелов), `name`, `command`, `cwd`, `terminal`, `icon`
  - `launcher.LaunchCommand` — запуск произвольной команды в выбранном эмуляторе (`kitty`, `ghostty`, `gnome-terminal`, `konsole`, `alacritty`, `foot`, `xfce4-terminal`, `xterm`)
  - POSIX single-quote escape для cwd, payload оборачивается в `sh -c "cd '<cwd>' && exec <command>"`
  - `[launcher] default_terminal` в config.toml — fallback если в ярлыке terminal не задан
  - `/api/apps` на :8443 сливает `catalog.List()` с shortcuts, id ярлыков идут с префиксом `custom:`
  - `/api/apps/{id}/launch` — если id начинается с `custom:`, ищет в shortcuts и дёргает `LaunchCommand`
  - Веб-морда: вкладка «Ярлыки» рядом с «Приложениями», форма CRUD, GET/PUT `/api/shortcuts`
  - На Android ничего не трогал — ярлыки приходят в общий `/api/apps`, клиент их не отличает от обычных приложений
  - Coverage: shortcuts 82.2%, launcher 91.6%, httpapi 90.8%

## Что осталось

### B4 — Zero-config удалённый доступ (следующий этап)

**Цель:** подключаться с телефона с GSM/4G (не из локалки) **без** ручных действий на роутере и **без** Tailscale/Cloudflare/других внешних приложений. Всё внутри бинаря + APK.

**Как:**
1. **UPnP / NAT-PMP / PCP** в бинаре — сервер находит роутер через SSDP, просит открыть внешний порт 8443 → внутренний 8443, держит lease актуальным. Лира: `github.com/huin/goupnp` или `github.com/jackpal/go-nat-pmp` (или обе как fallback). Плюс узнать внешний IP через тот же UPnP.
2. **Встроенный DDNS-клиент** — простой HTTP-запрос на API DuckDNS/No-IP/FreeDNS раз в несколько минут. Блок `[ddns]` в `config.toml`: провайдер, токен, домен. Сервер генерирует строку типа «твой адрес: `myname.duckdns.org:8443`».
3. **UI в админке** — показать внешний адрес + QR-код для удобства ввода на телефоне. Секция «Удалённый доступ» с индикатором «UPnP работает / нет», текущим внешним IP, DDNS-именем.
4. **Graceful fallback** — если UPnP отключён в роутере, показывать сообщение «UPnP недоступен, либо включи его в роутере, либо дай серверу прямой внешний IP». Никакого ручного port-forwarding как основного пути.

**Важно, что пользователь прямо сказал:**
- «**ничего ставить дополнительно**» — не Tailscale, не Cloudflare Tunnel, не ddclient
- «**в роутер руками не лезть**» — UPnP обязательный
- VPN-клиенты на ПК влияют на curl'овский внешний IP (видно 146.103.121.77 от VPN-провайдера, не реальный), надо проверять внешний адрес через UPnP API роутера, а не через сторонние сервисы

Новая сессия начинает с:
1. Прочитать `project_priorities.md` в memory
2. Создать `internal/nat` и `internal/ddns` пакеты
3. Интегрировать в `main.go` + веб-морду
4. Протестировать на реальном роутере пользователя

### A6.1 — Release APK с подписью (в самом конце)

Только когда весь функционал закрыт. Раньше **не делать** — пользователь прямо послал меня на хер, когда я зациклился на релизе. Это финальная упаковка, ничего функционального не добавляет.

- release keystore через `keytool` в `~/keys/` (вне репо)
- пароли в `~/.gradle/gradle.properties`
- `signingConfigs.release` в `build.gradle.kts`
- `minifyEnabled=true`, `shrinkResources=true`
- ProGuard keep-правила для `kotlinx.serialization`
- ProGuard-smoke instrumented-test с реальным JSON-парсом
- `apksigner verify --verbose` → v2+v3
- README с инструкцией установки + fingerprint ключа
- Бэкап keystore (потеря = невозможность выпускать обновления)

## Важные факты для новой сессии

- **Репо**: `/home/sasha/WORK/REMOTE-MY-LINUX/`
- **Go-код**: `server/`, Go 1.26.1
- **Android-код**: `android/`, Kotlin 2.x, Compose, minSdk 26, targetSdk 35
- **git user**: локально `Sasha Aither <ceo@aitherlab.org>`
- **Линт**: `make lint` (golangci-lint + gofumpt + goimports)
- **Тесты сервера**: `make test` или `make cover`
- **Интеграционный тест сервера**: `make integration` — **требует свободного порта 8443**, т.е. перед запуском надо `systemctl --user stop remotelauncher`
- **Тесты Android**: `./gradlew :app:testDebugUnitTest`
- **Реальная установка**: `make install` → systemd user-unit поднимается автоматически, `make uninstall` для сноса
- **Linger=yes** уже включён — сервис стартует с загрузкой системы, до логина

## Про рабочий процесс

- **НЕ использовать `TeamCreate`** — пользователь раздражается, считает слишком медленным. Работать напрямую в основной сессии. См. `feedback_team_create_overhead.md` в memory.
- **Порядок задач**: UPnP+DDNS (B4) → релиз-APK (A6.1). Ничего не переставлять, релиз в самом конце. См. `project_priorities.md` в memory.
- **Никакого Tailscale/Cloudflare Tunnel/ddclient** — пользователь хочет self-contained в бинаре и APK.
- **Пользователь не программист** — объяснять на уровне «что теперь умеет», не листингами кода. Общение на русском, неформальное.
- **Коммиты** в Conventional Commits с обязательным trailer'ом `Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>`.
- **Не пушить в remote** без явного разрешения.

## Как стартовать новую сессию над B4 (UPnP + DDNS)

```
# Прочитать в таком порядке:
# 1. /home/sasha/WORK/REMOTE-MY-LINUX/CLAUDE.md
# 2. /home/sasha/WORK/REMOTE-MY-LINUX/docs/PROGRESS.md (этот файл)
# 3. ~/.claude/projects/-home-sasha-WORK-REMOTE-MY-LINUX/memory/MEMORY.md
# 4. /home/sasha/.claude/plans/bubbly-greeting-knuth.md — разделы про безопасность и remote access
```

Первый шаг: выбрать UPnP-либу (`huin/goupnp` vs `jackpal/go-nat-pmp`), прототип `internal/nat/discover.go` который ищет IGD на локалке и получает внешний IP.
