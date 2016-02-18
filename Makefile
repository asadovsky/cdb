SHELL := /bin/bash -euo pipefail
PATH := node_modules/.bin:$(PATH)
GOPATH := $(HOME)/dev/go

define BROWSERIFY
	@mkdir -p $(dir $2)
	browserify $1 -d -t [ envify purge ] -o $2
endef

.DELETE_ON_ERROR:

all: build

node_modules: package.json
	npm prune
	npm install
	touch $@

.PHONY: build

build: dist/demo.min.js
dist/demo.min.js: demo/index.js $(shell find client) node_modules
	$(call BROWSERIFY,$<,$@)

build: dist/demo
dist/demo: $(shell find demo server)
	go build -o $@ github.com/asadovsky/cdb/demo

build: dist/server
dist/server: $(shell find server)
	go build -o $@ github.com/asadovsky/cdb/server

########################################
# Demos

.PHONY: demo-store
demo-store: build
	dist/demo -port=4001 | xargs -n 1 -t open

########################################
# Test, clean, and lint

.PHONY: test
test:
	go test github.com/asadovsky/cdb/...

.PHONY: clean
clean:
	rm -rf dist node_modules

.PHONY: lint
lint: node_modules
	go vet github.com/asadovsky/cdb/...
	jshint .
