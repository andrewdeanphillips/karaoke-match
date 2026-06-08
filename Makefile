IMAGE_BASE = asia-northeast1-docker.pkg.dev/karaokematch-dev/karaoke-match/api
TAG        ?= $(shell git rev-parse --short HEAD)
IMAGE       = $(IMAGE_BASE):$(TAG)

.PHONY: help test dev-backend dev-frontend build deploy migrate-up migrate-down setup

help: ## show available targets
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | awk -F':.*##' '{printf "  %-16s %s\n", $$1, $$2}'

test: ## run all backend tests
	cd backend && go test -count=1 ./...

dev-backend: ## start the backend dev server (requires backend/.env)
	cd backend && go run ./cmd/api

dev-frontend: ## start the frontend dev server
	cd frontend && npm run dev

build: ## build the docker image
	docker build -t $(IMAGE) .

deploy: ## build, push, and deploy to Cloud Run
	@git diff --quiet && git diff --cached --quiet || (echo "error: uncommitted changes — commit or stash before deploying"; exit 1)
	docker build -t $(IMAGE) .
	docker push $(IMAGE)
	gcloud run deploy karaoke-match \
		--image=$(IMAGE) \
		--region=asia-northeast1 \
		--project=karaokematch-dev

migrate-up: ## apply all pending migrations (requires DATABASE_URL)
	migrate -path backend/migrations -database "$(DATABASE_URL)?sslmode=disable" up

migrate-down: ## roll back one migration (requires DATABASE_URL)
	migrate -path backend/migrations -database "$(DATABASE_URL)?sslmode=disable" down 1

setup: ## one-time: configure Docker to authenticate with Artifact Registry
	gcloud auth configure-docker asia-northeast1-docker.pkg.dev
