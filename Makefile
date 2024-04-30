build:
	if [ ! -d "runtime" ]; then \
		mkdir runtime/configs -p; \
		chmod 0777 -R runtime; \
	fi
	chmod +x bin/get-binaries.sh; \
	if [ ! -f "bin/centrifugo" ]; then \
		cd bin; \
		./get-binaries.sh; \
		cd ../; \
	fi
	if [ ! -f "rr" ]; then \
		vendor/bin/rr get;\
	fi
	if [ ! -d "vendor" ]; then \
	    composer i --ignore-platform-reqs; \
	fi
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
