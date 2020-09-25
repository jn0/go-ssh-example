GOFILES := $(shell echo *.go)
INSTALL_PREFIX := /usr/local
BIN_PATH := $(INSTALL_PREFIX)/bin
INSTALL := install
INSTALL_NAME := rupdate
GO_EXE := $(shell bash -c 'type -P go')
GO_EXE := $(shell realpath -e $(GO_EXE))
# GOROOT := $(shell dirname $(GO_EXE))
# GOROOT := $(shell dirname $(GOROOT))
GOPATH := $(shell pwd)/.go
GOBIN := $(GOPATH)/bin
RM := rm -rf
# GO := env GOPATH=$(GOPATH) GOBIN=$(GOBIN) GOROOT=$(GOROOT) go
GO := env GOPATH=$(GOPATH) GOBIN=$(GOBIN) go

.PHONY:	all clean

all:	go-ssh-example
	@echo

install:	go-ssh-example
	$(INSTALL) $^ $(BIN_PATH)/$(INSTALL_NAME)

.go:	$(GOFILES)
	mkdir -pv .go .go/bin
	$(GO) get -v
	touch .go

go-ssh-example:	.go $(GOFILES)
	$(RM) $@
	$(GO) build

clean:
	$(RM) go-ssh-example .go
# EOF #
