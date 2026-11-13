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

В Docker (3 контейнера сервисов + postgres):

```sh
make up            # или docker compose up --build
```

