#  Copyright (c) 2018 Uber Technologies, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

PWD := ${CURDIR}

PACKAGE_NAME = github.com/uber/makisu
PACKAGE_VERSION ?= $(shell git describe --always --tags)
OS = $(shell uname)

ALL_SRC = $(shell find . -name "*.go" | grep -v -e vendor \
	-e ".*/\..*" \
	-e ".*/_.*" \
	-e ".*/mocks.*" \
	-e ".*/*.pb.go")
ALL_PKGS = $(shell go list $(sort $(dir $(ALL_SRC))) | grep -v vendor)
ALL_PKG_PATHS = $(shell go list -f '{{.Dir}}' ./...)
FMT_SRC = $(shell echo "$(ALL_SRC)" | tr ' ' '\n')
EXT_TOOLS = github.com/axw/gocov/gocov github.com/AlekSi/gocov-xml github.com/matm/gocov-html github.com/golang/mock/mockgen golang.org/x/lint/golint golang.org/x/tools/cmd/goimports github.com/client9/misspell/cmd/misspell
EXT_TOOLS_DIR = ext-tools/$(OS)
DEP_TOOL = $(EXT_TOOLS_DIR)/dep

BUILD_LDFLAGS = -X $(PACKAGE_NAME)/lib/utils.BuildHash=$(PACKAGE_VERSION)
GO_FLAGS = -gcflags '-N -l' -ldflags "$(BUILD_LDFLAGS)"
GO_VERSION = 1.11

REGISTRY ?= gcr.io/makisu-project


### Targets to compile the makisu binaries.
.PHONY: bins lbins cbins
bins: bin/makisu/makisu

bin/makisu/makisu: $(ALL_SRC) vendor
	go build -tags bins $(GO_FLAGS) -o $@ bin/makisu/*.go

lbins: bin/makisu/makisu.linux

bin/makisu/makisu.linux: $(ALL_SRC) vendor
	CGO_ENABLED=0 GOOS=linux go build -tags bins $(GO_FLAGS) -o $@ bin/makisu/*.go

cbins:
	docker run -i --rm -v $(PWD):/go/src/$(PACKAGE_NAME) \
		--net=host \
		--entrypoint=bash \
		-w /go/src/$(PACKAGE_NAME) \
		golang:$(GO_VERSION) \
		-c "make lbins"

$(ALL_SRC): ;


### Targets to install the dependencies.
$(DEP_TOOL):
	mkdir -p $(EXT_TOOLS_DIR)
	go get github.com/golang/dep/cmd/dep
	cp $(GOPATH)/bin/dep $(EXT_TOOLS_DIR)

# TODO(pourchet): Remove this hack to make dep more reliable. For some reason `dep ensure` fails
# sometimes on TravisCI, so run it twice if it fails the first time.
vendor: $(DEP_TOOL) Gopkg.toml
	$(EXT_TOOLS_DIR)/dep ensure || $(EXT_TOOLS_DIR)/dep ensure

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

env: test/python/requirements.txt
	[ -d env ] || virtualenv --setuptools env
	./env/bin/pip install -q -r test/python/requirements.txt



### Target to build the makisu docker images.
.PHONY: images publish
images:
	docker build -t $(REGISTRY)/makisu:$(PACKAGE_VERSION) -f Dockerfile .
	docker tag $(REGISTRY)/makisu:$(PACKAGE_VERSION) makisu:$(PACKAGE_VERSION)
	docker build -t $(REGISTRY)/makisu-alpine:$(PACKAGE_VERSION) -f Dockerfile.alpine .
	docker tag $(REGISTRY)/makisu-alpine:$(PACKAGE_VERSION) makisu-alpine:$(PACKAGE_VERSION)

publish: images
	docker push $(REGISTRY)/makisu:$(PACKAGE_VERSION)
	docker push $(REGISTRY)/makisu-alpine:$(PACKAGE_VERSION)



### Targets to test the codebase.
.PHONY: test unit-test integration cunit-test
test: unit-test integration

unit-test: $(ALL_SRC) vendor ext-tools mocks
	$(EXT_TOOLS_DIR)/gocov test $(ALL_PKGS) --tags "unit" | $(EXT_TOOLS_DIR)/gocov report

cunit-test: $(ALL_SRC)
	docker run -i --rm -v $(PWD):/go/src/$(PACKAGE_NAME) \
		--net=host \
		--entrypoint=bash \
		-w /go/src/$(PACKAGE_NAME) \
		golang:$(GO_VERSION) \
		-c "make ext-tools unit-test"

integration: env images
	PACKAGE_VERSION=$(PACKAGE_VERSION) ./env/bin/py.test --maxfail=1 --durations=6 --timeout=300 -vv test/python



### Misc targets
.PHONY: clean integration-single
integration-single: env images
	PACKAGE_VERSION=$(PACKAGE_VERSION) ./env/bin/py.test test/python/test_build.py::$(TEST_NAME)


# TODO(pourchet) fix gometalinter installation from source
lint: ext-tools
	@echo "Running ineffassign, gofmt, misspell, gometalinter, gocyclo"
	@ineffassign <<< $(ALL_PKG_PATHS)
	@gofmt -l -s $(ALL_PKG_PATHS) | read; if [ $$? == 0 ]; then echo "gofmt check failed for:"; gofmt -l -s $(ALL_PKG_PATHS); exit 1; fi
	@$(EXT_TOOLS_DIR)/misspell -w --error -i hardlinked $(ALL_PKG_PATHS)
	gometalinter --vendor --disable vet -e 'warning' --fast ./...
	@xargs -I@ gocyclo --over 15 @ <<< $(ALL_PKG_PATHS)


clean:
	git clean -fd
	-rm -rf vendor ext-tools mocks env
	-rm bin/makisu/makisu
