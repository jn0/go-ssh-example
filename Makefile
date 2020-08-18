GOFILES := $(shell echo *.go)
INSTALL_PREFIX := /usr/local
BIN_PATH := $(INSTALL_PREFIX)/bin
INSTALL := install
INSTALL_NAME := rupdate
GO_EXE := $(shell bash -c 'type -P go')
GO_EXE := $(shell realpath -e $(GO_EXE))
GOROOT := $(shell dirname $(GO_EXE))
GOROOT := $(shell dirname $(GOROOT))
GOPATH := $(shell realpath -e .go)
GOBIN := $(GOPATH)/bin
RM := rm -rf
GO := env GOPATH=$(GOPATH) GOBIN=$(GOBIN) GOROOT=$(GOROOT) go

.PHONY:	all clean

all:	go-ssh-example
	@echo

install:	go-ssh-example
	$(INSTALL) $^ $(BIN_PATH)/$(INSTALL_NAME)

$(GOPATH):	$(GOFILES)
	mkdir -pv $(GOPATH) $(GOBIN)
	$(GO) get -v
	touch $(GOPATH)

go-ssh-example:	$(GOPATH) $(GOFILES)
	$(RM) $@
	$(GO) get -v -u
	$(GO) build

clean:
	$(RM) go-ssh-example $(GOPATH)
# EOF #
