run-comment:
	go run ./cmd/comment

run-poster:
	go run ./cmd/poster

run-user:
	go run ./cmd/user

up:
	docker compose up --build

down:
	docker compose down

# Применить все новые миграции из migrations/*.sql (креды — как в .env).
# Каждый файл выполняется ОДНОЙ транзакцией: при любой ошибке внутри файла
# он откатывается целиком и не фиксируется как применённый.
# Журнал применённых — таблица schema_migrations в базе.
migrate:
	@docker compose exec -T db psql -U user -d mydb -q -c \
		'CREATE TABLE IF NOT EXISTS schema_migrations (name text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())'
	@ls migrations/*.sql >/dev/null 2>&1 || { echo 'В migrations/ нет .sql файлов'; exit 0; }
	@for f in migrations/*.sql; do \
		name=$$(basename "$$f"); \
		applied=$$(docker compose exec -T db psql -U user -d mydb -tA -c \
			"SELECT 1 FROM schema_migrations WHERE name = '$$name'"); \
		if [ "$$applied" = "1" ]; then \
			echo "== $$name — уже применена, пропускаю"; \
		else \
			echo "== $$name — применяю"; \
			{ cat "$$f"; printf '\nINSERT INTO schema_migrations (name) VALUES (%s);\n' "'$$name'"; } \
				| docker compose exec -T db psql -U user -d mydb -v ON_ERROR_STOP=1 --single-transaction \
				|| { echo "!! $$name упала, транзакция откачена"; exit 1; }; \
		fi; \
	done
	@echo 'Готово.'

# Ручной прогон линтеров (то же, что делает pre-commit)
lint:
	go vet ./...
	golangci-lint run

# Автоформатирование (gofmt из .golangci.yml)
fmt:
	golangci-lint fmt
