.PHONY: help check backend-build backend-test backend-vet backend-run web-install web-build web-lint up down fmt

help:
	@echo "Neko 平台 — 常用命令"
	@echo "  make check          运行全部检查（合并主线前必须通过）"
	@echo "  make backend-build  编译后端"
	@echo "  make backend-test   后端单元测试"
	@echo "  make backend-run    运行 API 服务"
	@echo "  make web-build      前端构建"
	@echo "  make up / down      启动 / 停止本地依赖"

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
