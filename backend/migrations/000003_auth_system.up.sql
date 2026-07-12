
CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email varchar(255) UNIQUE NOT NULL,
    password_hash varchar(255) NOT NULL,
    nickname varchar(50) UNIQUE NOT NULL,
    role user_role DEFAULT 'user',
    university university,
    course int,
    faculty varchar(100),
    email_verified_at timestamp,
    banned_at timestamp,
    ban_reason text,
    banned_by uuid,
    created_at timestamp DEFAULT now(),
    updated_at timestamp DEFAULT now()
);

CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_banned_at ON users(banned_at);

CREATE TABLE refresh_tokens (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    token_hash varchar(255) UNIQUE NOT NULL,
    expires_at timestamp NOT NULL,
    created_at timestamp DEFAULT now(),
    CONSTRAINT fk_refresh_tokens_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

ALTER TABLE users ADD CONSTRAINT fk_users_banned_by FOREIGN KEY (banned_by) REFERENCES users(id);
