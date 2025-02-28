# Nama project
PROJECT_NAME = dubaiDeals

# Environment file
ENV_FILE = .env

# Docker Compose file
DOCKER_COMPOSE_FILE = docker-compose.yml

# Jalankan semua container
.PHONY: up
up:
	docker-compose -f $(DOCKER_COMPOSE_FILE) --env-file $(ENV_FILE) up -d --build

# Hentikan semua container
.PHONY: down
down:
	docker-compose -f $(DOCKER_COMPOSE_FILE) down

# Hapus semua container dan volume
.PHONY: clean
clean:
	docker-compose -f $(DOCKER_COMPOSE_FILE) down -v

# Restart service tertentu
.PHONY: restart
restart:
	docker-compose -f $(DOCKER_COMPOSE_FILE) restart

# Lihat logs semua service
.PHONY: logs
logs:
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

# Lihat logs service tertentu
.PHONY: logs-api-gateway
logs-api-gateway:
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f api-gateway

.PHONY: logs-user-service
logs-user-service:
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f user-service

# Tampilkan status container
.PHONY: ps
ps:
	docker-compose -f $(DOCKER_COMPOSE_FILE) ps

# Hapus semua image yang tidak digunakan
.PHONY: prune
prune:
	docker system prune -af

# Jalankan shell dalam container api-gateway
.PHONY: shell-api-gateway
shell-api-gateway:
	docker exec -it api_gateway_container sh

# Jalankan shell dalam container user-service
.PHONY: shell-user-service
shell-user-service:
	docker exec -it user_service_container sh
