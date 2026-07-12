CREATE TABLE textbooks (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title varchar(500) NOT NULL,
    authors text,
    isbn varchar(20),
    year int,
    pages int,
    description text,
    subject varchar(100),
    course int,
    department varchar(100),
    storage_type textbook_storage_type NOT NULL,
    license_type textbook_license_type,
    copyright_holder varchar(255),
    hidden_at timestamp,
    hidden_by uuid,
    hidden_reason text,
    deleted_at timestamp,
    created_at timestamp DEFAULT now(),
    updated_at timestamp DEFAULT now()
);

CREATE INDEX idx_textbooks_storage_type ON textbooks(storage_type);
CREATE INDEX idx_textbooks_subject ON textbooks(subject);
CREATE INDEX idx_textbooks_course ON textbooks(course);
CREATE INDEX idx_textbooks_department ON textbooks(department);
CREATE INDEX idx_textbooks_hidden_at ON textbooks(hidden_at);
CREATE INDEX idx_textbooks_deleted_at ON textbooks(deleted_at);

ALTER TABLE textbooks ADD CONSTRAINT fk_textbooks_hidden_by FOREIGN KEY (hidden_by) REFERENCES users(id);

CREATE TABLE textbook_files (
    textbook_id uuid PRIMARY KEY,
    pdf_s3_key varchar(500) NOT NULL,
    created_at timestamp DEFAULT now(),
    updated_at timestamp DEFAULT now(),
    CONSTRAINT fk_textbook_files_textbook FOREIGN KEY (textbook_id) REFERENCES textbooks(id)
);

CREATE TABLE textbook_links (
    textbook_id uuid PRIMARY KEY,
    source_url varchar(500) NOT NULL,
    source_name varchar(255),
    created_at timestamp DEFAULT now(),
    updated_at timestamp DEFAULT now(),
    CONSTRAINT fk_textbook_links_textbook FOREIGN KEY (textbook_id) REFERENCES textbooks(id)
);
