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

# Ручной прогон линтеров (то же, что делает pre-commit)
lint:
	go vet ./...
	golangci-lint run

# Автоформатирование (gofmt из .golangci.yml)
fmt:
	golangci-lint fmt
