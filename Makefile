up:
	docker-compose up -d

ps:
	docker-compose ps

down:
	docker-compose down

psql:
	docker-compose exec db psql -U postgres -d trading_dashboard -h localhost -p 5432

setup-db:
	docker-compose exec db psql postgresql://postgres:example@localhost:5432 -f /docker-entrypoint-initdb.d/trading.sql

test:
	go test ./... -v -coverprofile=coverage.out
	@echo "\nTotal coverage:"
	@go tool cover -func=coverage.out | tail -1

test-cover:
	go test ./... -cover

