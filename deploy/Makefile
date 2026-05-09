.PHONY: up down restart logs ps pull config

COMPOSE ?= docker compose
COMPOSE_FILE ?= docker-compose.yml

up:
	@$(COMPOSE) -f $(COMPOSE_FILE) up -d

down:
	@$(COMPOSE) -f $(COMPOSE_FILE) down

restart:
	@$(COMPOSE) -f $(COMPOSE_FILE) restart

logs:
	@$(COMPOSE) -f $(COMPOSE_FILE) logs -f sub2api

ps:
	@$(COMPOSE) -f $(COMPOSE_FILE) ps

pull:
	@$(COMPOSE) -f $(COMPOSE_FILE) pull

config:
	@$(COMPOSE) -f $(COMPOSE_FILE) config
