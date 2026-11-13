# Один Dockerfile на все сервисы: какой собирать, выбирается через
# build-аргумент SERVICE (comment | poster | user), см. compose.yaml.
FROM golang:1.27-alpine AS builder

ENV CGO_ENABLED=0

ARG SERVICE=comment

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /app ./cmd/${SERVICE}

# -------------
FROM alpine:3.21 AS final

COPY --from=builder /app /bin/app

# Порты публикуются в compose.yaml (comment 8080, user 8081, poster 8082).
CMD ["/bin/app"]
