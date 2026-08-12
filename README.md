# Trading Dashboard (AE Telelink System Ltd.)

A trading dashboard for Indian stocks and mutual funds: search assets, view historical price/NAV charts, with email/password authentication.

## Repository layout

```
trading-dashboard/
├── backend/    # Go REST API (this directory)
|   └── data/db/trading.sql   # Postgres schema, loaded by `make setup-db`
└── frontend/
    └── trading-app/   # React (Create React App) frontend

```

The backend's `docker-compose.yaml` and `make setup-db` reference `../data/db/trading.sql`, so keep this repo layout intact (don't clone `backend/` on its own).

## Prerequisites

- Go 1.26+ (see `go.mod`)
- Docker and docker-compose (for the local Postgres database)

## Backend application

### Go to application path
```
$ cd backend
```

### Configuration

The backend does not use a `.env` file — the database connection string and the external API endpoints (Yahoo Finance for stocks, mfapi.in for mutual funds) are hardcoded constants in `config/constants.go`. By default it expects Postgres reachable at `postgres://postgres:example@localhost:5432/trading_dashboard` (matching `docker-compose.yaml`'s credentials below).

### Start database service
```
$ make up
```
Starts Postgres on `localhost:5432` and Adminer (a DB admin UI) on `http://localhost:8090`. Credentials: user `postgres`, password `example`, database `trading_dashboard`.

### Setup/Generate the db schema if application is being started first time
```
$ make setup-db
```

### Run the backend application
```
$ go mod download
$ go run ./cmd/app-api/
```
The API listens on `http://localhost:8080`.

### Run tests
```
$ make test
```
Runs the full test suite with `-v` and prints a total coverage summary at the end. Use `make test-cover` for a quicker per-package coverage summary without verbose test output.

### Other useful commands
```
$ make ps      # show status of the docker-compose services
$ make psql    # open a psql shell against the running db
$ make down    # stop the database service
```

## API Endpoints

All endpoints are prefixed with `/api/v1`. `/search` and `/market-data` require a JWT obtained from `/auth/login`, sent as `Authorization: Bearer <token>` — calling them without one returns `401 Unauthorized`.

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/auth/register` | – | Create an account (`{email, password}`) |
| POST | `/auth/login` | – | Log in, returns a JWT (`{email, password}` → `{token}`) |
| GET | `/search?q=` | Bearer | Search stocks and mutual funds by name |
| GET | `/market-data?id=&type=&days=` | Bearer | Historical price/NAV points for an asset (`type` is `STOCK` or `MUTUAL_FUND`) |

## Ports

| Service | Port |
|---|---|
| Backend API | 8080 |
| Adminer (DB admin UI) | 8090 |
| Postgres | 5432 |

## Frontend

See `frontend/trading-app/README.md` for running the React frontend.
