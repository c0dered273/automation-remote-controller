include .env

PROJECT_NAME=automation-remote-controller
VERSION=0.0.1

PROJECT_DIR=$(shell pwd)
BUILD_DIR=$(PROJECT_DIR)/bin
MAIN=$(PROJECT_DIR)/cmd/$(PROJECT_NAME)/main.go
DOCKER_REGISTRY?= #if set it should finished by /

.PHONY: all
all: clean vendor lint build

## Clean:
.PHONY: clean
clean: ## Remove build related file
	@rm -fr $(BUILD_DIR)
	@rm -f $(PROJECT_DIR)/profile.cov
	@echo "  >  Cleaning build cache"
	@go clean

.PHONY: vendor
vendor: ## Copy of all packages needed to support builds and tests in the vendor directory
	@go mod vendor

.PHONY: generate
generate: ## Generate dependency files
	@echo "  >  Generating dependency files..."
	go generate ./...

## Test:
.PHONY: test
test: generate ## Run the tests
	@echo "  >  Run tests..."
	go test -v -race ./...

.PHONY: coverage
coverage: generate ## Run the tests and export the coverage
	@echo "  >  Checking tests coverage..."
	@go test -cover -covermode=count -coverprofile=profile.cov ./...
	@go tool cover -func profile.cov

## Build:
.PHONY: build
build: test ## Build your project and put the output binary in /bin
	@echo "  >  Building binary..."
	@mkdir -p $(BUILD_DIR)
	# go build -o $(BUILD_DIR)/$(PROJECT_NAME) $(MAIN)
	go build -o $(BUILD_DIR)/rc-tg-bot $(PROJECT_DIR)/cmd/rc-tg-bot/main.go
	go build -o $(BUILD_DIR)/remote-control-client $(PROJECT_DIR)/cmd/remote-control-client/main.go
	go build -o $(BUILD_DIR)/user-account-api $(PROJECT_DIR)/cmd/user-account-api/main.go

## Migrate:
.PHONY: migrate
migrate: ## Migrate database
	@echo "  >  Migrating database..."
	docker run --rm --name golang-migrate -v $(PROJECT_DIR)/migrations:/migrations --network host migrate/migrate -verbose -path=/migrations/ -database 'postgres://test:test@localhost:5432/remote-ctrl?sslmode=disable' up

## Lint:
.PHONY: lint
lint: lint-go #lint-yaml ## Run all available linters

.PHONY: lint-go
lint-go: ## Use golintci-lint on your project
	@echo "  >  Running go linters..."
#	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:latest-alpine golangci-lint run --deadline=65s
	golangci-lint -v run

#lint-yaml: ## Use yamllint on the yaml file of your projects
#	docker run --rm -it -v $(shell pwd):/data cytopia/yamllint -f parsable $(shell git ls-files '*.yml' '*.yaml')

.PHONY: install
install: docker-build docker-release

## Docker:
.PHONY: docker-build
docker-build: ## Use the dockerfile to build the container
	#docker build --rm --tag $(PROJECT_NAME) .
	docker build --rm --tag rc-tg-bot -f ./docker/rc-tg-bot/Dockerfile .
	docker build --rm --memory=2048m --tag remote-control-client -f ./docker/remote-control-client/Dockerfile .
	docker build --rm --tag user-account-api -f ./docker/user-account-api/Dockerfile .

.PHONY: docker-release
docker-release: ## Release the container with tag latest and version
	docker tag $(PROJECT_NAME) $(DOCKER_REGISTRY)$(PROJECT_NAME):latest
	docker tag $(PROJECT_NAME) $(DOCKER_REGISTRY)$(PROJECT_NAME):$(VERSION)
	# Push the docker images
	docker push $(DOCKER_REGISTRY)$(PROJECT_NAME):latest
	docker push $(DOCKER_REGISTRY)$(PROJECT_NAME):$(VERSION)


## Help:
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)

.PHONY: help
help: Makefile ## Show this help.
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    ${YELLOW}%-20s${GREEN}%s${RESET}\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  ${CYAN}%s${RESET}\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)