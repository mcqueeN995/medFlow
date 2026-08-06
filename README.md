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

Порты (дефолты из `.env.example` — реальные см. в своём `.env`):

| Сервис | URL | Что там |
|---|---|---|
| Сайт (Caddy, весь стек) | http://localhost:8081 | frontend + `/api/*` → backend |
| Backend напрямую | http://localhost:8080 | API, `/health`, `/metrics` |
| Postgres | `localhost:5433` | psql / любой pg-клиент |
| Redis | `localhost:6380` | |
| MinIO API / Console | http://localhost:9000 / :9001 | S3 API / веб-консоль |
| Mailhog UI | http://localhost:8025 | письма, перехваченные в dev |
| Prometheus (профиль `monitoring`) | http://localhost:9090 | сырые метрики, `/targets` |
| Pushgateway (профиль `monitoring`) | http://localhost:9091 | метрики тест-прогонов |
| Grafana (профиль `monitoring`) | http://localhost:3000 | дашборды, admin/admin по умолчанию |

### Карта сайта (`http://localhost:8081/...`)

| Путь | Доступ | Что там |
|---|---|---|
| `/library`, `/library/:id` | гость | каталог учебников, скачивание (A) / переход к первоисточнику (B) |
| `/library/upload` | вход | загрузка своего PDF для генерации карточек |
| `/navigator` | гость | карта кампуса, POI, маршруты |
| `/cards`, `/cards/create`, `/cards/review`, `/cards/tasks/:id` | вход | генерация ИИ-карточек, повторение по SM-2 |
| `/forum`, `/forum/create`, `/forum/:id` | вход | треды, комментарии в 2 уровня, реакции, жалобы |
| `/profile` | вход | профиль, push-уведомления, выход из аккаунта, удаление аккаунта |
| `/admin/reports` | moderator/admin | рассмотрение жалоб |
| `/admin/users`, `/admin/stats`, `/admin/audit-log` | admin | пользователи/роли/баны, статистика, аудит-лог |
| `/login`, `/register`, `/terms`, `/privacy` | гость | вход/регистрация, условия использования, политика конфиденциальности |

### Тестовые аккаунты

Реальные — существуют в БД поднятого стека (`http://localhost:8081`), не в моках:

| Email | Пароль | Роль |
|---|---|---|
| `admin@medflow.local` | `admin12345` | admin — виден весь `/admin/*` |
| `student@sechenov.ru` | `password123` | user — обычный доступ (форум, карточки, библиотека) |

`admin@medflow.local` создан регистрацией, роль на admin выдана вручную (`UPDATE users SET role = 'admin' WHERE email = '...'`) — так и должен создаваться первый админ в системе, см. раздел «Статус модулей» → Navigator/Library.

Отдельно, только для `npm run dev` с `VITE_USE_MOCKS=true` (MSW, без реального backend) — те же email/пароль работают и там, но это независимая in-memory реализация (сбрасывается при перезапуске dev-сервера), не связанная с БД поднятого docker-стека.

Если нужна локальная LLM в контейнере (а не нативный Ollama на хосте):

```bash
make up-ai
```

Если нужны метрики и дашборды (Prometheus + Grafana + Pushgateway, см. раздел «Тесты и мониторинг»):

```bash
make up-monitoring
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
npm run test                # Vitest + Testing Library (юнит/компонентные, на MSW-моках)
npm run test:e2e             # Playwright e2e — против уже поднятого `make up` стека (localhost:8081)
npm run test:report           # test + test:e2e + пуш результатов в Prometheus Pushgateway (нужен `make up-monitoring`)
```

Разработка идёт API-first: контракт правится первым в `backend/docs/openapi.yaml`, дальше — `npm run generate:api`. Пока бэкенд для какого-то модуля не готов, frontend работает на реалистичных стейтфул-моках (MSW, `frontend/src/mocks/handlers/`) с теми же сценариями, что будут на реальном API.

Тестовые аккаунты (реальные и мок-only) — см. раздел «Тестовые аккаунты» выше.

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
| Library (каталог учебников категорий A/B, загрузка PDF) | ✅ | ✅ |
| Cards (RAG-конвейер ИИ-карточек через asynq-воркер, SM-2 повторение) | ✅ | ✅ |
| Navigator (карта кампуса, POI) | ✅ | ✅ |
| Admin-панель (пользователи, жалобы, модерация форума, статистика, аудит-лог) | ✅ | ✅ |
| PWA-полировка (офлайн-баннер, 429, background sync SM-2, Web Push, Lighthouse ≥90) | ✅ | ✅ |
| Тесты и QA (Vitest+Testing Library, Playwright e2e) + мониторинг (Prometheus/Grafana) | ✅ | ✅ |

Guest/User-доступ: гость может листать Библиотеку и Навигатор без входа; Форум и Карточки полностью закрыты — так задано в контракте (`security` не переопределён ни для одного эндпоинта этих модулей в openapi.yaml).

Cards — реальный RAG-конвейер: PDF (`internal/pkg/pdf`) → чанкинг (`internal/pkg/chunker`) → эмбеддинги и векторный поиск по pgvector → генерация строгого JSON через `internal/pkg/llm` → SM-2 (`internal/pkg/sm2`, порт `frontend/src/lib/sm2.ts`). Генерация асинхронная — `POST /cards/tasks` кладёт задачу в очередь asynq (Redis), обрабатывает отдельный `worker`-контейнер. Эмбеддинги всегда идут через Ollama (см. `internal/pkg/llm`) — без поднятого Ollama (или без `LLM_API_KEY` для облачного провайдера) задачи корректно уходят в `status=failed` с понятным `error_message`, каталог/загрузка/rate-limit при этом работают и без LLM.

Navigator — расстояние/время пешком считаются формулой Haversine (`internal/service/poi_service.go`, точный порт `frontend/src/lib/geo.ts`) относительно `lat`/`lon` из запроса; если вместо этого передан `campus_id`, значения берутся из заранее сохранённой связи `poi_campus_links` (управляемого API для этой таблицы пока нет — заполняется вручную/будущей admin-панелью). Наполнение каталога POI, как и Library, требует роли `admin` (`/admin/poi`).

Наполнение каталога Библиотеки (`POST/PATCH/DELETE /admin/library/textbooks`) требует роли `admin` — роль меняется через Admin-панель (вкладка «Пользователи»), но самому первому админу в системе выдать её можно только вручную: `UPDATE users SET role = 'admin' WHERE email = '...';` в контейнере `postgres`, дальше перелогиниться, чтобы роль попала в JWT.

PWA-полировка: офлайн-баннер (`useOnlineStatus`) и обработка 429 (`frontend/src/api/axios-instance.ts`, читает `Retry-After` от `middleware.RateLimiter`) — на фронте глобально; оценки SM-2, поставленные офлайн, уходят в очередь на IndexedDB (`frontend/src/lib/review-queue.ts`) и досылаются при восстановлении связи (`sync-review-queue.ts`) — это не браузерный Background Sync API (плохая кроссбраузерная поддержка), а собственная реализация "sync on reconnect" с тем же результатом для пользователя. Web Push (VAPID) — `internal/service/push_service.go`, реально триггерится на ответ в форуме (`thread_reply`/`comment_reply`) и на завершение/провал генерации карточек (`card_task_done`/`card_task_failed`); подписка/настройки — в профиле. Service worker собран в режиме `injectManifest` (`frontend/src/sw.ts`) — прекэш app shell + `push`/`notificationclick`, регистрируется только в прод-сборке (в dev конфликтовал бы по scope с MSW-воркером).

Lighthouse (локально, `npx lighthouse http://localhost:8081 --only-categories=performance,accessibility,best-practices,seo`): Performance 92, Accessibility 94, Best Practices 100, SEO 100. Отдельной категории "PWA" в CLI Lighthouse v13+ больше нет (аудиты manifest/service-worker перенесены в Chrome DevTools) — корректность манифеста и регистрация SW проверены вручную (`curl`/Playwright).

`/terms` и `/privacy` (`frontend/src/features/auth/{terms,privacy}-page.tsx`) — статические страницы условий использования и политики конфиденциальности; раньше ссылки на них со страниц входа/регистрации вели в никуда (роутов не существовало).

**Все 8 этапов ТЗ (`medFlowObsidian/Техническое задание...md`, раздел 10) закрыты**: Фундамент → Библиотека → ИИ-карточки → Треды → Навигатор → Админ-панель → PWA Frontend → Полировка (тесты, Lighthouse, юридические страницы, мониторинг).

## Тесты и мониторинг

**Vitest + Testing Library** (`frontend/src/**/*.test.{ts,tsx}`) — юнит-тесты чистой логики (`lib/sm2.ts`, `lib/review-queue.ts`) и компонентные тесты ключевых экранов (`LoginPage`, `ReviewPage`, `OfflineBanner`, `useOnlineStatus`, 429-интерцептор). Переиспользует те же стейтфул MSW-хендлеры, что и dev-режим (`frontend/src/mocks/handlers/`), через `msw/node` (`frontend/src/test/setup.ts`) — один источник правды для мок-поведения. `npm run test`.

Node 22+/25 добавил нативный `localStorage`, конфликтующий с jsdom-окружением тестов (ломает Zustand `persist`) — поэтому `test`/`test:watch` в `package.json` явно запускаются с `NODE_OPTIONS=--no-experimental-webstorage`.

**Playwright e2e** (`frontend/e2e/*.spec.ts`) — гоняются против уже поднятого `make up` стека (`http://localhost:8081`, переопределяется `E2E_BASE_URL`), а не dev-сервера. Критические пути: `auth.spec.ts` (регистрация/логин/logout/гостевой доступ), `cards-generation.spec.ts` (асинхронный конвейер генерации карточек до terminal-статуса — без LLM-провайдера в окружении детерминированно `failed`, это тоже валидный проверяемый исход), `sm2-review.spec.ts`, `offline.spec.ts` (офлайн-баннер + background sync SM-2). SM-2/офлайн-сценариям для повторяемости нужна гарантированно "просроченная" карточка — `frontend/e2e/fixtures/db.ts` сеет её напрямую в Postgres (`pg`-клиент, те же креды из корневого `.env`), а не полагается на реальный RAG-конвейер. Каждый спек создаёт своего тестового пользователя и подчищает его в `afterEach`. `npm run test:e2e`.

**Prometheus + Grafana + Pushgateway** (профиль `monitoring`, `make up-monitoring`) — Prometheus скрейпит `backend:8080/metrics` (`internal/middleware/metrics.go`: счётчик запросов и латency по методу/роуту/статусу) и Pushgateway; Grafana поднимается с уже провижененными датасорсом и дашбордом `medFlow — Overview` (`infra/grafana/`) — открыл `http://localhost:3000` (admin/admin) и сразу видно графики, без ручной настройки. Результаты тест-прогонов туда тоже попадают: `npm run test:report` гоняет Vitest+Playwright с JSON-репортёрами и пушит агрегаты (`frontend/scripts/push-test-metrics.mjs`) в Pushgateway — панели "Тесты" на дашборде показывают passed/failed/total по каждому набору и время последнего прогона. Прямой push, а не отдельный CI — в репозитории нет `.github/workflows`, тесты гоняются по требованию.

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
    test/setup.ts                  — Vitest + msw/node, переиспользует mocks/handlers/
    **/*.test.{ts,tsx}               — Vitest-тесты рядом с тестируемым кодом
  e2e/                          — Playwright e2e (против поднятого docker-стека, не dev-сервера)
  scripts/push-test-metrics.mjs — пуш результатов Vitest/Playwright в Pushgateway

infra/
  docker-compose.yml, Caddyfile, Dockerfile-ы backend/frontend
  prometheus/prometheus.yml     — scrape-конфиг (профиль monitoring)
  grafana/                       — provisioning (датасорс, дашборд) + сам дашборд (профиль monitoring)
```

Рядом с этим репозиторием (не внутри него, на уровень выше) лежит `medFlowObsidian` — ТЗ и дизайн-референсы, не билд-зависимость.

## Переменные окружения

Полный список с комментариями — в [`.env.example`](.env.example). Ключевое:

- `POSTGRES_*`, `REDIS_*`, `MINIO_*` — порты и креды инфраструктуры (порты в примере — дефолтные для контейнеров, для доступа с хоста см. актуальный `.env`)
- `JWT_ACCESS_SECRET` / `JWT_REFRESH_SECRET` — обязательно сменить перед продом
- `LLM_PROVIDER` — `ollama` для локальной разработки, `deepseek`/`qwen`/`openrouter` для облака (прод по умолчанию — облако; эмбеддинги всегда идут через Ollama, у облачных провайдеров нет единого контракта на эмбеддинги)
- `HTTP_PORT` — порт, на котором слушает Caddy (весь стек снаружи)
- `S3_ENDPOINT` (internal, `minio:9000`) vs `S3_PUBLIC_ENDPOINT` (`localhost:9000` в dev) — presigned-ссылки на скачивание PDF подписываются под второй, иначе они были бы нерабочими для браузера, у которого нет DNS-имени `minio`
- `VAPID_PUBLIC_KEY` / `VAPID_PRIVATE_KEY` / `VAPID_SUBJECT` — ключевая пара для Web Push, сгенерировать один раз (`webpush.GenerateVAPIDKeys()` из `github.com/SherClockHolmes/webpush-go`) и сменить перед продом; публичный ключ не секрет — прокидывается на фронт как `VITE_VAPID_PUBLIC_KEY` build-arg'ом (см. `infra/docker-compose.yml`), из того же `VAPID_PUBLIC_KEY`
- `PROMETHEUS_PORT` / `PUSHGATEWAY_PORT` / `GRAFANA_PORT` / `GRAFANA_ADMIN_PASSWORD` — только для профиля `monitoring` (`make up-monitoring`), пароль сменить перед продом
