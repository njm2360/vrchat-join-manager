.PHONY: gen run build vet tidy web-install web-gen web-dev web-build

ifeq ($(OS),Windows_NT)
    SETENV := set CGO_ENABLED=0&&
    BIN := bin/vjm-server.exe
else
    SETENV := CGO_ENABLED=0
    BIN := bin/vjm-server
endif

GEN := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
WEB := ../web

gen:
	$(GEN) -config oapi-codegen.yaml ../api/openapi.yaml

tidy:
	go mod tidy

vet:
	go vet ./...

run: gen
	go run .

build: gen
	$(SETENV) go build -trimpath -ldflags="-s -w" -o $(BIN) .

web-install:
	cd $(WEB) && pnpm install

web-gen: web-install
	cd $(WEB) && pnpm run gen:api

web-dev: web-gen
	cd $(WEB) && pnpm run dev

web-build: web-gen
	cd $(WEB) && pnpm run build
