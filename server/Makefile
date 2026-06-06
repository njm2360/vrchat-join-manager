GO ?= go

GEN := $(GO) run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen

ifeq ($(OS),Windows_NT)
    SETENV := set CGO_ENABLED=0&&
    BIN := bin/vjm-server.exe
else
    SETENV := CGO_ENABLED=0
    BIN := bin/vjm-server
endif

.DEFAULT_GOAL := build

.PHONY: gen
gen:
	$(GEN) -config oapi-codegen.yaml ../api/openapi.yaml

.PHONY: run
run: gen
	$(GO) run .

.PHONY: build
build: gen
	$(SETENV) $(GO) build -trimpath -ldflags="-s -w" -o $(BIN) .

.PHONY: test
test:
	$(GO) test ./...

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: tidy
tidy:
	$(GO) mod tidy

.PHONY: clean
clean:
	rm -rf bin
