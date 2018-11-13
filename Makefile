PWD = $(shell pwd)

PACKAGE_NAME = github.com/uber/makisu
PACKAGE_VERSION = $(shell git describe --always --tags)
OS = $(shell uname)

ALL_SRC = $(shell find . -name "*.go" | grep -v -e vendor \
	-e ".*/\..*" \
	-e ".*/_.*" \
	-e ".*/mocks.*" \
	-e ".*/*.pb.go")
ALL_PKGS = $(shell go list $(sort $(dir $(ALL_SRC))) | grep -v vendor)
FMT_SRC = $(shell echo "$(ALL_SRC)" | tr ' ' '\n')
EXT_TOOLS = github.com/axw/gocov/gocov github.com/AlekSi/gocov-xml github.com/matm/gocov-html github.com/golang/mock/mockgen golang.org/x/lint/golint golang.org/x/tools/cmd/goimports
EXT_TOOLS_DIR = ext-tools/$(OS)
DEP_TOOL = $(EXT_TOOLS_DIR)/dep

BUILD_LDFLAGS = -X $(PACKAGE_NAME)/lib/utils.BuildHash=$(PACKAGE_VERSION)
GO_FLAGS = -gcflags '-N -l' -ldflags "$(BUILD_LDFLAGS)"
GO_VERSION = 1.10

REGISTRY ?= gcr.io/makisu-project

# Targets to compile the makisu binaries.
.PHONY: cbins
bins: bins/makisu-builder bins/makisu-worker bins/makisu-client

bins/%: $(ALL_SRC) vendor
	@mkdir -p bins
	CGO_ENABLED=0 GOOS=linux go build -tags bins $(GO_FLAGS) -o $@ cmd/$(notdir $@)/*.go

cbins:
	docker run -i --rm -v $(PWD):/go/src/$(PACKAGE_NAME) \
		--net=host \
		--entrypoint=bash \
		-w /go/src/$(PACKAGE_NAME) \
		golang:$(GO_VERSION) \
		-c "make bins"

# Targets to install the dependencies.
$(DEP_TOOL):
	mkdir -p $(EXT_TOOLS_DIR)
	go get github.com/golang/dep/cmd/dep
	cp $(GOPATH)/bin/dep $(EXT_TOOLS_DIR)

vendor: $(DEP_TOOL) Gopkg.toml
	$(EXT_TOOLS_DIR)/dep ensure

cvendor:
	docker run --rm -v $(PWD):/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		--entrypoint=/bin/sh \
		instrumentisto/dep \
		-c "dep ensure"

ext-tools: vendor $(EXT_TOOLS)

.PHONY: $(EXT_TOOLS)
$(EXT_TOOLS): vendor
	@echo "Installing external tool $@"
	@(ls $(EXT_TOOLS_DIR)/$(notdir $@) > /dev/null 2>&1) || GOBIN=$(PWD)/$(EXT_TOOLS_DIR) go install ./vendor/$@

mocks: ext-tools
	@echo "Generating mocks"
	mkdir -p mocks/net/http
	$(EXT_TOOLS_DIR)/mockgen -destination=mocks/net/http/mockhttp.go -package=mockhttp net/http RoundTripper

env: integration/python/requirements.txt
	[ -d env ] || virtualenv --setuptools env
	./env/bin/pip install -q -r integration/python/requirements.txt

# Target to build the makisu docker image. The docker image contains the builder and worker
# binaries.
.PHONY: images publish
image-%: bins
	docker build -t $(REGISTRY)/makisu-$*:$(PACKAGE_VERSION) -f dockerfiles/$*.df .
	docker tag $(REGISTRY)/makisu-$*:$(PACKAGE_VERSION) makisu-$*:$(PACKAGE_VERSION)

publish-%:
	$(MAKE) image-$*
	docker push $(REGISTRY)/makisu-$*:$(PACKAGE_VERSION) 

images: image-builder image-worker image-client

publish: publish-builder publish-worker publish-client

# Targets to test the codebase.
.PHONY: test unit-test integration cunit-test
test: unit-test integration

unit-test: $(ALL_SRC) vendor ext-tools mocks
	$(EXT_TOOLS_DIR)/gocov test $(ALL_PKGS) --tags "unit" | $(EXT_TOOLS_DIR)/gocov report

cunit-test: $(ALL_SRC) vendor
	docker run -i --rm -v $(PWD):/go/src/$(PACKAGE_NAME) \
		--net=host \
		--entrypoint=bash \
		-w /go/src/$(PACKAGE_NAME) \
		golang:$(GO_VERSION) \
		-c "make ext-tools unit-test"

integration: bins env image-builder
	PACKAGE_VERSION=$(PACKAGE_VERSION) ./env/bin/py.test --maxfail=1 --durations=6 --timeout=300 -vv integration/python

# Misc targets
.PHONY: clean
clean:
	git clean -fd
	-rm -rf bins vendor ext-tools mocks env
