-- Голосование за комментарии (up/down) переиспользует таблицу reactions
-- вместо отдельной таблицы, но эмодзи-лайк и голос должны сосуществовать
-- независимо на одной и той же цели - поэтому вводим дискриминатор kind и
-- расширяем уникальность до 4 колонок (было user_id, target_type, target_id).
CREATE TYPE reaction_kind AS ENUM ('emoji', 'vote');

ALTER TABLE reactions ADD COLUMN kind reaction_kind NOT NULL DEFAULT 'emoji';

ALTER TABLE reactions DROP CONSTRAINT uq_reactions_user_target;
ALTER TABLE reactions ADD CONSTRAINT uq_reactions_user_target UNIQUE (user_id, target_type, target_id, kind);

ALTER TABLE reactions ADD CONSTRAINT chk_reactions_vote_emoji CHECK (kind <> 'vote' OR emoji IN ('up', 'down'));
