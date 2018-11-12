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
EXT_TOOLS = github.com/axw/gocov/gocov github.com/AlekSi/gocov-xml github.com/matm/gocov-html github.com/golang/mock/gomock github.com/golang/mock/mockgen golang.org/x/lint/golint golang.org/x/tools/cmd/goimports github.com/golang/dep/cmd/dep
EXT_TOOLS_DIR = ext-tools/$(OS)

BUILD_LDFLAGS = -X $(PACKAGE_NAME)/lib/utils.BuildHash=$(PACKAGE_VERSION)
GO_FLAGS = -gcflags '-N -l' -ldflags "$(BUILD_LDFLAGS)"
GO_VERSION = 1.10

# Targets to compile the makisu binaries.
.PHONY: cbins
bins: bins/builder bins/worker bins/client

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
$(EXT_TOOLS_DIR): ext-tools

ext-tools:
	@echo "Installing external tools"
	go get $(EXT_TOOLS)
	mkdir -p $(EXT_TOOLS_DIR)
	GOBIN=$(EXT_TOOLS_DIR) go install ./vendor/github.com/golang/dep/cmd/dep
	# cp $(shell which gocov) $(EXT_TOOLS_DIR)
	# cp $(shell which mockgen) $(EXT_TOOLS_DIR)
	# cp $(shell which dep) $(EXT_TOOLS_DIR)

vendor: Gopkg.toml $(EXT_TOOLS_DIR)
	$(EXT_TOOLS_DIR)/dep ensure

cvendor:
	docker run --rm -v $(PWD):/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		--entrypoint=/bin/sh \
		instrumentisto/dep \
		-c "dep ensure"

mocks: $(EXT_TOOLS_DIR)
	@echo "Generating mocks"
	mkdir -p mocks/net/http
	$(EXT_TOOLS_DIR)/mockgen -destination=mocks/net/http/mockhttp.go -package=mockhttp net/http RoundTripper

# Target to build the makisu docker image. The docker image contains the builder and worker
# binaries.
image: bins/builder bins/worker
	docker build -t makisu:$(PACKAGE_VERSION) -f Dockerfile .

# Targets to test the codebase.
.PHONY: test unit-test integration cunit-test
test: unit-test integration

unit-test: $(ALL_SRC) vendor ext-tools mocks
	$(EXT_TOOLS_DIR)/gocov test $(ALL_PKGS) --tags "unit" | $(EXT_TOOLS_DIR)/gocov report

integration:
	@echo "Nothing for now"

cunit-test: $(ALL_SRC) vendor
	docker run -i --rm -v $(PWD):/go/src/$(PACKAGE_NAME) \
		--net=host \
		--entrypoint=bash \
		-w /go/src/$(PACKAGE_NAME) \
		golang:$(GO_VERSION) \
		-c "rm -rf ext-tools && make ext-tools unit-test"

.PHONY: clean
clean:
	git clean -fd
	-rm -rf bins vendor ext-tools mocks
