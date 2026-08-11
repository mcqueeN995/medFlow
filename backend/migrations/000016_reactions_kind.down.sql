DELETE FROM reactions WHERE kind = 'vote';
ALTER TABLE reactions DROP CONSTRAINT IF EXISTS chk_reactions_vote_emoji;
ALTER TABLE reactions DROP CONSTRAINT IF EXISTS uq_reactions_user_target;
ALTER TABLE reactions ADD CONSTRAINT uq_reactions_user_target UNIQUE (user_id, target_type, target_id);
ALTER TABLE reactions DROP COLUMN IF EXISTS kind;
DROP TYPE IF EXISTS reaction_kind;
