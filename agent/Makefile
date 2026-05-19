.PHONY: generate run build vet tidy installer

GEN := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
ISCC ?= iscc.exe

generate:
	$(GEN) -config oapi-codegen.yaml ../api/openapi.yaml

tidy:
	go mod tidy

vet:
	go vet ./...

run:
	go run ./cmd/client

build: export GOOS=windows
build:
	go build -trimpath -ldflags="-s -w" -o bin/vjm-agent.exe ./cmd/client

installer: build
	$(ISCC) installer.iss
