.PHONY: up up-ai down build logs logs-backend migrate migrate-create migrate-down

# По умолчанию используем .env из корня
ENV_FILE := .env
COMPOSE_FILE := infra/docker-compose.yml
COMPOSE := docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE)

# Собрать образы backend/worker/frontend
build:
	$(COMPOSE) build

# Поднять весь стек (postgres/redis/minio/backend/worker/frontend/caddy) в фоне.
# Миграции применяются автоматически одноразовым сервисом `migrate`.
up:
	$(COMPOSE) up -d

# То же самое + локальная Ollama в контейнере (профиль ai).
# На macOS для разработки быстрее поставить Ollama нативно на хост, см. .env.example.
up-ai:
	$(COMPOSE) --profile ai up -d

# Остановить и удалить контейнеры (данные в volumes сохраняются)
down:
	$(COMPOSE) down

# Смотреть логи всех сервисов
logs:
	$(COMPOSE) logs -f

# Смотреть логи только backend/worker
logs-backend:
	$(COMPOSE) logs -f backend worker

# Применить все SQL миграции
migrate:
	@echo "Applying database migrations..."
	@export $$(grep -v '^#' .env | xargs) && \
	migrate -path backend/migrations -database "postgres://$${POSTGRES_USER}:$${POSTGRES_PASSWORD}@localhost:$${POSTGRES_PORT}/$${POSTGRES_DB}?sslmode=disable" up

# Откатить последнюю миграцию
migrate-down:
	@echo "Rolling back 1 migration..."
	@export $$(grep -v '^#' .env | xargs) && \
	migrate -path backend/migrations -database "postgres://$${POSTGRES_USER}:$${POSTGRES_PASSWORD}@localhost:$${POSTGRES_PORT}/$${POSTGRES_DB}?sslmode=disable" down 1
# Создать файл для новой миграции
migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir backend/migrations -seq $$name