SERVER_DIR := server
AGENT_DIR  := agent
WEB_DIR    := web

MAKE ?= make

.DEFAULT_GOAL := build

# ---------------------------------------------------------------------------
# Web (Vite)
# ---------------------------------------------------------------------------

.PHONY: web-build
web-build:
	$(MAKE) -C $(WEB_DIR) build

.PHONY: web-dev
web-dev:
	$(MAKE) -C $(WEB_DIR) dev

# ---------------------------------------------------------------------------
# Server (Go)
# ---------------------------------------------------------------------------

.PHONY: server-build
server-build:
	$(MAKE) -C $(SERVER_DIR) build

.PHONY: server-run
server-run:
	$(MAKE) -C $(SERVER_DIR) run

# ---------------------------------------------------------------------------
# Agent (Go)
# ---------------------------------------------------------------------------

.PHONY: agent-build
agent-build:
	$(MAKE) -C $(AGENT_DIR) build

.PHONY: agent-installer
agent-installer:
	$(MAKE) -C $(AGENT_DIR) installer

# ---------------------------------------------------------------------------
# Combined
# ---------------------------------------------------------------------------

.PHONY: build
build: web-build server-build

.PHONY: all
all: build agent-build

.PHONY: vet
vet:
	$(MAKE) -C $(SERVER_DIR) vet
	$(MAKE) -C $(AGENT_DIR) vet

.PHONY: tidy
tidy:
	$(MAKE) -C $(SERVER_DIR) tidy
	$(MAKE) -C $(AGENT_DIR) tidy

.PHONY: clean
clean:
	$(MAKE) -C $(WEB_DIR) clean
	$(MAKE) -C $(SERVER_DIR) clean
	$(MAKE) -C $(AGENT_DIR) clean
