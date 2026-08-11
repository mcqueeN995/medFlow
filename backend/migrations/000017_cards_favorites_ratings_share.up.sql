-- Избранное карточек - независимо от SM-2 прогресса повторения (card_progress),
-- отдельный список "хочу учить эти", кросс-юзерно доступный на
-- catalog_textbook-карточках (см. ослабление владения в CardService.GetTask/
-- ListTaskCards/authorizeCardAccess).
CREATE TABLE card_favorites (
    user_id uuid NOT NULL REFERENCES users(id),
    card_id uuid NOT NULL REFERENCES cards(id),
    created_at timestamp DEFAULT now(),
    PRIMARY KEY (user_id, card_id)
);

CREATE INDEX idx_card_favorites_card_id ON card_favorites(card_id);

-- Рейтинг звёздами - отдельно от избранного и от reactions (форумных лайков),
-- по одной оценке 1-5 на пользователя на карточку.
CREATE TABLE card_ratings (
    user_id uuid NOT NULL REFERENCES users(id),
    card_id uuid NOT NULL REFERENCES cards(id),
    stars smallint NOT NULL CHECK (stars BETWEEN 1 AND 5),
    created_at timestamp DEFAULT now(),
    updated_at timestamp DEFAULT now(),
    PRIMARY KEY (user_id, card_id)
);

CREATE INDEX idx_card_ratings_card_id ON card_ratings(card_id);

-- Публичная ссылка на набор карточек задачи (шеринг) - владелец задачи
-- включает/выключает, токен уникален и непредсказуем (см.
-- service.GenerateRandomToken), публичный эндпоинт отдаёт по нему только
-- задачи в статусе done (см. CardService.GetSharedTask).
ALTER TABLE card_tasks ADD COLUMN share_token varchar(64) UNIQUE;
