SHELL = /bin/bash -o pipefail

WATCH_TARGETS = $(shell find ./server/static ./server/templates -type f)
GO_FILES = $(shell find . -name '*.go')
BENCHSTAT := $(GOPATH)/bin/benchstat
BUMP_VERSION := $(GOPATH)/bin/bump_version
MEGACHECK := $(GOPATH)/bin/megacheck
JUSTRUN := $(GOPATH)/bin/justrun
RELEASE := $(GOPATH)/bin/github-release
UNAME = $(shell uname -s)

test:
	go test ./...

race-test: lint
	go test -race ./...

lint: | $(MEGACHECK)
	go vet -composites=false ./...
	go list ./... | grep -v vendor | xargs $(MEGACHECK) --ignore='github.com/kevinburke/gobike/geo.go:U1000'

bench: | $(BENCHSTAT)
	go list ./... | grep -v vendor | xargs go test -benchtime=2s -bench=. -run='^$$' 2>&1 | $(BENCHSTAT) /dev/stdin

serve: $(GOPATH)/bin/gobike-server
	$(GOPATH)/bin/gobike-server

dataset: $(GOPATH)/bin/gobike-dataset
	$(GOPATH)/bin/gobike-dataset data/ data/trips.csv

site: $(GOPATH)/bin/gobike-site
	$(GOPATH)/bin/gobike-site data/

polygons: geo/berkeley.go geo/sanfrancisco.go geo/oakland.go geo/emeryville.go geo/sanjose.go

$(GOPATH)/bin/gobike-server: $(GO_FILES)
	go install ./cmd/gobike-server

$(GOPATH)/bin/gobike-site: $(GO_FILES)
	go install ./cmd/gobike-site

$(GOPATH)/bin/gobike-dataset: $(GO_FILES)
	go install ./cmd/gobike-dataset

$(GOPATH)/bin/gobike-geo: cmd/gobike-geo/main.go
	go install ./cmd/gobike-geo

# Geojson mappings
geo/berkeley.go: geojson/2833528.geojson $(GOPATH)/bin/gobike-geo
	$(GOPATH)/bin/gobike-geo geojson/2833528.geojson geo/berkeley.go Berkeley

geo/sanfrancisco.go: geojson/111968.geojson $(GOPATH)/bin/gobike-geo
	$(GOPATH)/bin/gobike-geo geojson/111968.geojson geo/sanfrancisco.go SF

geo/oakland.go: geojson/2833530.geojson $(GOPATH)/bin/gobike-geo
	$(GOPATH)/bin/gobike-geo geojson/2833530.geojson geo/oakland.go Oakland

geo/emeryville.go: geojson/2833529.geojson $(GOPATH)/bin/gobike-geo
	$(GOPATH)/bin/gobike-geo geojson/2833529.geojson geo/emeryville.go Emeryville

geo/sanjose.go: geojson/112143.geojson $(GOPATH)/bin/gobike-geo
	$(GOPATH)/bin/gobike-geo geojson/112143.geojson geo/sanjose.go SanJose

watch: | $(JUSTRUN)
	$(JUSTRUN) -v --delay=100ms -c 'make stats serve' $(WATCH_TARGETS)

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
