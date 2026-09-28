SHELL := /bin/sh

API_DIR := api
WEB_DIR := web
IMAGE ?= counter-drop:local
AWS_REGION ?= ap-south-1
AWS_ACCOUNT_ID ?=
ECR_REPOSITORY ?= counter-drop
IMAGE_TAG ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo local)
# Privacy/terms contact baked into the web build: make build-image VITE_OPERATOR_NAME="..." VITE_SUPPORT_EMAIL=...
BUILD_ARGS = --build-arg VITE_OPERATOR_NAME="$(VITE_OPERATOR_NAME)" --build-arg VITE_SUPPORT_EMAIL="$(VITE_SUPPORT_EMAIL)"

.PHONY: help dev-db api-dev web-dev test test-api test-web build build-web build-image aws-image aws-push launch-check clean

help:
	@printf '%s\n' \
	  'Counter Drop commands:' \
	  '  make dev-db        Start local Postgres' \
	  '  make api-dev       Run API on :18080 for LAN phone testing' \
	  '  make web-dev       Run Vite on :5174 for LAN phone testing' \
	  '  make test          Run API tests and web build' \
	  '  make build         Build web and API binary' \
	  '  make build-image   Build production Docker image' \
	  '  make aws-image     Build image tagged for AWS ECR' \
	  '  make aws-push      Push image to AWS ECR' \
	  '  make launch-check  Run launch readiness checks'

dev-db:
	docker compose -f deploy/docker-compose.yml up -d postgres

api-dev:
	cd $(API_DIR) && CD_API_ADDR=:18080 CD_PUBLIC_API_URL=$${CD_PUBLIC_API_URL:-http://localhost:18080} CD_PUBLIC_WEB_URL=$${CD_PUBLIC_WEB_URL:-http://localhost:5174} CD_WEB_ORIGINS=$${CD_WEB_ORIGINS:-http://localhost:5174} CD_DATABASE_URL=$${CD_DATABASE_URL:-postgres://counter_drop:counter_drop@localhost:55433/counter_drop?sslmode=disable} go run ./cmd/api

web-dev:
	cd $(WEB_DIR) && VITE_API_PROXY=$${VITE_API_PROXY:-http://localhost:18080} npm run dev -- --host 0.0.0.0 --port 5174

test: test-api test-web

test-api:
	cd $(API_DIR) && go test ./...

test-web:
	cd $(WEB_DIR) && npm run build

build: build-web
	mkdir -p bin
	cd $(API_DIR) && CGO_ENABLED=0 go build -trimpath -o ../bin/counter-drop-api ./cmd/api
	cd $(API_DIR) && CGO_ENABLED=0 go build -trimpath -o ../bin/cdadmin ./cmd/cdadmin

build-web:
	cd $(WEB_DIR) && npm run build

build-image:
	docker build $(BUILD_ARGS) -t $(IMAGE) -f Dockerfile .

aws-image:
	@test -n "$(AWS_ACCOUNT_ID)" || (echo "Set AWS_ACCOUNT_ID=..." && exit 1)
	docker build $(BUILD_ARGS) -t $(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/$(ECR_REPOSITORY):$(IMAGE_TAG) -f Dockerfile .

aws-push: aws-image
	aws ecr get-login-password --region $(AWS_REGION) | docker login --username AWS --password-stdin $(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com
	docker push $(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/$(ECR_REPOSITORY):$(IMAGE_TAG)

launch-check: test build-image
	@printf '%s\n' 'Launch checks passed. Review docs/LAUNCH.md before deploying.'

clean:
	rm -rf bin web/dist web/tsconfig.tsbuildinfo
