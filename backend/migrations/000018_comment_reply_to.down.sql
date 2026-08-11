ALTER TABLE comments DROP CONSTRAINT IF EXISTS fk_comments_reply_to;
ALTER TABLE comments DROP COLUMN IF EXISTS reply_to_id;
