NAME := do
ORG := jcoc
PKG := $(ORG)/$(NAME)
BASEDIR := $(shell echo $${PWD})
PROG_NAME := do
DOCKERFILE ?= etc/docker/Dockerfile
DOCKERNAME ?= $(NAME)
DOCKERPKG ?= $(ORG)/$(DOCKERNAME):$(DOCKERLABEL)
SHA := $(shell git rev-parse HEAD)
VERSION := $(shell git describe --always --dirty='-dev')
BUILD := $(shell git rev-parse HEAD | cut -c1-8)

build: version

version:
	@echo "version: $(VERSION) build: $(BUILD) package: $(PKG) docker: $(DOCKERPKG) sha: $(SHA)"

osx: build
ifeq ($(UNAME_S),Darwin)
	@go build -race -tags client -ldflags '-s -w -X=github.com/$(PKG)/cmd.BUILD=$(BUILD) -X=github.com/$(PKG)/cmd.VERSION=$(VERSION) -X=github.com/$(PKG)/cmd.SHA=$(SHA)' -o build/$(PROG_NAME)-osx-amd64-$(VERSION) $(PKGMAIN)
endif

install-osx: osx
	@cp $(BASEDIR)/build/$(PROG_NAME)-osx-amd64-$(VERSION) $(GOPATH)/bin/$(PROG_NAME)

install:
	$(MAKE) install-osx
