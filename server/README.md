# RemoteLauncher Server

Go-сервер для RemoteLauncher — запускает Linux-приложения по запросу с Android-клиента.

## Сборка

```sh
make build
```

## Запуск

```sh
make run
```

## Тесты

```sh
make test
```

## Структура

```
server/
├── cmd/
│   └── remotelauncher/   # точка входа (main)
└── internal/             # внутренние пакеты (появятся в следующих этапах)
```
