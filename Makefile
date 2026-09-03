SHELL := /bin/bash

MONGO_CONTAINER ?= mongodb
MONGO_PORT ?= 27017
MONGO_DATA_DIR ?= $(HOME)/mongo-data
MONGO_IMAGE ?= mongo:latest
MONGO_ROOT_USERNAME ?= admin
MONGO_ROOT_PASSWORD ?= password
MONGO_NETWORK ?= bridge
MONGO_REPLICA_SET ?= rs0
MONGO_CONFIG_DIR ?= $(HOME)/mongo-config
MONGO_KEYFILE ?= $(MONGO_CONFIG_DIR)/mongodb-keyfile

.PHONY: mongo mongo-init-replica-set mongo-stop mongo-clean mongo-recreate mongo-shell mongo-wait mongo-status mongo-logs

mongo:
	@mkdir -p $(MONGO_DATA_DIR)
	@mkdir -p $(MONGO_CONFIG_DIR)
	@if [ -f $(MONGO_DATA_DIR)/mongodb-keyfile ]; then \
		echo "Removing legacy keyfile from MongoDB data directory..."; \
		rm -f $(MONGO_DATA_DIR)/mongodb-keyfile; \
	fi
	@if [ ! -f $(MONGO_KEYFILE) ]; then \
		echo "Creating MongoDB replica set keyfile..."; \
		openssl rand -base64 756 > $(MONGO_KEYFILE); \
		chmod 400 $(MONGO_KEYFILE); \
	fi
	@if docker ps -a --format '{{.Names}}' | grep -q '^$(MONGO_CONTAINER)$$'; then \
		echo "Container '$(MONGO_CONTAINER)' already exists; starting it..."; \
		docker start $(MONGO_CONTAINER); \
	else \
		echo "Creating MongoDB replica set container '$(MONGO_CONTAINER)'..."; \
		docker run -d \
			--name $(MONGO_CONTAINER) \
			--network $(MONGO_NETWORK) \
			-p 127.0.0.1:$(MONGO_PORT):27017 \
			-v $(MONGO_DATA_DIR):/data/db \
			-v $(MONGO_KEYFILE):/etc/mongo-keyfile:ro \
			-e MONGO_INITDB_ROOT_USERNAME=$(MONGO_ROOT_USERNAME) \
			-e MONGO_INITDB_ROOT_PASSWORD=$(MONGO_ROOT_PASSWORD) \
			$(MONGO_IMAGE) \
			mongod --replSet $(MONGO_REPLICA_SET) --bind_ip_all --keyFile /etc/mongo-keyfile; \
	fi
	@echo "Waiting for MongoDB to be ready..."
	@$(MAKE) mongo-wait
	@$(MAKE) mongo-init-replica-set

mongo-wait:
	@echo -n "Waiting for MongoDB"
	@for i in $$(seq 1 30); do \
		if docker exec $(MONGO_CONTAINER) mongosh --quiet \
			--username $(MONGO_ROOT_USERNAME) \
			--password $(MONGO_ROOT_PASSWORD) \
			--authenticationDatabase admin \
			--eval "db.adminCommand({ ping: 1 })" >/dev/null 2>&1; then \
			echo -e "\nMongoDB is ready!"; \
			echo "Connect with: mongodb://$(MONGO_ROOT_USERNAME):$(MONGO_ROOT_PASSWORD)@127.0.0.1:$(MONGO_PORT)/picpac?authSource=admin&replicaSet=$(MONGO_REPLICA_SET)"; \
			exit 0; \
		fi; \
		echo -n "."; \
		sleep 1; \
	done; \
	echo -e "\nMongoDB failed to start within 30 seconds"; \
	docker logs $(MONGO_CONTAINER) --tail 10; \
	exit 1

mongo-init-replica-set:
	@echo "Initializing MongoDB replica set '$(MONGO_REPLICA_SET)' if needed..."
	@docker exec $(MONGO_CONTAINER) mongosh --quiet \
		--username $(MONGO_ROOT_USERNAME) \
		--password $(MONGO_ROOT_PASSWORD) \
		--authenticationDatabase admin \
		--eval 'try { const status = rs.status(); print("Replica set already initialized: " + status.set); } catch (e) { if (e.codeName === "NotYetInitialized" || e.message.includes("no replset config has been received")) { rs.initiate({_id: "$(MONGO_REPLICA_SET)", members: [{_id: 0, host: "127.0.0.1:$(MONGO_PORT)"}]}); print("Replica set initialized: $(MONGO_REPLICA_SET)"); } else { throw e; } }'

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

mongo-recreate: mongo-clean mongo

mongo-status:
	@docker exec $(MONGO_CONTAINER) mongosh --quiet \
		--username $(MONGO_ROOT_USERNAME) \
		--password $(MONGO_ROOT_PASSWORD) \
		--authenticationDatabase admin \
		--eval 'rs.status()'

mongo-logs:
	docker logs $(MONGO_CONTAINER) --tail 100

mongo-shell:
	@if ! docker ps --format '{{.Names}}' | grep -q '^$(MONGO_CONTAINER)$$'; then \
		echo "MongoDB container '$(MONGO_CONTAINER)' is not running; starting it first..."; \
		$(MAKE) mongo; \
	fi
	docker exec -it $(MONGO_CONTAINER) mongosh \
		--username $(MONGO_ROOT_USERNAME) \
		--password $(MONGO_ROOT_PASSWORD) \
		--authenticationDatabase admin
