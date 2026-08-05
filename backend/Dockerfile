# Общий Dockerfile для api и worker — оба живут в одном модуле cmd/*.
# Какой бинарь собрать, выбирается build-arg'ом CMD_PATH (см. infra/docker-compose.yml).
ARG GO_VERSION=1.26
ARG CMD_PATH=cmd/api

FROM golang:${GO_VERSION} AS build
ARG CMD_PATH
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app ./${CMD_PATH}

FROM alpine:3.21 AS runtime
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/app /app/app
USER app
ENTRYPOINT ["/app/app"]
