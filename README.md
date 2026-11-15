# looqa_go
Backend сервиса Looqa

## Микросервисы

| Сервис | Порт | Роуты |
|---|---|---|
| comment | 8080 | `/comments` |
| user | 8081 | `/users` |
| poster | 8082 | `/posters` |
| auth | 8084 | `/reg`, `/login`, `/refresh`, `/logout`, `/jwks` |
| bridge | 8085 | `/message` |

Каждый сервис — отдельный бинарник (`cmd/comment`, `cmd/user`, `cmd/poster`,
`cmd/auth`, `cmd/bridge`) и отдельный контейнер. Все настройки собраны в `.env` — его читают и
`docker compose` (порты, креды postgres), и сами сервисы (`internal/config`
подгружает его через `godotenv`). Переменные окружения имеют приоритет над
`.env`. Значения по умолчанию — в `internal/config/default.go`.

## Запуск

Локально:

```sh
make run-comment   # http://localhost:8080
make run-user      # http://localhost:8081
make run-poster    # http://localhost:8082
make run-auth      # http://localhost:8084 (нужны postgres и mongo — DB_URI и MONGO_URI)
make run-bridge    # http://localhost:8085 (стартует без зависимостей, но для запросов нужен auth)
```

В Docker (сервисы + postgres + mongo + pgAdmin):

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
`User`/`Comment`/`Poster`/`Credential`. Миграции лежат в `migrations/*.sql`
и применяются командой `make migrate`: каждый файл — одной транзакцией,
журнал применённых ведётся в таблице `schema_migrations`. Новые миграции
добавляйте файлами `003_имя.sql`, `004_имя.sql` и т.д. — раннер подхватит
их автоматически.

Локально (`make run-*`) сервисам нужен доступный postgres — задайте
`DB_URI` (например, на поднятую в docker базу), auth-сервису ещё
и `MONGO_URI`. Без токена ручки вернут 401, а с валидным токеном — 503,
если не запущен auth-сервис (ресурсные берут у него ключи `/jwks`,
адрес — `AUTH_JWKS_URL`, по умолчанию `http://localhost:8084/jwks`):

```sh
DB_URI=postgres://user:pass@localhost:9090/mydb make run-user
```

## Аутентификация (JWT)

Вход — `POST /login` (email + пароль) — выдаёт пару токенов:

- **access** — JWT (алгоритм EdDSA/Ed25519), живёт `AUTH_ACCESS_TTL`
  (по умолчанию 15 минут). ВСЕ ручки comment/user/poster/bridge требуют
  его в заголовке `Authorization: Bearer <access>`.
- **refresh** — одноразовый, живёт `AUTH_REFRESH_TTL` (по умолчанию 30 дней).
  Хранится в MongoDB только в виде sha256-хеша (коллекция `refresh_tokens`,
  истёкшие чистит TTL-индекс). Обменивается на новую пару в `POST /refresh`,
  старый при этом ротируется.

Ресурсные сервисы проверяют access локально: забирают публичный ключ
с `GET /jwks` auth-сервиса и кэшируют его (при рестарте auth ключ меняется —
сервисы перечитают /jwks по неизвестному kid). Ответ **401** — токена нет /
битый / просрочен / сессия отозвана; **503** — auth-сервис недоступен.

Ротация защищает от кражи refresh: повторное использование уже
ротированного токена отзывает всю сессию. `POST /logout`
(с валидным access) тоже отзывает сессию; сам access отзывается только
истечением — для мгновенного отзыва понадобился бы чёрный список.

`/reg`, `/login` и `/refresh` ограничены по частоте (`AUTH_RATE_LIMIT`,
по умолчанию 30 запросов/мин с IP).

Быстрый обход:

```sh
curl -X POST localhost:8084/reg -d '{"email":"ivan@example.com","password":"secret123","first_name":"Ivan","last_name":"Petrov"}'
curl -X POST localhost:8084/login -d '{"email":"ivan@example.com","password":"secret123"}'
# → {"access_token":"eyJ...","refresh_token":"...","token_type":"Bearer","expires_in":900}
curl localhost:8080/comments -H "Authorization: Bearer <access_token>"
```

## bridge (AI-прокси)

`POST /message` пересылает сообщение LLM через библиотеку
`github.com/AlexeyBMSTU/ai-bridge`: в теле — `{"message": "..."}`, креды
провайдера — в заголовках запроса `X-API-Key` (API-ключ), `X-Model`
(название модели) и `X-Provider-URL` (адрес API). Ответ модели
возвращается как JSON-строка. Открытые соединения кэшируются в памяти
по ключу, сами креды не хранятся. Ручка защищена access-токеном.
Сервису не нужны ни postgres, ни mongo.

## API (OpenAPI)

Спецификация всех ручек — `openapi.yaml` в корне (OpenAPI 3.0.3):
18 операций пяти сервисов, сгруппированы по тегам `comment`/`user`/
`poster`/`auth`/`bridge`. В Swagger UI нажмите **Authorize** и вставьте
access-токен — все Try it out пойдут с ним.
У каждого пути указан свой базовый URL (порт сервиса), поэтому импорт
в Postman/Bruno работает сразу.

```sh
make up   # затем откройте http://localhost:5051
```

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

Юнит-тесты: `go test ./...` — покрыты `internal/auth` (выпуск/проверка JWT,
JWKS-верификатор), `auth_service` и `auth_handler` (моки хранилищ),
`cmd/server/middleware` (аутентификация).

