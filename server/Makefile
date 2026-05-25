.PHONY: generate run build vet tidy web-install web-dev web-build

ifeq ($(OS),Windows_NT)
    SETENV := set CGO_ENABLED=0&&
    BIN := bin/vjm-server.exe
else
    SETENV := CGO_ENABLED=0
    BIN := bin/vjm-server
endif

GEN := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
WEB := ../web

generate:
	$(GEN) -config oapi-codegen.yaml ../api/openapi.yaml

tidy:
	go mod tidy

vet:
	go vet ./...

run:
	go run ./cmd/server

# フロントエンド (Vite + React) の依存インストール
web-install:
	cd $(WEB) && npm install

# フロントエンドのDev server (Vite, port 5173, /api を :8080 にプロキシ)
web-dev:
	cd $(WEB) && npm run dev

# フロントエンドの本番ビルド (出力先: server/static)
web-build:
	cd $(WEB) && npm run build

build: web-build
	$(SETENV) go build -trimpath -ldflags="-s -w" -o $(BIN) ./cmd/server
