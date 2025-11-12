# FX Rates – Live Currency Data

FX Rates is a small, easy-to-use app that gets currency rates from ExchangeRate-API, saves them in Postgres, serves them through Go REST API, and displays in a nice UI. Hope you like it 😎❤️

---

## What You Get
- Friendly UI to inspect latest rates, request updates, and track statuses.
- Ready-to-use Swagger UI for manual testing.

---

## One-Minute Docker Compose Run 🚀
1. Add your `EXCHANGE_RATE_API_KEY` to `.env` (grab it at https://www.exchangerate-api.com/).
2. Run `docker compose up --build`.
3. Open <http://localhost:5173> (UI) or <http://localhost:8080/swagger/index.html> (Swagger).

Docker handles Postgres, migrations, backend, and frontend for you.

---

## 1. Manual Backend Run 🛠️
| Requirement | Notes |
| --- | --- |
| Go | 1.24+ |
| Postgres | 14+ (docker snippet below) |
| ExchangeRate-API key | grab one at <https://www.exchangerate-api.com/> |

1. Start Postgres:
   ```powershell
   docker run -d --name fxrates -e POSTGRES_DB=fxrates -e POSTGRES_USER=user -e POSTGRES_PASSWORD=pass -p 5432:5432 -v pgdata:/var/lib/postgresql/data postgres:17
   ```
2. Apply migrations:
   ```powershell
   go run github.com/pressly/goose/v3/cmd/goose@latest ^
     -dir internal/platform/db/migrations ^
     postgres "postgres://user:pass@localhost:5432/fxrates?sslmode=disable" up
   ```
3. Create `.env` in the project root.
4. Run `go run ./cmd`.

---

## 2. Manual UI Run 🛠️
```powershell
cd web/ui
npm install
npm run dev
```
**UI example:**

![UI Preview](./web/ui/public/ui.png)

---

## Config ⚙️
Defaults live in `config.yaml`. Environment vars override them. `.env` is auto-loaded unless `PROFILE` is set.

| Variable | Description | Example |
| --- | --- | --- |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASS`, `DB_NAME` | Postgres connection | `localhost`, `5432`, … |
| `EXCHANGE_RATE_API_BASE_URL` | ExchangeRate-API URL | `https://v6.exchangerate-api.com/v6` |
| `EXCHANGE_RATE_API_KEY` | Required API key | _none_ |
| `HTTP_CLIENT_TIMEOUT_SECONDS` | HTTP timeout | `10` |
| `UPDATE_RATES_JOB_DURATION_SEC` | Scheduler interval | `30` |
| `RATE_UPDATES_CACHE_MAX_ITEMS` | Cache size | `512` |
| `LOG_LEVEL` | `debug`, `info`, `warn`, … | `info` |
| `PROFILE` | Skip `.env` when set | _(empty locally)_ |

---

## API 📚
Swagger UI: <http://localhost:8080/swagger/index.html>

| Method | Path | Purpose                             |
| --- | --- |-------------------------------------|
| `GET` | `/api/v1/rates/supported-currencies` | List of supported currencies        |
| `GET` | `/api/v1/rates/{base}/{quote}` | Latest rate for a pair              |
| `POST` | `/api/v1/rates/updates` | Request a rate update (`update_id`) |
| `GET` | `/api/v1/rates/updates/{id}` | Look up a rate by `update_id`       |

---

## Project Map 🗺️
```
.
├── cmd/                  # Entrypoint
├── internal/
│   ├── app/              # Component wiring
│   ├── config/           # Config definitions + loading
│   ├── api/              # HTTP router
│   ├── adapters/
│   │   ├── postgres/     # DB logic
│   │   ├── cache/        # Cache helpers
│   │   └── httpclient/   # External API client
│   ├── rate/             # Business logic, scheduler, handlers
│   ├── platform/         # DB pool, migrations, HTTP server
│   └── domain/           # Domain types
├── web/ui/               # Web UI
└── docs/                 # Generated Swagger files
```

---

## Testing ✅
- Backend: `go test ./...` (uses `testcontainers`, so Docker must be running).

---

## Troubleshooting 💡
- **UI in docker, backend outside docker** – Vite proxy expects both in the same network. Run both locally or both in docker.
- **Missing tables** – make sure migrations ran.
