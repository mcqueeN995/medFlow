-- users.login - отдельный от email и nickname идентификатор для входа:
-- задаётся при регистрации, уникален, не меняется свободно (в отличие от
-- nickname, который остаётся только отображаемым именем). Бэкфилл из
-- nickname для существующих строк - на момент миграции они совпадают,
-- дальше расходятся независимо.
ALTER TABLE users ADD COLUMN login varchar(50);
UPDATE users SET login = nickname WHERE login IS NULL;
ALTER TABLE users ALTER COLUMN login SET NOT NULL;
ALTER TABLE users ADD CONSTRAINT uq_users_login UNIQUE (login);

-- Смена login требует подтверждения кодом на уже привязанный email
-- (доказывает, что запрос от владельца аккаунта, а не от угнавшего сессию).
CREATE TABLE login_change_requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    new_login varchar(50) NOT NULL,
    code_hash varchar(255) UNIQUE NOT NULL,
    expires_at timestamp NOT NULL,
    created_at timestamp DEFAULT now(),
    CONSTRAINT fk_login_change_requests_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_login_change_requests_user_id ON login_change_requests(user_id);
