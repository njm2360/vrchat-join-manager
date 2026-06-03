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

run: generate
	go run ./cmd/server

web-install:
	cd $(WEB) && pnpm install

web-dev: web-install
	cd $(WEB) && pnpm run dev

web-build: web-install
	cd $(WEB) && pnpm run build

build: generate
	$(SETENV) go build -trimpath -ldflags="-s -w" -o $(BIN) ./cmd/server
