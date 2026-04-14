# RemoteLauncher

Launch programs on your Linux desktop by tapping app icons on your Android
phone. Not VNC, not remote desktop: you see a grid of app icons on the
phone, you tap one, and the app starts on the PC.

Two parts:

- **Server** (Go, Linux) — single static binary, no dependencies. Parses
  `.desktop` files, exposes a small REST API over HTTPS, launches
  processes. Mandatory TLS, PIN-based pairing, SPKI certificate pinning on
  the client.
- **Client** (Kotlin, Jetpack Compose) — Android app. Material 3,
  connects by address and port, stores the bearer token in
  `EncryptedSharedPreferences`.

The current release works **on local networks only**. Plans for remote
access over a paid VPS tunnel are captured in
[`docs/FUTURE-REMOTE-ACCESS.md`](docs/FUTURE-REMOTE-ACCESS.md).

## Install the server

```sh
cd server
make install
```

This builds the binary, installs it into `~/.local/bin/`, deploys a
systemd user unit and enables autostart. More details in
[`server/README.md`](server/README.md).

On the first start the server prints a **pairing PIN** (valid for 10
minutes). You need to enter it on the phone when you connect for the
first time.

```sh
journalctl --user -u remotelauncher -f
```

## Install the Android client

1. Grab `app-release.apk` from the latest
   [GitHub Release](https://github.com/aitherlab-dev/remote-app-start-my-linux/releases).
2. On the phone, allow installation from unknown sources for the
   browser/file manager you're opening the APK with.
3. Install and launch. Enter the server address (`192.168.1.xxx:8443` or
   `your-host:8443`), trust the TLS certificate fingerprint that appears,
   enter the PIN shown by the server, and you're in.

### Signing key fingerprint

When you install updates, Android enforces that the new APK is signed
with the same key as the installed one. You can verify the expected key
against:

```
Signer:  CN=Sasha Aither, O=aitherlab, C=RU
SHA-256: 7B:55:60:CB:94:23:86:23:59:D0:1E:08:97:41:11:87:3C:9E:9B:89:1D:8D:6E:96:A0:D4:98:2C:88:D6:A0:91
SHA-1:   5C:55:AC:E6:0A:C6:FB:DD:8B:E4:DE:22:16:60:D5:8A:EA:CB:31:86
```

## Security

- All traffic between server and client runs over HTTPS using an
  ECDSA self-signed certificate generated on first server start.
- The client pins the server by its **SHA-256 SPKI hash** on first
  connection (TOFU flow with an explicit accept dialog) and refuses
  any other certificate afterwards.
- Authentication: PIN pairing exchanges a bearer token; only its SHA-256
  hash is kept server-side, and `/api/pair` is rate-limited.

## Server admin UI

A second HTTP server runs on `http://127.0.0.1:17843` (loopback only, no
authentication by design): list of parsed apps, toggle visibility for
unwanted ones, and a CRUD editor for custom shortcuts (arbitrary
commands executed inside a whitelisted terminal emulator such as kitty,
ghostty, alacritty, gnome-terminal, ...).

## Repository layout

```
server/     Go server
android/    Kotlin Android client
packaging/  systemd user unit + install/uninstall scripts
docs/       original spec, progress notes, future plans
```

## Status

| Component | State |
| --- | --- |
| Server (parsing, TLS, pairing, launching) | Working |
| Android client (pairing, grid, tap-to-launch) | Working |
| Admin UI + app visibility filter | Working |
| Custom shortcuts | Working |
| Remote access over the internet | [Future phase](docs/FUTURE-REMOTE-ACCESS.md) |
