GO   ?= go
ISCC ?= iscc.exe

GEN := $(GO) run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen

.DEFAULT_GOAL := build

.PHONY: gen
gen:
	$(GEN) -config oapi-codegen.yaml ../api/openapi.yaml

.PHONY: run
run: gen
	$(GO) run ./cmd/client

.PHONY: build
build: gen
	GOOS=windows $(GO) build -trimpath -ldflags="-s -w" -o bin/vjm-agent.exe ./cmd/client

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

.PHONY: installer
installer: build
	$(ISCC) installer.iss
