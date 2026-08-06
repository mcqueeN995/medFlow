CREATE TABLE uploads (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    uploader_id uuid NOT NULL,
    upload_type varchar(20) NOT NULL,
    s3_key varchar(500) NOT NULL,
    mime_type varchar(100) NOT NULL,
    size_bytes bigint NOT NULL,
    expires_at timestamp,
    created_at timestamp DEFAULT now(),
    CONSTRAINT fk_uploads_uploader FOREIGN KEY (uploader_id) REFERENCES users(id)
);

CREATE INDEX idx_uploads_uploader_id ON uploads(uploader_id);
CREATE INDEX idx_uploads_expires_at ON uploads(expires_at);
