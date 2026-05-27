SHELL := /bin/bash

MONGO_CONTAINER ?= mongodb
MONGO_PORT ?= 27017
MONGO_DATA_DIR ?= $(HOME)/mongo-data
MONGO_IMAGE ?= mongo:latest
MONGO_ROOT_USERNAME ?= admin
MONGO_ROOT_PASSWORD ?= password

.PHONY: mongo mongo-stop mongo-clean mongo-shell

SHELL := /bin/bash

MONGO_CONTAINER ?= mongodb
MONGO_PORT ?= 27017
MONGO_DATA_DIR ?= $(HOME)/mongo-data
MONGO_IMAGE ?= mongo:latest
MONGO_ROOT_USERNAME ?= admin
MONGO_ROOT_PASSWORD ?= password
MONGO_NETWORK ?= bridge

.PHONY: mongo mongo-stop mongo-clean mongo-shell mongo-wait mongo-status mongo-logs mongo-reset

mongo:
	@mkdir -p $(MONGO_DATA_DIR)
	@if docker ps -a --format '{{.Names}}' | grep -q '^$(MONGO_CONTAINER)$$'; then \
		echo "Container '$(MONGO_CONTAINER)' already exists; starting it..."; \
		docker start $(MONGO_CONTAINER); \
	else \
		echo "Creating MongoDB container '$(MONGO_CONTAINER)'..."; \
		docker run -d \
			--name $(MONGO_CONTAINER) \
			--network $(MONGO_NETWORK) \
			-p 127.0.0.1:$(MONGO_PORT):27017 \
			-v $(MONGO_DATA_DIR):/data/db \
			-e MONGO_INITDB_ROOT_USERNAME=$(MONGO_ROOT_USERNAME) \
			-e MONGO_INITDB_ROOT_PASSWORD=$(MONGO_ROOT_PASSWORD) \
			$(MONGO_IMAGE); \
	fi
	@echo "Waiting for MongoDB to be ready..."
	@$(MAKE) mongo-wait

mongo-wait:
	@echo -n "Waiting for MongoDB"
	@for i in $$(seq 1 30); do \
		if docker exec $(MONGO_CONTAINER) mongosh --quiet --eval "1+1" >/dev/null 2>&1; then \
			echo -e "\n✅ MongoDB is ready!"; \
			echo "Connect with: mongodb://$(MONGO_ROOT_USERNAME):$(MONGO_ROOT_PASSWORD)@127.0.0.1:$(MONGO_PORT)/admin"; \
			exit 0; \
		fi; \
		echo -n "."; \
		sleep 1; \
	done; \
	echo -e "\n❌ MongoDB failed to start within 30 seconds"; \
	docker logs $(MONGO_CONTAINER) --tail 10; \
	exit 1

mongo-stop:
	@if docker ps -a --format '{{.Names}}' | grep -q '^$(MONGO_CONTAINER)$$'; then \
		echo "Stopping MongoDB container '$(MONGO_CONTAINER)'..."; \
		docker stop $(MONGO_CONTAINER); \
	else \
		echo "No container named '$(MONGO_CONTAINER)' found."; \
	fi

mongo-clean: mongo-stop
	@if docker ps -a --format '{{.Names}}' | grep -q '^$(MONGO_CONTAINER)$$'; then \
		echo "Removing MongoDB container '$(MONGO_CONTAINER)'..."; \
		docker rm $(MONGO_CONTAINER); \
	fi

mongo-shell:
	@if ! docker ps --format '{{.Names}}' | grep -q '^$(MONGO_CONTAINER)$$'; then \
		echo "MongoDB container '$(MONGO_CONTAINER)' is not running; starting it first..."; \
		$(MAKE) mongo; \
	fi
	docker exec -it $(MONGO_CONTAINER) mongosh \
		--username $(MONGO_ROOT_USERNAME) \
		--password $(MONGO_ROOT_PASSWORD) \
		--authenticationDatabase admin
