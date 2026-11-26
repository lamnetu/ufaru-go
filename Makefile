.PHONY: dev up down logs

dev:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d --build

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f parser