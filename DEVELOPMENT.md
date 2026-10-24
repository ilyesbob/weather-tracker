# Разработка Weather Tracker (Backend)

Документ для команды: как поднять бэкенд локально, прогнать тесты и где смотреть API.
Написан так, чтобы им могли пользоваться как Go-разработчики, так и фронтендеры
(React SPA), которым нужно просто запустить бэкенд для интеграции.

---

## 1. О проекте

**Weather Tracker** — сервис для планирования поездок и активностей с учётом погоды.
Бэкенд отдаёт JSON по REST и потребляется React SPA.

Реализованные пользовательские истории:

- **US1 — ранжирование городов.** Пользователь задаёт приоритеты (температура,
  осадки, ветер) для списка до 5 городов; система считает балл соответствия,
  сортирует по убыванию, подсвечивает лидера и помечает устаревшие данные.
- **US2 — виндсерфинг.** Пользователь задаёт уровень катания и диапазон скорости
  ветра; система анализирует прогноз по точкам с учётом направления ветра
  относительно берега (оншор/оффшор/кросс-шор) и рекомендует лучшее место и время
  с обоснованием.

### Стек

| Слой | Технология |
|---|---|
| Язык | Go |
| HTTP-роутер | `github.com/go-chi/chi/v5` |
| Документация API | `swaggo/swag` + `swaggo/http-swagger/v2` |
| Аутентификация | JWT (`golang-jwt/jwt/v5`) + bcrypt (`golang.org/x/crypto`) |
| Конфигурация | переменные окружения + `.env` (`github.com/joho/godotenv`) |
| Логирование | стандартный `log/slog` |
| Внешние данные | Open-Meteo Weather API |
| Хранилище (MVP) | in-memory адаптеры (PostgreSQL/Redis — в планах) |

Архитектура слоистая: `handler → service → repository`. Зависимости передаются
через интерфейсы, поэтому тесты не требуют реальной БД или сети.

---

## 2. Системные требования

- **Go 1.26 или новее** (версия зафиксирована в `go.mod`). Проверить:
  ```bash
  go version
  ```
- **Утилита `swag`** — только если нужно перегенерировать Swagger:
  ```bash
  go install github.com/swaggo/swag/cmd/swag@latest
  ```
  Убедитесь, что `$HOME/go/bin` (или `$(go env GOPATH)/bin`) добавлен в `PATH`.
- **Make** (опционально) — для коротких команд из `Makefile`.
- Исходящий доступ в интернет к `api.open-meteo.com` и `geocoding-api.open-meteo.com`
  (для реальных погодных данных).

---

## 3. Быстрый старт (TL;DR)

Для тех, кто уже знаком с Go:

```bash
git clone <URL_РЕПОЗИТОРИЯ> && cd weather-tracker-main
go mod download
make run          # или: go run ./cmd/api
# Swagger UI: http://localhost:8080/docs/index.html
```

---

## 4. Пошаговая инструкция

### Шаг 1. Клонировать репозиторий

```bash
git clone <URL_РЕПОЗИТОРИЯ>
cd weather-tracker-main
```

### Шаг 2. Настроить окружение (опционально)

Для локального запуска **ничего настраивать не нужно** — все переменные имеют
безопасные значения по умолчанию (см. таблицу в разделе 7). Если нужно
переопределить значения, создайте в корне файл `.env` по образцу:

```dotenv
APP_ADDR=:8080
JWT_SECRET=dev-secret-change-me
JWT_TTL=24h
WEATHER_BASE_URL=https://api.open-meteo.com
GEOCODING_BASE_URL=https://geocoding-api.open-meteo.com
UPSTREAM_TIMEOUT=5s
WEATHER_CACHE_TTL=5m
STALE_THRESHOLD=5m
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
```

Сервер автоматически читает `.env` при запуске (отсутствие файла — не ошибка).
Реальный `.env` добавлен в `.gitignore` — **не коммитьте его**.

Переменные можно как задавать в `.env`, так и экспортировать в оболочке —
переменные окружения имеют приоритет над `.env`.

### Шаг 3. Установить зависимости

```bash
go mod download
```

### Шаг 4. Сгенерировать Swagger (при необходимости)

Готовая спецификация уже лежит в `docs/`. Перегенерируйте её после правки
аннотаций в хендлерах:

```bash
swag init -g cmd/api/main.go -o docs --parseInternal
# или короче:
make swagger
```

### Шаг 5. Запустить сервер

```bash
go run ./cmd/api
# или:
make run
```

По умолчанию сервер слушает `http://localhost:8080`. Порт меняется через
`APP_ADDR` в `.env`.

### Шаг 6. Запустить тесты

```bash
go test ./...                      # быстрый прогон
go test -v -cover ./...            # подробный лог + покрытие
go test -race ./...                # проверка гонок
# или:
make test
```

Проверка стиля и статический анализ:

```bash
go vet ./...
gofmt -l .        # должен ничего не вывести
```

---

## 5. API и документация

После запуска сервера:

- **Swagger UI (интерактивное тестирование):** <http://localhost:8080/docs/index.html>
- **OpenAPI JSON:** <http://localhost:8080/docs/doc.json>
- **Health-check:** <http://localhost:8080/healthz>

### Как фронтенду работать с API

1. Зарегистрируйтесь или войдите, чтобы получить JWT:
   - `POST /api/v1/auth/register` или `POST /api/v1/auth/login`
     с телом `{"email": "...", "password": "..."}`.
   - В ответе придёт `{"token": "...", "user": {...}}` — сохраните токен.
2. Передавайте токен в защищённые запросы в заголовке:
   ```
   Authorization: Bearer <token>
   ```
3. В Swagger UI нажмите **Authorize** и введите `Bearer <token>`.
4. Формат ошибок единообразен на всех эндпоинтах:
   ```json
   { "error": "текст ошибки" }
   ```
   с корректным HTTP-статусом (400 / 401 / 404 / 409 / 422 / 500).

### CORS

По умолчанию разрешены dev-серверы Vite (`http://localhost:5173`) и CRA
(`http://localhost:3000`). Если фронтенд работает на другом origin — добавьте его
в `ALLOWED_ORIGINS` (через запятую).

---

## 6. Карта проекта

```
weather-tracker-main/
├── cmd/
│   └── api/
│       └── main.go            # точка входа: сборка зависимостей, старт сервера
├── internal/
│   ├── config/                # чтение конфигурации из ENV/.env
│   ├── domain/                # бизнес-сущности (User, Location, Weather, Ranking, Windsurf)
│   ├── handler/               # HTTP-слой (роутер, хендлеры, Swagger-аннотации)
│   ├── middleware/            # CORS, JWT-аутентификация, логирование
│   ├── repository/            # контракты (интерфейсы) для хранилищ и провайдеров
│   │   ├── memory/            # in-memory реализации (MVP)
│   │   └── weather/           # клиент Open-Meteo + TTL-кэш
│   └── service/               # бизнес-логика: auth, locations, ranking (US1), windsurf (US2)
├── docs/                      # сгенерированный OpenAPI (docs.go, swagger.json/yaml)
├── Makefile                   # быстрые команды (run, build, test, swagger, ...)
├── go.mod / go.sum            # зависимости
└── README.md                  # продуктовое описание и архитектура
```

### Где писать код

- **Новый HTTP-эндпоинт** — хендлер в `internal/handler/`, регистрация в
  `internal/handler/router.go`. Не забудьте Swagger-аннотации в комментарии.
- **Новая бизнес-логика** — сервис в `internal/service/`.
- **Работа с данными/внешним API** — интерфейс в `internal/repository/repository.go`,
  реализация в подпакете (`memory/`, `weather/`).
- **Новые сущности/DTO** — `internal/domain/`.
- **Переменная окружения** — добавьте чтение с дефолтом в `internal/config/config.go`
  и опишите в таблице переменных (раздел 7 этого документа).

### Полезные команды (`Makefile`)

| Команда | Что делает |
|---|---|
| `make run` | запуск API |
| `make build` | сборка бинарника в `bin/` |
| `make test` | юнит-тесты |
| `make vet` | `go vet ./...` |
| `make fmt` | `gofmt -w .` |
| `make swagger` | перегенерация Swagger |
| `make tidy` | `go mod tidy` |

---

## 7. Конфигурация (переменные окружения)

| Переменная | Дефолт | Назначение |
|---|---|---|
| `APP_ADDR` | `:8080` | Адрес/порт HTTP-сервера |
| `JWT_SECRET` | `dev-secret-change-me` | Секрет подписи токенов (**заменить в prod**) |
| `JWT_TTL` | `24h` | Время жизни access-токена |
| `WEATHER_BASE_URL` | `https://api.open-meteo.com` | URL прогноза Open-Meteo |
| `GEOCODING_BASE_URL` | `https://geocoding-api.open-meteo.com` | URL геокодинга Open-Meteo |
| `UPSTREAM_TIMEOUT` | `5s` | Таймаут запросов к Open-Meteo |
| `WEATHER_CACHE_TTL` | `5m` | TTL кэша погодных снапшотов |
| `STALE_THRESHOLD` | `5m` | Порог устаревания данных (`isStale`) |
| `ALLOWED_ORIGINS` | `http://localhost:3000,http://localhost:5173` | Разрешённые CORS-origin |

---

## 8. Частые проблемы

- **`swag: command not found`** — установите `go install github.com/swaggo/swag/cmd/swag@latest`
  и добавьте `$(go env GOPATH)/bin` в `PATH`.
- **Порт занят** — измените `APP_ADDR` (например, `:8081`) в `.env`.
- **Фронтенд получает CORS-ошибку** — добавьте origin фронтенда в `ALLOWED_ORIGINS`
  и перезапустите сервер.
- **Погода «недоступна»** — проверьте доступ в интернет к Open-Meteo; сервис
  корректно деградирует и отдаёт последние известные данные с флагом `isStale`.
- **401 на защищённых эндпоинтах** — проверьте, что передаёте
  `Authorization: Bearer <token>` и что токен не истёк.
