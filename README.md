# medFlow

PWA-сервис для студентов-медиков Москвы: библиотека легальных учебников, ИИ-генерация карточек с интервальным повторением (SM-2), карта кампуса и форум.

Контракт API: [`backend/docs/openapi.yaml`](backend/docs/openapi.yaml) — источник истины при любых изменениях API. Frontend генерирует типы и клиент из него через orval.

## Стек

**Backend** — Go 1.26, Gin, pgx (PostgreSQL 15 + pgvector), Redis, asynq (очереди), JWT-аутентификация.

**Frontend** — React 19, TypeScript, Vite, Tailwind CSS v4, shadcn/ui (стиль `base-nova` на Base UI), Zustand, React Router v7, MSW (моки для разработки без готового бэкенда).

**Инфраструктура** — Docker Compose: Postgres, Redis, MinIO (S3-совместимое хранилище), Mailhog (SMTP-заглушка), Caddy (реверс-прокси + автоматический HTTPS при появлении домена), опционально Ollama для локальной LLM.

## Быстрый старт

```bash
cp .env.example .env   # и поправьте секреты при необходимости
make up                # поднимает весь стек в фоне, миграции применяются автоматически
make logs               # логи всех сервисов
make down                # остановить (данные в volumes сохраняются)
```

После `make up` стек доступен на `http://localhost:8081` (порт задаётся `HTTP_PORT` в `.env`) — Caddy проксирует `/api/*` и `/health` на backend, всё остальное отдаёт статику frontend.

Если нужна локальная LLM в контейнере (а не нативный Ollama на хосте):

```bash
make up-ai
```

## Разработка

### Backend

Локального Go-тулчейна на машине может не быть — все команды удобно гонять через контейнер:

```bash
docker run --rm -v "$(pwd)/backend:/src" -w /src golang:1.26 go build ./...
docker run --rm -v "$(pwd)/backend:/src" -w /src golang:1.26 go vet ./...
docker run --rm --network host -v "$(pwd)/backend:/src" -w /src golang:1.26 go test ./...
```

`--network host` в тестах нужен репозиторным тестам — они бьют напрямую в Postgres из `docker-compose` по `localhost:5433` (порт с хоста, см. `POSTGRES_PORT` в `.env`).

Миграции (`golang-migrate`, SQL в `backend/migrations/`):

```bash
make migrate           # применить все
make migrate-down      # откатить последнюю
make migrate-create    # создать новую (спросит имя)
```

### Frontend

```bash
cd frontend
npm install
npm run dev             # http://localhost:5173, с моками (VITE_USE_MOCKS=true в .env)
npm run build            # tsc -b + vite build
npm run lint              # oxlint
npm run generate:api       # перегенерировать клиент из openapi.yaml после изменения контракта
```

Разработка идёт API-first: контракт правится первым в `backend/docs/openapi.yaml`, дальше — `npm run generate:api`. Пока бэкенд для какого-то модуля не готов, frontend работает на реалистичных стейтфул-моках (MSW, `frontend/src/mocks/handlers/`) с теми же сценариями, что будут на реальном API.

Тестовые аккаунты в моках: `student@sechenov.ru` / `password123` (обычный пользователь), `admin@medflow.local` / `admin12345` (админ).

### Пересборка Docker-образов после изменений

```bash
docker compose --env-file .env -f infra/docker-compose.yml build <service>
docker compose --env-file .env -f infra/docker-compose.yml up -d <service>
```

`--env-file .env` обязателен, если команда выполняется не из `infra/` — иначе `${POSTGRES_USER}` и подобные переменные в `docker-compose.yml` не подставятся.

## Статус модулей

| Модуль | Backend (реальный API) | Frontend |
|---|---|---|
| Auth (регистрация, логин, refresh, logout) | ✅ | ✅ |
| Users (профиль, редактирование, удаление аккаунта) | ✅ | ✅ |
| Forum (треды, комментарии в 2 уровня, реакции, жалобы) | ✅ | ✅ |
| Library (каталог учебников категорий A/B, загрузка PDF) | — | ✅ (моки) |
| Cards (ИИ-генерация карточек, SM-2 повторение) | — | ✅ (моки) |
| Navigator (карта кампуса, POI) | — | ✅ (моки) |
| `backend/internal/pkg/llm` (Ollama + облачные провайдеры) | ✅ (пакет готов, без потребителя) | — |
| Admin-панель | — | — |

Guest/User-доступ: гость может листать Библиотеку и Навигатор без входа; Форум и Карточки полностью закрыты — так задано в контракте (`security` не переопределён ни для одного эндпоинта этих модулей в openapi.yaml).

## Структура репозитория

```
backend/
  cmd/api, cmd/worker     — точки входа (HTTP-сервер и asynq-воркер)
  internal/
    handler, service, repository, models, dto  — слои приложения
    middleware               — auth, CORS, rate limiter, recovery, request-id
    pkg/{jwt,password,llm}    — переиспользуемые пакеты без бизнес-логики модулей
  migrations                 — SQL-миграции (golang-migrate)
  docs/openapi.yaml           — контракт API

frontend/
  src/
    app/router.tsx            — маршруты, гейты по авторизации
    features/                  — по одному каталогу на модуль (auth, library, cards, navigator, forum, profile)
    components/{ui,shared}      — shadcn-примитивы и общие компоненты (AppShell, гейты, тема)
    mocks/handlers/               — стейтфул MSW-моки, один файл на модуль
    api/generated/                — сгенерировано orval'ом из openapi.yaml, руками не редактировать
    stores/                        — Zustand (auth, UI-состояние)

infra/
  docker-compose.yml, Caddyfile, Dockerfile-ы backend/frontend
```

Рядом с этим репозиторием (не внутри него, на уровень выше) лежит `medFlowObsidian` — ТЗ и дизайн-референсы, не билд-зависимость.

## Переменные окружения

Полный список с комментариями — в [`.env.example`](.env.example). Ключевое:

- `POSTGRES_*`, `REDIS_*`, `MINIO_*` — порты и креды инфраструктуры (порты в примере — дефолтные для контейнеров, для доступа с хоста см. актуальный `.env`)
- `JWT_ACCESS_SECRET` / `JWT_REFRESH_SECRET` — обязательно сменить перед продом
- `LLM_PROVIDER` — `ollama` для локальной разработки, `deepseek`/`qwen`/`openrouter` для облака (прод по умолчанию — облако; эмбеддинги всегда идут через Ollama, у облачных провайдеров нет единого контракта на эмбеддинги)
- `HTTP_PORT` — порт, на котором слушает Caddy (весь стек снаружи)
