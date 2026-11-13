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

## Хранилище

В docker сервисы пишут в postgres: строки подключения собираются в
`compose.yaml` из `POSTGRES_*` (хост `db:5432`). Схема — таблицы
`User`/`Comment`/`Poster`. Миграции лежат в `migrations/*.sql` и применяются
командой `make migrate`: каждый файл — одной транзакцией, журнал применённых
ведётся в таблице `schema_migrations`. Новые миграции добавляйте файлами
`002_имя.sql`, `003_имя.sql` и т.д. — р аннер подхватит их автоматически.

Локально без docker (`make run-*`) переменная `DB_URI` не задана, и сервисы
работают на in-memory хранилище (данные живут до перезапуска процесса).
Запустить локально с postgres:

```sh
DB_URI=postgres://user:pass@localhost:9090/mydb make run-user
```

## API (OpenAPI)

Спецификация всех ручек — `openapi.yaml` в корне (OpenAPI 3.0.3):
12 операций трёх сервисов, сгруппированы по тегам `comment`/`user`/`poster`.
У каждого пути указан свой базовый URL (порт сервиса), поэтому импорт
в Postman/Bruno/Swagger UI работает сразу. Ручками тоже можно проверять:
например, в VS Code — плагин OpenAPI Preview.

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

