-- Честный счётчик просмотров треда: до этой миграции ThreadRepo.IncrementViews
-- инкрементировал views_count безусловно на каждый GET, так что один и тот
-- же пользователь, перезаходя в тред, накручивал счётчик до бесконечности.
-- Теперь просмотр засчитывается не чаще раза в 24 часа на пользователя
-- (форум целиком закрыт для гостя, поэтому весь трекинг - по user_id, без IP).
CREATE TABLE thread_views (
    user_id uuid NOT NULL REFERENCES users(id),
    thread_id uuid NOT NULL REFERENCES threads(id),
    last_viewed_at timestamp NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, thread_id)
);

CREATE INDEX idx_thread_views_thread_id ON thread_views(thread_id);
