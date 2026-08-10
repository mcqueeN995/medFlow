# Прод-сборка PWA: Vite build -> статика за nginx.
# Локальная разработка использует `npm run dev` на хосте, не этот образ (см. README).
FROM node:22-alpine AS build
WORKDIR /src

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

# VITE_* встраиваются в бандл на этапе сборки, а не читаются в рантайме контейнера.
ARG VITE_API_URL=/api/v1
ARG VITE_USE_MOCKS=false
ARG VITE_VAPID_PUBLIC_KEY=""
ARG VITE_YANDEX_MAPS_API_KEY=""
ENV VITE_API_URL=${VITE_API_URL}
ENV VITE_USE_MOCKS=${VITE_USE_MOCKS}
ENV VITE_VAPID_PUBLIC_KEY=${VITE_VAPID_PUBLIC_KEY}
ENV VITE_YANDEX_MAPS_API_KEY=${VITE_YANDEX_MAPS_API_KEY}
RUN npm run build

FROM nginx:1.27-alpine AS runtime
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html
EXPOSE 80
