build:
	docker compose up --no-start;

start:
	docker compose up --remove-orphans -d;

up: build start

stop:
	docker compose stop;

down:
	docker compose down;

restart:
	docker compose restart;

list:
	docker compose ps;

log-tail:
	docker compose logs --tail=50 -f;

pull-latest:
	docker compose pull;

# =========================

bash:
	docker compose exec buggregator-server /bin/sh;

reset-server:
	docker compose exec buggregator-server ./rr reset;

reset: reset-server
