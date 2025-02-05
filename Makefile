GOPATH := $(go env GOPATH)
SRC_FILES := $(shell find . -name '*.go')
GO_MOD := $(go list -m)
ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
PATH := $(ROOT_DIR)/bin:$(ROOT_DIR)/.local/bin:$(GOPATH)/bin:$(PATH)
CC = env PATH=$(PATH) go build -ldflags '$(LD_FLAGS)'

# Formatting/Display
Q:=$(if $(filter 1,$(VERBOSE)),,@)
M = $(shell printf "\033[34;1m▶\033[0m")
T = $(shell printf " ")

.PHONY: help
help: ## show this
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-25s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: | fmt vet test bin/heroku-integration-service-mesh
	$(info $(M) done)

.PHONY: lint
lint: ## run go linters
	$(Q) staticcheck ./...
	$(Q) golangci-lint run


.PHONY: test
test: ## run all of the test cases
	$(info $(M) testing …)
	$(Q) go test ./test/**

.PHONY: test-coverage
test-coverage: ## run all of the test cases with code coverage
	$(info $(M) test-coverage …)
	$(Q) go test -v ./test/** -coverpkg=./... -coverprofile ./coverage.out
	go tool cover -func ./coverage.out

.PHONY: test-fmt
test-fmt: ## run all of the test cases with formatting (requires https://github.com/GoTestTools/gotestfmt)
	$(info $(M) test-fmt …)
	$(Q) go test -json -v ./test/** 2>&1 | tee /tmp/heroku-integration-service-mesh-gotestfmt.log | gotestfmt

.PHONY: fmt
fmt: ## run go fmt on all source files
	$(info $(M) formatting …)
	$(Q) env PATH=$(PATH) go fmt ./...

.PHONY: vet
vet: ## run go vet on all source files
	$(info $(M) vetting …)
	$(Q) env PATH=$(PATH) go vet ./...

.PHONY: generate
generate: ## run go generate
	$(info $(M) generating ...)
	$(Q) env PATH=$(PATH) go generate ./...

bin/%: $(SRC_FILES)
	$(info $(M) building $@ …)
	$(Q) $(CC) -o $@ github.com/heroku/heroku-integration-service-mesh

tar-gz: | build ## build and tar the binary
	$(info $(M) tar heroku-integration-service-mesh ...)
	$(Q) tar -zcvf heroku-integration-service-mesh.tar.gz -C bin heroku-integration-service-mesh