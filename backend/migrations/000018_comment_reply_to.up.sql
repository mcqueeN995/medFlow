-- reply_to_id - конкретный комментарий, которому реально отвечают, отдельно
-- от parent_id: дерево комментариев ограничено 2 уровнями (см.
-- ForumService.CreateComment), ответ на ответ "схлопывается" к родителю
-- верхнего уровня в parent_id, из-за чего терялось, кому именно отвечали.
-- reply_to_id хранит исходного адресата - фронтенд резолвит его автора из
-- уже загруженного списка реплаев того же родителя (без лишнего запроса).
ALTER TABLE comments ADD COLUMN reply_to_id uuid;
ALTER TABLE comments ADD CONSTRAINT fk_comments_reply_to FOREIGN KEY (reply_to_id) REFERENCES comments(id);
