# Makefile for Erato
# https://www.gnu.org/software/make/manual/make.html

MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules
MAKEFLAGS += --no-builtin-variables

.SUFFIXES:
.DELETE_ON_ERROR:
.DEFAULT_GOAL := help

SHELL := /usr/bin/env bash -euo pipefail -c

#: Go binary
GO ?= go

#: Nix binary
NIX ?= nix

rwildcard = $(foreach d,$(wildcard $(1:=/*)),$(call rwildcard,$d,$2) $(filter $(subst *,%,$2),$d))
check_defined = $(strip $(foreach 1,$1,$(call __check_defined,$1,$(strip $(value 2)))))
__check_defined = $(if $(value $1),,$(error Undefined $1$(if $2, ($2))))

GO_MOD_FILES = go.mod go.sum

BUF_CONFIG_FILES = buf.yaml buf.gen.yaml buf.lock
PROTO_FILES = $(call rwildcard,proto,*.proto)
PROTO_GEN_FILES = $(subst proto,internal/gen,$(PROTO_FILES:.proto=.pb.go))
GEN_FILES += $(PROTO_GEN_FILES)

SQLC_CONFIG_FILES = sqlc.yaml
SQL_DIR = internal/storage/db
SQL_MIGRATIONS_DIR = $(SQL_DIR)/migrations
SQL_FILES = $(call rwildcard,$(SQL_MIGRATIONS_DIR),*.sql)
SQL_FILES += $(SQL_DIR)/queries.sql
SQL_GEN_FILES += $(SQL_DIR)/models.go
SQL_GEN_FILES += $(SQL_DIR)/queries.sql.go
SQL_GEN_FILES += $(SQL_DIR)/db.go
GEN_FILES += $(SQL_GEN_FILES)

TEMPL_DIR = internal/app/component
TEMPL_FILES = $(call rwildcard,$(TEMPL_DIR),*.templ)
TEMPL_GEN_FILES = $(TEMPL_FILES:.templ=_templ.go)
GEN_FILES += $(TEMPL_GEN_FILES)

.PHONY: all
#: Run format, generate, lint, and test
all: format generate lint test

.PHONY: build
#: Build the erato binary
build: $(GEN_FILES)
	$(GO) build -o .tmp/erato ./cmd/erato

.PHONY: build-nix
#: Build the erato nix package
build-nix:
	$(NIX) build --out-link .tmp/result

.PHONY: watch
#: Start hot-reload development server
watch:
	$(GO) tool air

.PHONY: test
#: Run all unit and UI tests
test: $(GEN_FILES)
	$(GO) test -v -cover -race -vet=off ./...

.PHONY: test-unit
#: Run unit tests only (skip UI tests)
test-unit: $(GEN_FILES)
	$(GO) test -v -cover -race -vet=off -short ./...

.PHONY: test-ui
#: Run UI tests only
test-ui: $(GEN_FILES)
	$(GO) test -v -race -vet=off ./internal/uitest/...

.PHONY: format
#: Format all code (Go, protos, templ)
format: format-go format-protos format-templ

.PHONY: format-go
format-go: $(GEN_FILES)
	$(GO) tool golangci-lint fmt
	$(GO) mod tidy

.PHONY: format-protos
format-protos:
	$(GO) tool buf format -w

.PHONY: format-templ
format-templ:
	$(GO) tool templ fmt .

.PHONY: format-nix
format-nix:
	$(NIX) fmt

.PHONY: lint
#: Lint all code (Go and protos)
lint: lint-go lint-protos

.PHONY: lint-fix
#: Lint Go code and auto-fix issues
lint-fix: $(GEN_FILES)
	$(GO) tool golangci-lint run --fix

.PHONY: lint-go
lint-go: $(GEN_FILES)
	$(GO) tool golangci-lint run

.PHONY: lint-protos
lint-protos:
	$(GO) tool buf lint

.PHONY: lint-nix
lint-nix:
	$(NIX) flake check

.PHONY: generate
#: Run all code generators
generate: $(GEN_FILES)

.PHONY: generate-protos
generate-protos:
	$(GO) tool buf generate

$(PROTO_GEN_FILES) &: $(GO_MOD_FILES) $(BUF_CONFIG_FILES) $(PROTO_FILES)
	$(MAKE) generate-protos

.PHONY: generate-sql
generate-sql:
	$(GO) tool sqlc generate --no-remote

$(SQL_GEN_FILES) &: $(GO_MOD_FILES) $(SQLC_CONFIG_FILES) $(SQL_FILES)
	$(MAKE) generate-sql

.PHONY: new-goose-migration
#: Create a new SQL migration (requires NAME=migration_name)
new-goose-migration:
	@:$(call check_defined, NAME)
	$(GO) tool goose -dir "$(SQL_MIGRATIONS_DIR)" create "$(NAME)" sql

.PHONY: generate-templ
generate-templ:
	$(GO) tool templ generate

$(TEMPL_GEN_FILES) &: $(GO_MOD_FILES) $(TEMPL_FILES)
	$(MAKE) generate-templ

.PHONY: update
#: Update all dependencies (Go, buf, htmx, nix)
update: update-go update-buf update-htmx update-nix

.PHONY: update-go
update-go:
	$(GO) get -u ./...
	$(GO) mod tidy

.PHONY: update-buf
update-buf:
	$(GO) tool buf dep update

.PHONY: update-htmx
update-htmx:
	$(eval HTMX_VERSION := $(shell curl -sf https://api.github.com/repos/bigskysoftware/htmx/releases/latest | jq -r '.tag_name'))
	@if [ -z "$(HTMX_VERSION)" ]; then echo "error: failed to fetch htmx version from GitHub API" >&2; exit 1; fi
	curl -sfL "https://unpkg.com/htmx.org@$(HTMX_VERSION)/dist/htmx.min.js" -o internal/app/static/htmx.min.js

.PHONY: update-nix
update-nix:
	$(NIX) flake update
	$(NIX) run nixpkgs#nix-update -- --flake --version=skip erato

# Help targets (https://github.com/rodaine/make-help)

.PHONY: help
#: Show this help
help: help_variables help_targets

.PHONY: help_variables
help_variables: $(eval ECHO=$(shell \
	sed -nrE \
		-e 'N;s/^#:[ 	]*(.+)\n[ 	]*([^ 	+?:=]+)[ 	]*([+?:]|::)?=.*$$/\2	\1	\$$(\2)/p;D' \
		$(MAKEFILE_LIST) \
	| awk '!x[$$1]++' \
	| sort -d \
	| { echo 'VARIABLE	DESCRIPTION	VALUE'; cat; } \
	| column -t -s'	' \
	| tr '\n' '\1'))
	@echo "$(ECHO)" | tr '\1' '\n'

.PHONY: help_targets
help_targets:
	@sed -nrE \
		-e 'N;s/^#:[ 	]*(.+)\n([^ 	:]+)[ 	]*[:]{1,2}($$|[^=].*$$)/\2	\1/p;D' \
		$(MAKEFILE_LIST) \
	| awk '!x[$$1]++' \
	| sort -d \
	| { echo 'TARGET	DESCRIPTION'; cat; } \
	| column -t -s'	'
