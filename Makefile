SHELL := /bin/bash -euo pipefail
PATH := node_modules/.bin:$(PATH)
GOPATH := $(HOME)/dev/go

define BROWSERIFY
	@mkdir -p $(dir $2)
	browserify $1 -d -t [ envify purge ] -o $2
endef

define BROWSERIFY_STANDALONE
	@mkdir -p $(dir $2)
	browserify $1 -s cdb.$3 -d -t [ envify purge ] -o $2
endef

.DELETE_ON_ERROR:

all: build

node_modules: package.json
	npm prune
	npm install
	touch $@

.PHONY: build

build: dist/server
dist/server: $(shell find server)
	go build -o $@ github.com/asadovsky/cdb/server

########################################
# Test, clean, and lint

.PHONY: clean
clean:
	rm -rf dist node_modules

.PHONY: lint
lint: node_modules
	go vet github.com/asadovsky/cdb/server/...
	jshint .
