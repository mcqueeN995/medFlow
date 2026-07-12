CREATE TABLE card_tasks (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    textbook_id uuid,
    source_type card_task_source_type NOT NULL,
    topic varchar(255),
    difficulty card_difficulty DEFAULT 'medium',
    cards_count int,
    cache_key varchar(255),
    temp_s3_key varchar(500),
    temp_file_name varchar(255),
    temp_file_size bigint,
    status card_task_status DEFAULT 'pending',
    position_in_queue int,
    estimated_wait_seconds int,
    error_message text,
    started_at timestamp,
    finished_at timestamp,
    created_at timestamp DEFAULT now(),
    CONSTRAINT fk_card_tasks_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_card_tasks_textbook FOREIGN KEY (textbook_id) REFERENCES textbooks(id)
);

CREATE INDEX idx_card_tasks_user_id ON card_tasks(user_id);
CREATE INDEX idx_card_tasks_textbook_id ON card_tasks(textbook_id);
CREATE INDEX idx_card_tasks_status ON card_tasks(status);
CREATE INDEX idx_card_tasks_created_at ON card_tasks(created_at);
CREATE INDEX idx_card_tasks_cache_key ON card_tasks(cache_key);

CREATE TABLE cards (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id uuid NOT NULL,
    textbook_id uuid,
    chapter varchar(255),
    topic varchar(255),
    subtopic varchar(255),
    question text NOT NULL,
    answer text NOT NULL,
    page_approx int,
    source_reference varchar(500),
    difficulty card_difficulty DEFAULT 'medium',
    report_count int DEFAULT 0,
    created_at timestamp DEFAULT now(),
    CONSTRAINT fk_cards_task FOREIGN KEY (task_id) REFERENCES card_tasks(id),
    CONSTRAINT fk_cards_textbook FOREIGN KEY (textbook_id) REFERENCES textbooks(id)
);

CREATE INDEX idx_cards_task_id ON cards(task_id);
CREATE INDEX idx_cards_textbook_id ON cards(textbook_id);
CREATE INDEX idx_cards_chapter ON cards(chapter);
CREATE INDEX idx_cards_topic ON cards(topic);
CREATE INDEX idx_cards_difficulty ON cards(difficulty);

CREATE TABLE textbook_chunks (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    textbook_id uuid,
    task_id uuid,
    chunk_index int NOT NULL,
    content text NOT NULL,
    page_number int,
    embedding vector(1536),
    created_at timestamp DEFAULT now(),
    CONSTRAINT fk_textbook_chunks_textbook FOREIGN KEY (textbook_id) REFERENCES textbooks(id),
    CONSTRAINT fk_textbook_chunks_task FOREIGN KEY (task_id) REFERENCES card_tasks(id)
);

CREATE INDEX idx_textbook_chunks_textbook_id ON textbook_chunks(textbook_id);
CREATE INDEX idx_textbook_chunks_task_id ON textbook_chunks(task_id);
CREATE INDEX idx_textbook_chunks_chunk_index ON textbook_chunks(chunk_index);

CREATE TABLE card_progress (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    card_id uuid NOT NULL,
    ease_factor double precision DEFAULT 2.5,
    interval_days int DEFAULT 0,
    repetitions int DEFAULT 0,
    next_review_at timestamp NOT NULL,
    last_review_at timestamp,
    last_grade int,
    created_at timestamp DEFAULT now(),
    updated_at timestamp DEFAULT now(),
    CONSTRAINT fk_card_progress_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_card_progress_card FOREIGN KEY (card_id) REFERENCES cards(id),
    CONSTRAINT uq_card_progress_user_card UNIQUE (user_id, card_id)
);

CREATE INDEX idx_card_progress_user_id ON card_progress(user_id);
CREATE INDEX idx_card_progress_card_id ON card_progress(card_id);
CREATE INDEX idx_card_progress_next_review_at ON card_progress(next_review_at);