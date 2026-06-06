PNPM ?= pnpm

.DEFAULT_GOAL := build

.PHONY: install
install:
	$(PNPM) install

.PHONY: gen
gen: install
	$(PNPM) run gen:api

.PHONY: dev
dev: gen
	$(PNPM) run dev

.PHONY: build
build: gen
	$(PNPM) run build

.PHONY: lint
lint:
	$(PNPM) run lint

.PHONY: format
format:
	$(PNPM) run format

.PHONY: clean
clean:
	rm -rf dist
