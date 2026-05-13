.PHONY: generate run build vet tidy

ifeq ($(OS),Windows_NT)
    SETENV := set CGO_ENABLED=0&&
    BIN := bin/vjm-server.exe
else
    SETENV := CGO_ENABLED=0
    BIN := bin/vjm-server
endif

GEN := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen

generate:
	$(GEN) -config oapi-codegen.yaml ../api/openapi.yaml

tidy:
	go mod tidy

vet:
	go vet ./...

run:
	go run ./cmd/server

build:
	$(SETENV) go build -trimpath -ldflags="-s -w" -o $(BIN) ./cmd/server
