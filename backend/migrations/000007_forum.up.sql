CREATE TABLE threads (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id uuid NOT NULL,
    title varchar(500),
    content text NOT NULL,
    tags thread_tag[],
    files jsonb,
    views_count int DEFAULT 0,
    likes_count int DEFAULT 0,
    comments_count int DEFAULT 0,
    hidden_at timestamp,
    hidden_by uuid,
    hidden_reason text,
    deleted_at timestamp,
    created_at timestamp DEFAULT now(),
    updated_at timestamp DEFAULT now(),
    CONSTRAINT fk_threads_author FOREIGN KEY (author_id) REFERENCES users(id),
    CONSTRAINT fk_threads_hidden_by FOREIGN KEY (hidden_by) REFERENCES users(id)
);

CREATE INDEX idx_threads_author_id ON threads(author_id);
CREATE INDEX idx_threads_created_at ON threads(created_at);
CREATE INDEX idx_threads_hidden_at ON threads(hidden_at);
CREATE INDEX idx_threads_deleted_at ON threads(deleted_at);

CREATE TABLE comments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id uuid NOT NULL,
    parent_id uuid,
    author_id uuid NOT NULL,
    content text NOT NULL,
    depth int DEFAULT 0,
    likes_count int DEFAULT 0,
    hidden_at timestamp,
    hidden_by uuid,
    hidden_reason text,
    deleted_at timestamp,
    created_at timestamp DEFAULT now(),
    updated_at timestamp DEFAULT now(),
    CONSTRAINT fk_comments_thread FOREIGN KEY (thread_id) REFERENCES threads(id),
    CONSTRAINT fk_comments_parent FOREIGN KEY (parent_id) REFERENCES comments(id),
    CONSTRAINT fk_comments_author FOREIGN KEY (author_id) REFERENCES users(id),
    CONSTRAINT fk_comments_hidden_by FOREIGN KEY (hidden_by) REFERENCES users(id)
);

CREATE INDEX idx_comments_thread_id ON comments(thread_id);
CREATE INDEX idx_comments_parent_id ON comments(parent_id);
CREATE INDEX idx_comments_author_id ON comments(author_id);
CREATE INDEX idx_comments_depth ON comments(depth);
CREATE INDEX idx_comments_created_at ON comments(created_at);

CREATE TABLE reactions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    target_type reaction_target_type NOT NULL,
    target_id uuid NOT NULL,
    emoji varchar(10) NOT NULL,
    created_at timestamp DEFAULT now(),
    CONSTRAINT fk_reactions_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT uq_reactions_user_target UNIQUE (user_id, target_type, target_id)
);

CREATE INDEX idx_reactions_user_id ON reactions(user_id);
CREATE INDEX idx_reactions_target ON reactions(target_type, target_id);
