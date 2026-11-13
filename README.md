# looqa_go
Backend сервиса Looqa

## Микросервисы

| Сервис | Порт | Роуты |
|---|---|---|
| comment | 8080 | `/comments` |
| user | 8081 | `/users` |
| poster | 8082 | `/posters` |

Каждый сервис — отдельный бинарник (`cmd/comment`, `cmd/user`, `cmd/poster`)
и отдельный контейнер. Все настройки собраны в `.env` — его читают и
`docker compose` (порты, креды postgres), и сами сервисы (`internal/config`
подгружает его через `godotenv`). Переменные окружения имеют приоритет над
`.env`. Значения по умолчанию — в `internal/config/default.go`.

## Запуск

Локально:

```sh
make run-comment   # http://localhost:8080
make run-user      # http://localhost:8081
make run-poster    # http://localhost:8082
```

В Docker (сервисы + postgres + pgAdmin):

```sh
make up            # или docker compose up --build
```

pgAdmin — веб-интерфейс для postgres: http://localhost:5050
(логин/пароль веб-интерфейса — `PGADMIN_DEFAULT_*` из `.env`).
Внутри pgAdmin сервер БД добавляйте с host=`db`, port=`5432` —
это внутренняя сеть compose, а не localhost.

## Линтеры и pre-commit

Перед каждым коммитом прогоняются `go vet ./...` и `golangci-lint run`
(конфиг — `.golangci.yml`: стандартный набор линтеров + gofmt).

Активация хука (один раз после клона):

```sh
make hooks   # git config core.hooksPath .githooks
```

Ручной прогон: `make lint`. Автоформатирование: `make fmt`.
Нужен установленный [golangci-lint](https://golangci-lint.run):
`brew install golangci-lint`.

