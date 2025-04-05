folder=$(shell basename $(CURDIR))
include ./secret/.env
MIGRATION_PATH=./internal/db/schema

clear-dist:
	rm -rf frontend/build
	rm backend/bin/main

clear-db:
	docker volume rm -f db_data
	docker volume rm -f pgadmin-data

clear-front:
	docker volume rm -f app_front_node_modules

clear: clear-dist clear-db clear-front

build:
	docker compose -f docker-compose.yml --env-file ./secret/.env build

up:
	docker compose -f docker-compose.yml --env-file ./secret/.env up -d

down:
	docker compose -f docker-compose.yml --env-file ./secret/.env down

migrate-create:
	docker compose -f docker-compose.yml --env-file ./secret/.env run --rm backend migrate create -seq -ext sql -dir $(MIGRATION_PATH) $(filter-out $@,$(MAKECMDGOALS))

migrate-up:
	docker compose -f docker-compose.yml --env-file ./secret/.env run --rm backend migrate -path $(MIGRATION_PATH) -database $(DB_ADDRESS) up

migrate-down:
	docker compose -f docker-compose.yml --env-file ./secret/.env run --rm backend migrate -path $(MIGRATION_PATH) -database $(DB_ADDRESS) down $(filter-out $@,$(MAKECMDGOALS))

generate:
	docker compose -f docker-compose.yml --env-file ./secret/.env run --rm backend sqlc generate
