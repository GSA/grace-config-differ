GOOS = linux
GOARCH = amd64
GOBIN := $(GOPATH)/bin
GOLANGCILINT := $(GOBIN)/golangci-lint
GOSEC := $(GOBIN)/gosec
GHR := $(GOBIN)/ghr
RELEASEDIR ?= ../release/
CIRCLE_PROJECT_REPONAME ?= grace-config-differ
export GO111MODULE=on

.PHONY: build release clean test lint dependencies
build: test clean
	mkdir -p $(RELEASEDIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(RELEASEDIR)$(CIRCLE_PROJECT_REPONAME) -v
	zip -j $(RELEASEDIR)$(CIRCLE_PROJECT_REPONAME).zip $(RELEASEDIR)$(CIRCLE_PROJECT_REPONAME)
	rm -f $(RELEASEDIR)$(CIRCLE_PROJECT_REPONAME)

release: build
	export CIRCLE_TAG=$(CIRCLE_TAG)
ifeq ($(strip $(GITHUB_TOKEN)),)
	@echo "GITHUB_TOKEN must be provided"
	@exit 0
else ifeq ($(strip $(CIRCLE_TAG)),)
	@echo "CIRCLE_TAG must be set"
	@exit 0
else ifeq ($(shell echo $(CIRCLE_TAG) | egrep '^v\d+\.\d+\.\d+'),)
	@echo "CIRCLE_TAG must match version pattern (i.e. v.1.2.3)"
	@exit 0
else
	ghr -u $(CIRCLE_PROJECT_USERNAME) -r $(CIRCLE_PROJECT_REPONAME) -c $(CIRCLE_SHA1) -n $(CIRCLE_TAG) -delete $(CIRCLE_TAG) $(RELEASEDIR)
endif

clean:
	rm -rf $(RELEASEDIR)

test: lint
	go test -v ./...

lint: go.mod dependencies
	./bin/golangci-lint run ./...
	./bin/gosec ./...

go.mod:
ifeq (,$(wildcard go.mod))
	go mod init
endif

dependencies: $(GOLANGCILINT) $(GOSEC) $(GHR)

$(GOLANGCILINT):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.23.6
	# linter fails on latest version, pinning to last passing version. latest version is:
		# curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.41.1

$(GOSEC):
	curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s v2.8.1

$(GHR):
	go get -u github.com/tcnksm/ghr
