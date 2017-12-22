BASEDIR := $(shell echo $${PWD})

osx: 
	@go build -race -o build/run_build main.go

build:
	@mkdir build

install-osx: build osx
	@cp $(BASEDIR)/build/run_build $(GOPATH)/bin/run

install:
	$(MAKE) install-osx
