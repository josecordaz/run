BASEDIR := $(shell echo $${PWD})

osx: 
ifeq ($(UNAME_S),Darwin)
	@go build -race -o build/run_build main.go
endif

install-osx: osx
	@cp $(BASEDIR)/build/run_build $(GOPATH)/bin/run

install:
	$(MAKE) install-osx
