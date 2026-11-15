# Utils — HTTP API пользователей на Go

Учебный HTTP-сервер на **Go** с SQLite. Проект использует только стандартный
`net/http` без веб-фреймворков. Данные пользователей хранятся в SQLite, пароли —
только в виде **Argon2id**-хешей.

## Возможности

- `POST /adduser` — создание пользователя (имя + необязательные теги).
- `GET /user/<id>` — получение пользователя по `id`.
- `PUT /user/<id>/password` — установка или замена пароля.
- `POST /user/<id>/password/verify` — проверка пароля по Argon2id-хешу.
- `GET /active` — недавно созданные или прочитанные пользователи.
- Автоматическое добавление тега `"new"` и удаление дублей тегов.
- Потокобезопасный список активных пользователей с настраиваемым лимитом.
- Graceful shutdown по `SIGINT`/`SIGTERM`, таймауты HTTP-сервера.
- SQLite: автосоздание схемы, миграция `password_hash`, режим WAL,
  параметризованные запросы.
- JSON-ответы со статусами: 200, 201, 204, 400, 404, 405, 409, 500.
- Unit-тесты для всех пакетов, `go vet`, `gofmt`, CI на GitHub Actions.

## Структура проекта

```
Utils/
├── cmd/
│   └── server/
│       └── main.go           # точка входа, сборка зависимостей, shutdown
├── internal/
│   ├── config/               # настройки из переменных окружения
│   ├── models/               # User и доменные ошибки
│   ├── database/             # SQLite, схема, миграция, WAL
│   ├── repository/           # UserRepository: доступ к данным
│   ├── services/
│   │   ├── password/         # PasswordService: Argon2id
│   │   └── presence/         # ActiveUsers: потокобезопасный список
│   ├── handlers/             # HTTP: server, routes, users, request, response
│   └── utils/                # общие проверки имени и тегов
├── .github/workflows/ci.yml  # CI: форматирование, vet, build, тесты
├── Dockerfile                # образ для запуска
├── docker-compose.yml        # запуск через Docker Compose
├── .env.example              # пример переменных окружения
├── .dockerignore
├── .gitignore
├── Makefile                  # build/run/test/cover/lint/docker
├── go.mod                    # описание модуля и зависимостей
├── go.sum                    # контрольные суммы зависимостей
├── openapi.yaml              # спецификация OpenAPI 3.1
├── LICENSE                   # MIT
└── README.md                 # этот файл
```

Тесты лежат рядом с кодом в файлах `*_test.go`.

## Требования

- **Go 1.22+**
- **gcc** и заголовки SQLite для CGO (драйвер `mattn/go-sqlite3`).
  В Ubuntu/Debian: `sudo apt install gcc libc6-dev`.
- Для Docker-сборки дополнительно ничего не нужно — зависимости ставятся в образе.

## Установка и запуск (без Docker)

1. Загрузите зависимости:

   ```bash
   go mod download
   ```

2. Запустите сервер:

   ```bash
   go run ./cmd/server
   ```

   Сервер будет доступен по адресу `http://127.0.0.1:8000`.

3. Задайте порт и другие параметры через переменные окружения:

   ```bash
   PORT=8080 DB_PATH=/tmp/users.db ACTIVE_USERS_LIMIT=10 go run ./cmd/server
   ```

> В Windows PowerShell переменные задаются так:
> `$env:PORT = "8080"; go run ./cmd/server`.

## Переменные окружения

| Переменная | По умолчанию | Назначение |
| --- | --- | --- |
| `PORT` | `8000` | Порт, на котором слушает сервер |
| `DB_PATH` | `users.db` в рабочей директории | Путь к файлу SQLite |
| `ACTIVE_USERS_LIMIT` | `5` | Размер списка активных пользователей |

```bash
export DB_PATH=/tmp/users.db
export ACTIVE_USERS_LIMIT=10
export PORT=8080
go run ./cmd/server
```

## API

Базовый URL: `http://127.0.0.1:8000`. Все ответы — JSON в кодировке UTF-8.
Тело запроса ограничено 1 МиБ и должно содержать ровно один JSON-объект.

Тело ошибки:

```json
{ "error": "user not found" }
```

### `POST /adduser`

Создаёт пользователя. Пробелы по краям `name` обрезаются. Тег `new`
добавляется автоматически, дубли тегов удаляются. Созданный пользователь
попадает в список активных.

Тело запроса:

```json
{
  "name": "alice",
  "tags": ["vip"]
}
```

- `name` — обязательная непустая строка.
- `tags` — необязательный массив непустых строк.

Успешный ответ — `201 Created`:

```json
{ "id": 1 }
```

| Код | Причина |
| --- | --- |
| `400` | Тело не JSON-объект, пустое/нестроковое `name`, `tags` не массив строк |
| `405` | HTTP-метод не поддерживается |
| `409` | Пользователь с таким именем уже существует |
| `500` | Внутренняя ошибка сервера |

```bash
curl -X POST http://127.0.0.1:8000/adduser \
  -H "Content-Type: application/json" \
  -d '{"name":"alice","tags":["vip"]}'
```

### `GET /user/<id>`

Возвращает пользователя по числовому `id`. Пароль не возвращается.
Успешно полученный пользователь попадает в список активных.

Успешный ответ — `200 OK`:

```json
{
  "id": 1,
  "name": "alice",
  "tags": ["vip", "new"]
}
```

| Код | Причина |
| --- | --- |
| `404` | Пользователь не найден или `id` не является числом |
| `405` | HTTP-метод не поддерживается |
| `500` | Внутренняя ошибка сервера |

```bash
curl http://127.0.0.1:8000/user/1
```

### `PUT /user/<id>/password`

Устанавливает или заменяет пароль. Пароль хешируется через Argon2id,
сохраняется только хеш. Пароль не возвращается и не логируется.

Тело запроса:

```json
{ "password": "s3cret" }
```

- `password` — обязательная непустая строка длиной до 1024 символов.

Успешный ответ — `204 No Content` без тела.

| Код | Причина |
| --- | --- |
| `400` | Тело не JSON-объект, пустой/слишком длинный `password` |
| `404` | Пользователь не найден или `id` не является числом |
| `405` | HTTP-метод не поддерживается |
| `500` | Внутренняя ошибка сервера |

```bash
curl -i -X PUT http://127.0.0.1:8000/user/1/password \
  -H "Content-Type: application/json" \
  -d '{"password":"s3cret"}'
```

### `POST /user/<id>/password/verify`

Проверяет пароль по сохранённому хешу. Если пароль ещё не установлен,
возвращает `valid: false` со статусом `200`.

Тело запроса:

```json
{ "password": "s3cret" }
```

Успешный ответ — `200 OK`:

```json
{ "valid": true }
```

| Код | Причина |
| --- | --- |
| `400` | Тело не JSON-объект, пустой/слишком длинный `password` |
| `404` | Пользователь не найден или `id` не является числом |
| `405` | HTTP-метод не поддерживается |
| `500` | Внутренняя ошибка сервера |

```bash
curl -X POST http://127.0.0.1:8000/user/1/password/verify \
  -H "Content-Type: application/json" \
  -d '{"password":"s3cret"}'
```

### `GET /active`

Возвращает идентификаторы недавно созданных или прочитанных пользователей.
Список хранится в памяти, ограничен `ACTIVE_USERS_LIMIT`, упорядочен по
времени добавления и сбрасывается при перезапуске сервера. Идентификаторы
уникальны: повторный доступ перемещает пользователя в конец списка.

Успешный ответ — `200 OK`:

```json
{ "active_users": [1, 2] }
```

| Код | Причина |
| --- | --- |
| `405` | HTTP-метод не поддерживается |
| `500` | Внутренняя ошибка сервера |

```bash
curl http://127.0.0.1:8000/active
```

## OpenAPI

Спецификация API — [`openapi.yaml`](openapi.yaml) (OpenAPI 3.1).

Просмотр и проверка (требуется Node.js):

```bash
npx @redocly/cli lint openapi.yaml
npx @redocly/cli preview-docs openapi.yaml
```

## Тестирование

```bash
# Все тесты
go test ./...

# С проверкой гонок
go test -race ./...

# Покрытие
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

Или через Makefile:

```bash
make test
make test-race
make cover
```

## Makefile

| Цель | Действие |
| --- | --- |
| `make build` | Собрать бинарник в `bin/server` |
| `make run` | Запустить сервер через `go run` |
| `make test` | Запустить тесты |
| `make test-race` | Тесты с `-race` |
| `make cover` | Отчёт покрытия |
| `make vet` | `go vet ./...` |
| `make fmt` | Форматировать код |
| `make fmt-check` | Проверить форматирование (для CI) |
| `make ci` | `fmt-check` + `vet` + `test-race` |
| `make docker-build` | Собрать Docker-образ |
| `make docker-up` | `docker compose up -d --build` |
| `make docker-down` | `docker compose down` |
| `make clean` | Удалить `bin/` и `coverage.out` |

## CI

GitHub Actions workflow [`.github/workflows/ci.yml`](.github/workflows/ci.yml)
на каждый push и pull request проверяет `gofmt`, запускает `go vet`,
`go build` и `go test -race` с отчётом покрытия.

## Docker

Ниже описаны локальный запуск, публикация образа в Docker Hub и развёртывание
на сервере с HTTPS, обновлением, откатом и резервным копированием.

### Docker Compose (рекомендуется)

Запуск из корня проекта:

```bash
docker compose up -d --build
docker compose ps
docker compose logs -f api
```

Остановка без удаления данных:

```bash
docker compose down
```

Удаление контейнера вместе с volume и базой данных:

```bash
docker compose down -v
```

Внешний порт задаётся переменной `API_PORT` (по умолчанию `8000`), лимит
активных пользователей — `ACTIVE_USERS_LIMIT`:

```bash
API_PORT=8080 ACTIVE_USERS_LIMIT=10 docker compose up -d --build
```

Переменные также читаются из файла `.env` (пример — `.env.example`).

### Без Docker Compose

Сборка образа:

```bash
docker build -t utils-server-go .
```

Локальный запуск:

```bash
docker run --rm -p 8000:8000 -v utils-data:/data --name utils-server-go utils-server-go
```

Запуск на сервере в фоне:

```bash
docker run -d --restart unless-stopped --name utils-server-go \
  -p 8000:8000 -v utils-data:/data -e ACTIVE_USERS_LIMIT=10 utils-server-go
```

Полезные команды:

```bash
docker logs -f utils-server-go
docker stop utils-server-go
docker rm utils-server-go
```

### Публикация в Docker Hub

Ниже — полный путь от локальной сборки до образа в Docker Hub. В примерах
`<dockerhub-username>` — ваш логин на Docker Hub.

#### 1. Аккаунт и репозиторий

1. Зарегистрируйтесь на [Docker Hub](https://hub.docker.com) и включите
   двухфакторную аутентификацию.
2. Создайте репозиторий: Repositories → Create repository.
   - Name: `utils-server-go`
   - Visibility: `Public` или `Private`.
3. Итоговое имя образа:

   ```text
   <dockerhub-username>/utils-server-go
   ```

#### 2. Access Token

Пароль от аккаунта для `docker login` лучше не использовать.

1. Account Settings → Personal access tokens → Generate new token.
2. Для публикации нужен токен с правами **Read & Write**.
3. Для сервера с приватным репозиторием создайте отдельный токен
   **Read-only**.
4. Токен показывается один раз — сохраните его в менеджере секретов.

#### 3. Логин

```bash
docker login -u <dockerhub-username>
```

В качестве пароля вставьте токен. Проверка:

```bash
docker info | grep -i username
```

Вариант без интерактивного ввода:

```bash
read -rs DOCKERHUB_TOKEN
echo "$DOCKERHUB_TOKEN" | docker login -u <dockerhub-username> --password-stdin
unset DOCKERHUB_TOKEN
```

#### 4. Сборка образа

Из корня проекта:

```bash
cd /path/to/Utils

VERSION=1.0.0
IMAGE=<dockerhub-username>/utils-server-go

docker build -t "$IMAGE:$VERSION" -t "$IMAGE:latest" .
```

Проверка:

```bash
docker images | grep utils-server-go
```

`latest` — удобный тег для тестов, но на production лучше фиксировать
конкретную версию (`1.0.0`, `1.0.1`, `1.1.0`), чтобы деплой был
воспроизводимым.

#### 5. Локальная проверка перед публикацией

```bash
docker run --rm -d \
  --name utils-local-test \
  -p 8000:8000 \
  -v utils-data-local-test:/data \
  -e ACTIVE_USERS_LIMIT=10 \
  "$IMAGE:$VERSION"

sleep 2

curl -i -X POST http://127.0.0.1:8000/adduser \
  -H 'Content-Type: application/json' \
  -d '{"name":"local-test"}'

curl -i http://127.0.0.1:8000/active
```

Остановка и очистка:

```bash
docker rm -f utils-local-test
docker volume rm utils-data-local-test
```

#### 6. Отправка образа

```bash
docker push "$IMAGE:$VERSION"
docker push "$IMAGE:latest"
```

Проверка:

```bash
docker manifest inspect "$IMAGE:$VERSION"
```

Также откройте репозиторий на Docker Hub — теги должны появиться в списке.
Проверить скачивание:

```bash
docker rmi "$IMAGE:$VERSION" "$IMAGE:latest"
docker pull "$IMAGE:$VERSION"
```

#### 7. Если сервер другой архитектуры

Проверьте архитектуру сервера:

```bash
uname -m
```

- `x86_64` / `amd64` — подходит обычная сборка.
- `aarch64` / `arm64` — нужен образ под `linux/arm64`.

Вариант A — собрать нативно на сервере (самый надёжный для CGO):

```bash
git clone <repo-url>   # или скопируйте проект
cd Utils
docker build -t "$IMAGE:$VERSION" -t "$IMAGE:latest" .
```

Вариант B — multi-arch через `buildx`:

```bash
docker buildx create --use --name utils-builder
docker buildx inspect --bootstrap

docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t "$IMAGE:$VERSION" \
  -t "$IMAGE:latest" \
  --push .
```

Проект использует CGO (`mattn/go-sqlite3`). Кросс-сборка arm64 на amd64
идёт через QEMU, может быть медленной и иногда требует дополнительных
инструментов. Если она падает — собирайте нативно на сервере нужной
архитектуры или в CI на runner с нужной архитектурой.

#### 8. Автоматическая публикация через GitHub Actions (опционально)

Если проект будет в GitHub, можно публиковать образ по тегам `v*`.
Пример workflow:

```yaml
name: Publish image

on:
  push:
    tags:
      - 'v*'

jobs:
  publish:
    runs-on: ubuntu-latest
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@v4

      - uses: docker/setup-buildx-action@v3

      - uses: docker/login-action@v3
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}

      - uses: docker/metadata-action@v5
        id: meta
        with:
          images: ${{ secrets.DOCKERHUB_USERNAME }}/utils-server-go
          tags: |
            type=semver,pattern={{version}}
            type=raw,value=latest

      - uses: docker/build-push-action@v6
        with:
          context: .
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
```

Секреты `DOCKERHUB_USERNAME` и `DOCKERHUB_TOKEN` задаются в GitHub →
Settings → Secrets and variables → Actions. Для arm64 добавьте
`platforms: linux/amd64,linux/arm64`, но учитывайте стоимость и скорость
CGO-сборки через эмуляцию.

### Развёртывание на сервере из Docker Hub

#### 1. Установка Docker

Пример для Ubuntu/Debian. Для другого дистрибутива используйте официальную
инструкцию Docker.

```bash
sudo apt update
sudo apt install -y ca-certificates curl gnupg

sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
  | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
  | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt update
sudo apt install -y \
  docker-ce docker-ce-cli containerd.io \
  docker-buildx-plugin docker-compose-plugin

sudo systemctl enable --now docker
```

Добавьте пользователя в группу Docker и перезайдите по SSH:

```bash
sudo usermod -aG docker "$USER"
```

При необходимости примените группу в текущей сессии:

```bash
newgrp docker
```

Проверка:

```bash
docker version
docker compose version
docker run --rm hello-world
```

#### 2. Firewall и DNS

Для схемы с reverse proxy наружу нужны только SSH, HTTP и HTTPS:

```bash
sudo ufw allow OpenSSH
sudo ufw allow 'Nginx Full'
sudo ufw enable
sudo ufw status
```

Если сервер в облаке (AWS/GCP/Azure), дополнительно откройте 80/443 в
security group. **Не открывайте наружу порт 8000**, если используете
reverse proxy.

Создайте A-запись DNS:

```text
api.example.com -> <IP-адрес сервера>
```

Проверка:

```bash
dig +short api.example.com
```

#### 3. Каталог деплоя

```bash
sudo mkdir -p /opt/utils
sudo chown "$USER":"$USER" /opt/utils
cd /opt/utils
```

#### 4. Файл `.env`

Создайте `/opt/utils/.env`:

```dotenv
IMAGE_NAME=<dockerhub-username>/utils-server-go
IMAGE_TAG=1.0.0
API_PORT=8000
ACTIVE_USERS_LIMIT=10
```

Ограничьте права:

```bash
chmod 600 /opt/utils/.env
```

Если репозиторий приватный, залогиньтесь на сервере read-only токеном:

```bash
docker login -u <dockerhub-username>
```

#### 5. Production `docker-compose.yml`

Файл `docker-compose.yml` из репозитория содержит `build:`, поэтому при
запуске на сервере он может попытаться собрать образ локально. Для сервера
лучше отдельный файл **без сборки**, только с `image:`.

Создайте `/opt/utils/docker-compose.yml`:

```yaml
name: utils-server-go

services:
  api:
    image: "${IMAGE_NAME}:${IMAGE_TAG}"
    container_name: utils-server-go
    restart: unless-stopped
    ports:
      - "127.0.0.1:${API_PORT:-8000}:8000"
    environment:
      PORT: "8000"
      DB_PATH: /data/users.db
      ACTIVE_USERS_LIMIT: "${ACTIVE_USERS_LIMIT:-5}"
    volumes:
      - utils-data:/data
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"

volumes:
  utils-data:
    name: utils-data
```

Почему так:

- `127.0.0.1:${API_PORT}:8000` — контейнер доступен только локально,
  наружу его отдаёт Nginx/Caddy.
- `utils-data` — именованный volume, база не теряется при пересоздании
  контейнера.
- `restart: unless-stopped` — автозапуск после перезагрузки сервера.
- Логи ограничены по размеру, чтобы не забить диск.

Проверка конфигурации:

```bash
docker compose config
```

Быстрый вариант без отдельного файла: скопировать проект на сервер, выполнить
`docker compose pull` и `docker compose up -d --no-build`. Но production-файл
надёжнее: он не зависит от исходников и не пытается что-то собирать.

#### 6. Запуск

```bash
cd /opt/utils

docker compose pull
docker compose up -d
docker compose ps
docker compose logs -f --tail=100 api
```

Проверка API:

```bash
curl -i http://127.0.0.1:8000/active

curl -i -X POST http://127.0.0.1:8000/adduser \
  -H 'Content-Type: application/json' \
  -d '{"name":"prod-smoke","tags":["server"]}'

curl -i http://127.0.0.1:8000/user/1
```

Проверка сохранности данных после перезапуска:

```bash
docker compose restart api
sleep 2
curl -i http://127.0.0.1:8000/user/1
```

Где лежит база:

```bash
docker volume inspect utils-data
```

Обычно это каталог `/var/lib/docker/volumes/utils-data/_data`, в нём файлы
`users.db`, `users.db-wal`, `users.db-shm`. Не копируйте только `users.db`
при работающем контейнере: WAL-файлы нужны для консистентности.

### Reverse proxy и HTTPS

#### Nginx + Let's Encrypt

```bash
sudo apt install -y nginx certbot python3-certbot-nginx
```

Создайте `/etc/nginx/sites-available/utils`:

```nginx
server {
    listen 80;
    server_name api.example.com;

    client_max_body_size 1m;

    location / {
        proxy_pass http://127.0.0.1:8000;
        proxy_http_version 1.1;

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        proxy_read_timeout 30s;
        proxy_send_timeout 30s;
    }
}
```

Включите сайт и получите сертификат:

```bash
sudo ln -s /etc/nginx/sites-available/utils /etc/nginx/sites-enabled/utils
sudo nginx -t
sudo systemctl reload nginx

sudo certbot --nginx -d api.example.com
sudo certbot renew --dry-run
```

Проверка:

```bash
curl -i https://api.example.com/active
```

#### Caddy (альтернатива)

Caddy сам получает TLS-сертификаты. Пример `Caddyfile`:

```caddyfile
api.example.com {
    reverse_proxy 127.0.0.1:8000
}
```

Нужны открытые 80/443 и корректная DNS-запись.

#### Rate limiting для проверки пароля

У приложения нет аутентификации, а `POST /user/{id}/password/verify` можно
перебирать. Для публичного сервиса ограничьте частоту на прокси.

Пример для Nginx. В `http`-контексте (например,
`/etc/nginx/conf.d/limits.conf`):

```nginx
limit_req_zone $binary_remote_addr zone=pwd_verify:10m rate=5r/m;
```

В `server`:

```nginx
location ~ ^/user/[0-9]+/password/verify$ {
    limit_req zone=pwd_verify burst=10 nodelay;

    proxy_pass http://127.0.0.1:8000;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

Для Caddy нужен модуль rate limit (не входит в стандартную сборку) либо
ограничение на уровне облачного WAF.

### Обновление и откат

Сборка и публикация новой версии:

```bash
VERSION=1.1.0
IMAGE=<dockerhub-username>/utils-server-go

docker build -t "$IMAGE:$VERSION" -t "$IMAGE:latest" .
docker push "$IMAGE:$VERSION"
docker push "$IMAGE:latest"
```

Обновление на сервере:

```bash
cd /opt/utils

# Сначала сделайте бэкап, см. раздел ниже

sed -i 's/^IMAGE_TAG=.*/IMAGE_TAG=1.1.0/' .env
# или отредактируйте .env вручную

docker compose config
docker compose pull
docker compose up -d
docker compose ps
docker compose logs -f --tail=100 api
```

Проверка:

```bash
curl -i https://api.example.com/active
```

Очистка старых неиспользуемых образов:

```bash
docker image prune -f
```

Откат — верните предыдущий тег в `.env` и повторите:

```dotenv
IMAGE_TAG=1.0.0
```

```bash
cd /opt/utils
docker compose pull
docker compose up -d
docker compose ps
```

Схема БД мигрируется автоматически, но перед обновлением всё равно делайте
резервную копию.

### Резервное копирование SQLite

Самый простой и надёжный способ — кратко остановить контейнер и упаковать
volume.

Бэкап:

```bash
cd /opt/utils

BACKUP="users-$(date +%F-%H%M).tgz"

docker compose stop api
docker run --rm \
  -v utils-data:/data \
  -v "$PWD:/backup" \
  alpine tar czf "/backup/$BACKUP" -C /data .
docker compose start api

ls -lh "$BACKUP"
```

Восстановление:

```bash
cd /opt/utils

BACKUP=users-2026-11-15-2300.tgz

docker compose stop api
docker run --rm \
  -v utils-data:/data \
  -v "$PWD:/backup" \
  alpine sh -c "rm -rf /data/* && tar xzf '/backup/$BACKUP' -C /data"
docker compose start api
```

Онлайн-бэкап без остановки через SQLite backup API:

```bash
cd /opt/utils
BACKUP="users-$(date +%F-%H%M).db"

docker run --rm \
  -v utils-data:/data \
  -v "$PWD:/backup" \
  alpine sh -c "apk add --no-cache sqlite >/dev/null && sqlite3 /data/users.db \".backup '/backup/$BACKUP'\""

ls -lh "$BACKUP"
```

Храните копии вне сервера: S3, другой сервер, rsync и т.п. Не используйте
`docker compose down -v` в production — это удалит volume вместе с базой.

### Диагностика

| Симптом | Причина и решение |
| --- | --- |
| `denied: requested access to the resource is denied` | Неверный username/тег, нет логина или у токена нет прав |
| `exec format error` | Образ собран под другую архитектуру; проверьте `uname -m`, соберите multi-arch |
| `bind: address already in use` | Порт занят; проверьте `ss -ltnp \| grep 8000`, смените `API_PORT` |
| `no such image` | Образ не отправлен или неверный тег; выполните `docker compose pull` |
| `pull access denied` для приватного репозитория | На сервере не выполнен `docker login` |
| `permission denied` при записи в `/data` | Проблема прав volume, особенно если позже перейдёте на non-root |
| `docker compose: command not found` | Не установлен `docker-compose-plugin` |
| Сайт не открывается по HTTPS | DNS не указывает на сервер, закрыты 80/443 или Nginx не запущен |
| `502 Bad Gateway` | Контейнер не запущен или слушает не тот порт; смотрите `docker compose ps` и логи |
| Данные пропали | Был выполнен `docker compose down -v` или изменено имя volume |
| `latest` не обновился | Не выполнен `docker compose pull` или используется старый тег |
| Docker Hub ограничивает скачивания | Войдите в аккаунт Docker Hub на сервере, используйте кэш/зеркало |

Полезные команды:

```bash
cd /opt/utils
docker compose ps
docker compose logs -f --tail=200 api
docker stats utils-server-go
docker inspect utils-server-go
ss -ltnp | grep 8000
```

### Production checklist

- [ ] Образ опубликован с immutable-тегом и `latest`.
- [ ] На сервере зафиксирован конкретный `IMAGE_TAG`, а не `latest`.
- [ ] Порт 8000 не открыт наружу, используется `127.0.0.1`.
- [ ] Настроен HTTPS через Nginx или Caddy.
- [ ] Открыты только 22/80/443.
- [ ] `.env` имеет права `600`.
- [ ] Токен Docker Hub на сервере — read-only, если репозиторий приватный.
- [ ] Настроены бэкапы volume `utils-data`.
- [ ] Включён `restart: unless-stopped`.
- [ ] Ограничены размеры логов.
- [ ] Проверены обновление и откат.
- [ ] Для публичного API добавлен rate limit на `/password/verify`.
- [ ] Проверено восстановление из бэкапа.

## Проверка эндпоинтов

Для запущенного сервера:

```bash
# Создание пользователя
curl -X POST http://127.0.0.1:8000/adduser \
  -H "Content-Type: application/json" \
  -d '{"name":"alice","tags":["vip"]}'

# Получение пользователя
curl http://127.0.0.1:8000/user/1

# Установка пароля (ожидается 204)
curl -i -X PUT http://127.0.0.1:8000/user/1/password \
  -H "Content-Type: application/json" \
  -d '{"password":"s3cret"}'

# Проверка пароля (ожидается {"valid":true})
curl -X POST http://127.0.0.1:8000/user/1/password/verify \
  -H "Content-Type: application/json" \
  -d '{"password":"s3cret"}'

# Активные пользователи
curl http://127.0.0.1:8000/active

# Дубликат (ожидается 409)
curl -i -X POST http://127.0.0.1:8000/adduser \
  -H "Content-Type: application/json" \
  -d '{"name":"alice"}'

# Несуществующий пользователь (ожидается 404)
curl -i http://127.0.0.1:8000/user/99999

# Неподдерживаемый метод (ожидается 405)
curl -i -X DELETE http://127.0.0.1:8000/user/1
```

## Архитектура

| Модуль | Ответственность |
| --- | --- |
| `internal/config` | `Settings`, чтение `DB_PATH`, `ACTIVE_USERS_LIMIT`, `PORT` |
| `internal/models` | `User`, `ErrUserExists`, `ErrInvalidInput` |
| `internal/database` | `DB`: соединение, схема, миграция, WAL |
| `internal/repository` | `Repository`: `Add`, `GetByID`, `SetPasswordHash`, `GetPasswordHash` |
| `internal/services/password` | `PasswordService`: Argon2id-хеширование и проверка |
| `internal/services/presence` | `ActiveUsers`: потокобезопасный список активных |
| `internal/handlers` | HTTP: `Server`, маршруты, валидация, JSON-ответы, пароли, `/active` |
| `internal/utils` | `NormalizeName`, `NormalizeTags` |

Ключевые типы:

| Тип | Поля |
| --- | --- |
| `models.User` | `ID int`, `Name string`, `Tags []string` |
| `config.Settings` | `DBPath string`, `ActiveUsersLimit int`, `Port string` |

## Безопасность и ограничения

- SQL-инъекции исключены параметризованными запросами.
- Пароли не хранятся и не передаются в открытом виде: только Argon2id-хеш
  (`salt` 16 байт, память 64 МиБ, 1 итерация, 4 потока, ключ 32 байта).
- Пароль никогда не возвращается в `GET /user/<id>` и не логируется.
- Тело запроса ограничено 1 МиБ; принимается ровно один JSON-объект.
- HTTP-сервер использует таймауты чтения/записи/простоя и корректно
  завершается по `SIGINT`/`SIGTERM`.
- Аутентификации, авторизации и сессий нет — это учебный проект.
  Эндпоинт проверки пароля при публикации стоит защищать rate limiting
  на обратном прокси.
- Для production используйте сервер за обратным прокси (Nginx/Caddy) с HTTPS.
- Список активных пользователей хранится в памяти и не переживает перезапуск.

## Что можно улучшить дальше

Проект завершён как учебный, но для production есть несколько направлений
развития. Ниже — приоритизированный список.

| Приоритет | Улучшение | Зачем |
| --- | --- | --- |
| Высокий | Multi-stage Dockerfile, запуск от non-root, минимальный runtime-образ | Меньше размер образа и поверхность атаки |
| Высокий | Аутентификация и авторизация (API-ключи, JWT или сессии) | Сейчас любой может создавать пользователей и менять пароли |
| Высокий | Rate limiting и защита от перебора для `/password/verify` | Эндпоинт проверки пароля сейчас можно брутфорсить |
| Высокий | HTTPS, security headers, CORS, ограничение доступа по IP/VPN | Безопасная публикация в интернет |
| Высокий | Эндпоинты `/healthz` и `/readyz` | Healthcheck для Docker, Kubernetes и мониторинга |
| Высокий | Структурированное логирование (`log/slog`), request ID | Удобная диагностика без утечки секретов |
| Средний | Метрики Prometheus и трассировка OpenTelemetry | Наблюдаемость: latency, ошибки, нагрузка |
| Средний | Версионированные миграции БД (goose/migrate) | Предсказуемые изменения схемы вместо `EnsureSchema` |
| Средний | Список пользователей с пагинацией, обновление и удаление | Сейчас API умеет только создавать и читать по `id` |
| Средний | Лимиты на длину имени, количество и длину тегов | Защита от раздувания БД и ответов |
| Средний | Политика паролей, pepper, настраиваемые параметры Argon2id | Гибкая безопасность под разные окружения |
| Средний | Rehash пароля при изменении параметров Argon2id | Плавное усиление хеширования без сброса паролей |
| Средний | Вынести active users в Redis с TTL | Корректная работа при нескольких инстансах |
| Средний | Переход на PostgreSQL при горизонтальном масштабировании | SQLite ограничивает конкурентную запись и репликацию |
| Средний | Интеграционные и e2e-тесты, fuzz-тесты, порог покрытия в CI | Раннее обнаружение регрессий |
| Средний | Контрактные тесты OpenAPI | Гарантия, что код и спецификация не расходятся |
| Средний | Публикация образа в CI, multi-arch, SBOM, Trivy, cosign, Dependabot | Безопасная и воспроизводимая поставка |
| Низкий | Валидация `PORT`, настраиваемые таймауты и лимиты через флаги | Гибкая конфигурация без правки кода |
| Низкий | Версионирование API (`/v1`), генерация клиентов из OpenAPI | Совместимость и удобство интеграций |
| Низкий | Kubernetes manifests или Helm chart, probes, resource limits | Оркестрация и отказоустойчивость |
| Низкий | Автоматизация бэкапов, retention и учебное восстановление | Защита данных и проверка процедур |
| Низкий | `golangci-lint`, `govulncheck`, `/healthz` в Makefile | Единый локальный и CI-контроль качества |
| Низкий | Docker Compose для разработки и hot reload (`air`) | Быстрый локальный цикл разработки |
| Низкий | CHANGELOG, ADR, перевод README на английский | Поддержка и передача проекта |
| Низкий | Аудит действий и эндпоинт удаления данных пользователя | Требования приватности и разбор инцидентов |

Ближайшие практичные шаги:

1. Сделать multi-stage Dockerfile с non-root пользователем и healthcheck.
2. Добавить `/healthz`, structured logging и rate limiting на прокси.
3. Ввести аутентификацию и версионированные миграции БД.
4. Настроить автоматическую публикацию образа и сканирование уязвимостей.
5. Добавить интеграционные тесты и контрактные проверки OpenAPI.

## Лицензия

[MIT](LICENSE).
