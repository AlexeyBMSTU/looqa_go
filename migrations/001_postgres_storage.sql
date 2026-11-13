-- Миграция схемы хранения сервисов looqa в postgres.
-- Применяется командой make migrate: каждый файл из migrations/ выполняется
-- одной транзакцией и фиксируется в таблице schema_migrations.
-- Новые миграции добавляйте новыми файлами: 002_имя.sql, 003_имя.sql, ...
-- Этот файл идемпотентен: применим и к пустой базе, и к базе с таблицами,
-- созданными вручную (они будут адаптированы ALTER-ами внизу).

CREATE TABLE IF NOT EXISTS "User" (
    id         uuid PRIMARY KEY,
    first_name varchar(255),
    last_name  varchar(255),
    avatar_url text
);

CREATE TABLE IF NOT EXISTS "Comment" (
    id          uuid PRIMARY KEY,
    author_id   uuid NOT NULL REFERENCES "User" (id) ON DELETE CASCADE,
    likes_count integer
);

CREATE TABLE IF NOT EXISTS "Poster" (
    id             uuid PRIMARY KEY,
    label          text,
    text           text,
    author_id      uuid NOT NULL REFERENCES "User" (id) ON DELETE CASCADE,
    comments_count integer,
    likes_count    integer,
    publication_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE "User"   ALTER COLUMN first_name TYPE varchar(255);
ALTER TABLE "User"   ALTER COLUMN last_name  TYPE varchar(255);
ALTER TABLE "Poster" ADD COLUMN IF NOT EXISTS text text;
