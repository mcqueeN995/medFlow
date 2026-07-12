.PHONY: up down logs migrate migrate-create migrate-down

# По умолчанию используем .env из корня
ENV_FILE := .env
COMPOSE_FILE := infra/docker-compose.yml

# Поднять все сервисы в фоне
up:
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) up -d

# Остановить и удалить контейнеры (данные в volumes сохраняются)
down:
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) down

# Смотреть логи всех сервисов
logs:
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) logs -f

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