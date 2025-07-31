# GlusterFS Cluster Makefile

.PHONY: help build build-all push push-all clean start stop status logs test

# Variables
REGISTRY ?= gluster-cluster
VERSION ?= latest
GLUSTER_CLUSTER_NAME ?= gluster-cluster
DOCKER_COMPOSE_FILE ?= deployments/docker/docker-compose.yml

# Docker images
MANAGER_IMAGE = $(REGISTRY)/manager:$(VERSION)
NODE_IMAGE = $(REGISTRY)/node:$(VERSION)
CLIENT_IMAGE = $(REGISTRY)/client:$(VERSION)

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build all Docker images
	@echo "Building GlusterFS cluster images..."
	docker build -f deployments/docker/Dockerfile.gluster-manager -t $(MANAGER_IMAGE) .
	docker build -f deployments/docker/Dockerfile.gluster-node -t $(NODE_IMAGE) .
	docker build -f deployments/docker/Dockerfile.gluster-client -t $(CLIENT_IMAGE) .

build-manager: ## Build manager image
	docker build -f deployments/docker/Dockerfile.gluster-manager -t $(MANAGER_IMAGE) .

build-node: ## Build node image
	docker build -f deployments/docker/Dockerfile.gluster-node -t $(NODE_IMAGE) .

build-client: ## Build client image
	docker build -f deployments/docker/Dockerfile.gluster-client -t $(CLIENT_IMAGE) .

push: build ## Build and push all images to registry
	@echo "Pushing images to registry..."
	docker push $(MANAGER_IMAGE)
	docker push $(NODE_IMAGE)
	docker push $(CLIENT_IMAGE)

clean: ## Clean up containers and images
	@echo "Cleaning up..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) down -v --remove-orphans || true
	docker system prune -f

start: ## Start the GlusterFS cluster
	@echo "Starting GlusterFS cluster..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) up -d
	@echo "Waiting for cluster to be ready..."
	sleep 30
	@echo "Cluster started successfully!"
	@echo "Use 'make status' to check cluster status"

stop: ## Stop the GlusterFS cluster
	@echo "Stopping GlusterFS cluster..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) down

status: ## Show cluster status
	@echo "=== Container Status ==="
	docker-compose -f $(DOCKER_COMPOSE_FILE) ps
	@echo ""
	@echo "=== GlusterFS Cluster Status ==="
	@if docker ps --format "table {{.Names}}" | grep -q "$(GLUSTER_CLUSTER_NAME)-manager"; then \
		docker exec $(GLUSTER_CLUSTER_NAME)-manager gluster-manager status || echo "Manager not ready yet"; \
	else \
		echo "Manager container not running"; \
	fi

logs: ## Show logs from all services
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

logs-manager: ## Show manager logs
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f gluster-manager

logs-nodes: ## Show node logs
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f gluster-node1 gluster-node2 gluster-node3

restart: stop start ## Restart the cluster

test: ## Run basic tests
	@echo "Running basic cluster tests..."
	@if docker ps --format "table {{.Names}}" | grep -q "$(GLUSTER_CLUSTER_NAME)-node1"; then \
		echo "Testing node connectivity..."; \
		docker exec $(GLUSTER_CLUSTER_NAME)-node1 gluster peer status; \
		echo "Testing volume status..."; \
		docker exec $(GLUSTER_CLUSTER_NAME)-node1 gluster volume status || echo "No volumes created yet"; \
	else \
		echo "Cluster is not running. Use 'make start' first."; \
		exit 1; \
	fi

# Development targets
dev-setup: ## Setup development environment
	@echo "Setting up development environment..."
	go mod tidy
	go mod download

dev-build: ## Build Go binaries for development
	@echo "Building Go binaries..."
	go build -o bin/gluster-manager ./cmd/gluster-manager
	go build -o bin/gluster-node ./cmd/gluster-node

dev-test: ## Run Go tests
	@echo "Running Go tests..."
	go test -v ./...

dev-lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run

# Example configurations
example-tomcat: ## Start example with Tomcat application
	@echo "Starting example with Tomcat..."
	@cat examples/environment-variables.txt > .env.example
	GLUSTER_VOLUMES="shared:replicated:3:/mnt/shared/tomcat-resources:/opt/tomcat/webapps/zenoptics/resources" \
	docker-compose -f examples/simple-deployment.yml up -d

# Kubernetes targets
k8s-deploy: ## Deploy to Kubernetes
	@echo "Deploying to Kubernetes..."
	kubectl apply -f deployments/k8s/

k8s-delete: ## Delete from Kubernetes
	@echo "Deleting from Kubernetes..."
	kubectl delete -f deployments/k8s/

k8s-status: ## Show Kubernetes status
	@echo "=== Kubernetes Status ==="
	kubectl get all -n gluster-system

# Generate manifests with different configurations
generate-5-node: ## Generate 5-node cluster configuration
	@echo "Generating 5-node cluster configuration..."
	@GLUSTER_NODE_IPS="172.20.0.10,172.20.0.11,172.20.0.12,172.20.0.13,172.20.0.14" \
	GLUSTER_REPLICA_COUNT=5 \
	envsubst < deployments/docker/docker-compose.yml > docker-compose-5node.yml
	@echo "Generated: docker-compose-5node.yml"

docs: ## Generate documentation
	@echo "Generating documentation..."
	@echo "See README.md for usage instructions"