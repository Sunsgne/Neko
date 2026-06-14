.PHONY: help check demo backend-build backend-test backend-vet backend-run backend-run-seed web-install web-build web-lint up down fmt

help:
	@echo "Neko 平台 — 常用命令"
	@echo "  make demo           一键本地 Demo（后端+演示数据+前端，无需外部依赖）"
	@echo "  make deploy         Ubuntu 24.04 一键容器化部署（生产，自动生成密钥）"
	@echo "  make deploy-down    停止部署的全栈服务"
	@echo "  make check          运行全部检查（合并主线前必须通过）"
	@echo "  make backend-build  编译后端"
	@echo "  make backend-test   后端单元测试"
	@echo "  make backend-run    运行 API 服务"
	@echo "  make backend-run-seed 运行 API 服务（含演示数据）"
	@echo "  make web-build      前端构建"
	@echo "  make up / down      启动 / 停止本地依赖"

demo:
	./scripts/demo.sh

deploy:
	./scripts/deploy-ubuntu.sh

deploy-down:
	docker compose -f docker-compose.yml -f docker-compose.deploy.yml down

check: backend-vet backend-test backend-build web-build
	@echo "✅ all checks passed"

backend-vet:
	cd backend && go vet ./...

backend-test:
	cd backend && go test ./...

backend-build:
	cd backend && go build -o bin/api ./cmd/api && go build -o bin/worker ./cmd/worker

backend-run:
	cd backend && go run ./cmd/api

backend-run-seed:
	cd backend && NEKO_SEED=true go run ./cmd/api

fmt:
	cd backend && gofmt -w .

web-install:
	cd web && npm install

web-build:
	cd web && npm run build

web-lint:
	cd web && npm run lint

up:
	docker compose up -d

down:
	docker compose down
