REPO_ROOT := ../..
BACKEND_ROOT := ..
COMPOSE_FILE := $(REPO_ROOT)/docker-compose.yml
SERVER_URL := http://127.0.0.1:8080/api/v1/auth/login
TMP_DIR := $(BACKEND_ROOT)/.tmp
LOG_FILE := $(TMP_DIR)/game-server.log
PID_FILE := $(TMP_DIR)/game-server.pid
GO_CACHE := $(TMP_DIR)/go-build-cache

.PHONY: ensure docker-up server-up server-stop restart status logs

ensure: docker-up server-up

docker-up:
	@set -e; \
	if docker compose -f $(COMPOSE_FILE) ps --services --filter status=running | grep -qx postgres \
		&& docker compose -f $(COMPOSE_FILE) ps --services --filter status=running | grep -qx redis; then \
		echo "docker compose services already running"; \
	else \
		echo "starting docker compose services..."; \
		docker compose -f $(COMPOSE_FILE) up -d; \
	fi

server-up:
	@set -e; \
	if curl -s -o /dev/null -w "%{http_code}" $(SERVER_URL) | grep -Eq '^(200|405)$$'; then \
		echo "game server already running on :8080"; \
	else \
		echo "starting game server..."; \
		mkdir -p $(TMP_DIR) $(GO_CACHE); \
		nohup sh -c 'cd $(abspath $(BACKEND_ROOT)) && env GOCACHE=$(abspath $(GO_CACHE)) go run ./server/cmd/game-server' > $(abspath $(LOG_FILE)) 2>&1 & echo $$! > $(abspath $(PID_FILE)); \
		started=0; \
		for i in 1 2 3 4 5 6 7 8 9 10; do \
			if curl -s -o /dev/null -w "%{http_code}" $(SERVER_URL) | grep -Eq '^(200|405)$$'; then \
				started=1; \
				break; \
			fi; \
			sleep 1; \
		done; \
		if [ $$started -eq 1 ]; then \
			echo "game server started"; \
		else \
			echo "game server failed to start, see $(LOG_FILE)"; \
			exit 1; \
		fi; \
	fi

server-stop:
	@set -e; \
	if [ -f $(PID_FILE) ]; then \
		pid=$$(cat $(PID_FILE)); \
		if kill -0 $$pid 2>/dev/null; then \
			echo "stopping game server pid=$$pid"; \
			kill $$pid; \
			sleep 1; \
		fi; \
		rm -f $(PID_FILE); \
	elif lsof -tiTCP:8080 -sTCP:LISTEN >/dev/null 2>&1; then \
		pid=$$(lsof -tiTCP:8080 -sTCP:LISTEN | head -n 1); \
		echo "stopping game server pid=$$pid"; \
		kill $$pid; \
		sleep 1; \
	else \
		echo "game server is not running"; \
	fi

restart: server-stop server-up

status:
	@set -e; \
	echo "[docker compose]"; \
	docker compose -f $(COMPOSE_FILE) ps; \
	echo ""; \
	echo "[game server]"; \
	if curl -s -o /dev/null -w "%{http_code}" $(SERVER_URL) | grep -Eq '^(200|405)$$'; then \
		if lsof -tiTCP:8080 -sTCP:LISTEN >/dev/null 2>&1; then \
			pid=$$(lsof -tiTCP:8080 -sTCP:LISTEN | head -n 1); \
			echo "running on :8080 (pid=$$pid)"; \
		else \
			echo "running on :8080"; \
		fi; \
	else \
		echo "not running"; \
	fi

logs:
	@mkdir -p $(TMP_DIR); \
	if [ -f $(LOG_FILE) ]; then \
		tail -n 50 $(LOG_FILE); \
	else \
		echo "no log file yet: $(LOG_FILE)"; \
	fi

#   - make -C backend/server ensure
#   - make -C backend/server status
#   - make -C backend/server logs
