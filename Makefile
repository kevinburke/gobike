SHELL = /bin/bash -o pipefail

GO_BINDATA := $(GOPATH)/bin/go-bindata
WATCH_TARGETS = $(shell find ./server/static ./server/templates -type f)
GO_FILES = $(shell find . -name '*.go')
GENERATE_TLS_CERT = $(GOPATH)/bin/generate-tls-cert
BENCHSTAT := $(GOPATH)/bin/benchstat
BUMP_VERSION := $(GOPATH)/bin/bump_version
MEGACHECK := $(GOPATH)/bin/megacheck
JUSTRUN := $(GOPATH)/bin/justrun
RELEASE := $(GOPATH)/bin/github-release
UNAME = $(shell uname -s)
GO_NOASSET_FILES := $(filter-out ./server/assets/bindata.go,$(GO_FILES))

test:
	go test ./...

race-test: lint
	go test -race ./...

lint: | $(MEGACHECK)
	go vet -composites=false ./...
	go list ./... | grep -v vendor | xargs $(MEGACHECK)

bench: | $(BENCHSTAT)
	go list ./... | grep -v vendor | xargs go test -benchtime=2s -bench=. -run='^$$' 2>&1 | $(BENCHSTAT) /dev/stdin

server/assets/bindata.go: $(WATCH_TARGETS) | $(GO_BINDATA)
	cd server && $(GO_BINDATA) -o=assets/bindata.go --nocompress --nometadata --pkg=assets templates/... static/...

assets: server/assets/bindata.go

serve: $(GOPATH)/bin/gobike-server
	$(GOPATH)/bin/gobike-server

$(GOPATH)/bin/gobike-server: $(GO_FILES)
	go install ./cmd/gobike-server

$(GENERATE_TLS_CERT):
	go get -u github.com/kevinburke/generate-tls-cert

certs/leaf.pem: | $(GENERATE_TLS_CERT)
	mkdir -p certs
	cd certs && $(GENERATE_TLS_CERT) --host=localhost,127.0.0.1

# Generate TLS certificates for local development.
generate_cert: certs/leaf.pem | $(GENERATE_TLS_CERT)

watch: | $(JUSTRUN)
	$(JUSTRUN) -v --delay=100ms -c 'make assets serve' $(WATCH_TARGETS) $(GO_NOASSET_FILES)

$(GO_BINDATA):
	go get -u github.com/kevinburke/go-bindata/...

$(JUSTRUN):
	go get -u github.com/jmhodges/justrun

$(BUMP_VERSION):
	go get -u github.com/kevinburke/bump_version

$(BENCHSTAT):
	go get golang.org/x/perf/cmd/benchstat

$(RELEASE):
	go get -u github.com/aktau/github-release

$(GOPATH)/bin:
	mkdir -p $(GOPATH)/bin

$(MEGACHECK): | $(GOPATH)/bin
ifeq ($(UNAME),Darwin)
	curl --silent --location --output $(MEGACHECK) https://github.com/kevinburke/go-tools/releases/download/2018-04-15/megacheck-darwin-amd64
else
	curl --silent --location --output $(MEGACHECK) https://github.com/kevinburke/go-tools/releases/download/2018-04-15/megacheck-linux-amd64
endif
	chmod +x $(MEGACHECK)
