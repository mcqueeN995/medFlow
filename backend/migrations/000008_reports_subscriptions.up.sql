CREATE TABLE reports (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id uuid NOT NULL,
    target_type varchar(50) NOT NULL,
    target_id uuid NOT NULL,
    reason text NOT NULL,
    status report_status DEFAULT 'pending',
    reviewed_by uuid,
    reviewed_at timestamp,
    resolution_note text,
    created_at timestamp DEFAULT now(),
    CONSTRAINT fk_reports_reporter FOREIGN KEY (reporter_id) REFERENCES users(id),
    CONSTRAINT fk_reports_reviewed_by FOREIGN KEY (reviewed_by) REFERENCES users(id)
);

CREATE INDEX idx_reports_reporter_id ON reports(reporter_id);
CREATE INDEX idx_reports_status ON reports(status);
CREATE INDEX idx_reports_target ON reports(target_type, target_id);
CREATE INDEX idx_reports_created_at ON reports(created_at);

CREATE TABLE subscriptions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    target_type subscription_target_type NOT NULL,
    target_id uuid NOT NULL,
    created_at timestamp DEFAULT now(),
    CONSTRAINT fk_subscriptions_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT uq_subscriptions_user_target UNIQUE (user_id, target_type, target_id)
);

CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_target ON subscriptions(target_type, target_id);
