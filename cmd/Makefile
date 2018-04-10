PKG=github.com/exoscale/egoscale

GOPATH=$(CURDIR)/.gopath
DEP=$(GOPATH)/bin/dep

SRCS=$(wildcard *.go)

DEST=build
BIN=cs

export GOPATH

.PHONY: default
default: $(BIN)

$(GOPATH)/src/$(PKG):
	mkdir -p $(GOPATH)
	go get -u github.com/golang/dep/cmd/dep
	mkdir -p $(shell dirname $(GOPATH)/src/$(PKG))
	ln -sf ../../../../.. $(GOPATH)/src/$(PKG)

$(BIN): $(SRCS)
	(cd $(GOPATH)/src/$(PKG)/cmd && \
		go build \
			-o $@ \
			$<)

.PHONY: deps
deps: $(GOPATH)/src/$(PKG)
	(cd $(GOPATH)/src/$(PKG)/cmd && \
		$(DEP) ensure)

.PHONY: deps-update
deps-update: deps
	(cd $(GOPATH)/src/$(PKG)/cmd && \
		$(DEP) ensure -update)

.PHONY: clean
clean:
	$(RM) -r $(DEST)
	go clean
