CREATE TABLE push_subscriptions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    endpoint text NOT NULL UNIQUE,
    p256dh text NOT NULL,
    auth text NOT NULL,
    created_at timestamp DEFAULT now(),
    CONSTRAINT fk_push_subscriptions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_push_subscriptions_user_id ON push_subscriptions(user_id);

CREATE TABLE push_preferences (
    user_id uuid PRIMARY KEY,
    thread_reply boolean NOT NULL DEFAULT true,
    comment_reply boolean NOT NULL DEFAULT true,
    reaction boolean NOT NULL DEFAULT true,
    card_task_done boolean NOT NULL DEFAULT true,
    card_task_failed boolean NOT NULL DEFAULT true,
    moderation_action boolean NOT NULL DEFAULT true,
    system boolean NOT NULL DEFAULT true,
    updated_at timestamp DEFAULT now(),
    CONSTRAINT fk_push_preferences_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
