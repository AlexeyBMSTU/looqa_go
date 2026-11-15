-- Учётные данные для входа: email (уникальный) + хеш пароля (bcrypt).
-- user_id ссылается на "User": при удалении пользователя его учётные
-- данные удаляются каскадом.
CREATE TABLE IF NOT EXISTS "Credential" (
    user_id       uuid PRIMARY KEY REFERENCES "User" (id) ON DELETE CASCADE,
    email         varchar(255) NOT NULL UNIQUE,
    password_hash text NOT NULL
);
