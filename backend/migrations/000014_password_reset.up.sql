-- Восстановление пароля через код, отправленный на email - тот же паттерн,
-- что и login_change_requests (000013), но без "new_*" поля: новый пароль
-- приходит вместе с кодом на шаге подтверждения, а не на шаге запроса.
CREATE TABLE password_reset_requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    code_hash varchar(255) UNIQUE NOT NULL,
    expires_at timestamp NOT NULL,
    created_at timestamp DEFAULT now(),
    CONSTRAINT fk_password_reset_requests_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_password_reset_requests_user_id ON password_reset_requests(user_id);
