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

$(GOPATH)/bin/gobike-rewind: $(GO_FILES)
	go install ./cmd/gobike-rewind

$(GOPATH)/bin/gobike-geo: cmd/gobike-geo/main.go
	go install ./cmd/gobike-geo

sf-districts:
	$(GOPATH)/bin/gobike-geo geojson/sf-districts/1.geojson geo/sfdistrict1.go SFD1
	$(GOPATH)/bin/gobike-geo geojson/sf-districts/2.geojson geo/sfdistrict2.go SFD2
	$(GOPATH)/bin/gobike-geo geojson/sf-districts/3.geojson geo/sfdistrict3.go SFD3
	$(GOPATH)/bin/gobike-geo geojson/sf-districts/4.geojson geo/sfdistrict4.go SFD4
	$(GOPATH)/bin/gobike-geo geojson/sf-districts/5.geojson geo/sfdistrict5.go SFD5
	$(GOPATH)/bin/gobike-geo geojson/sf-districts/6.geojson geo/sfdistrict6.go SFD6
	$(GOPATH)/bin/gobike-geo geojson/sf-districts/7.geojson geo/sfdistrict7.go SFD7
	$(GOPATH)/bin/gobike-geo geojson/sf-districts/8.geojson geo/sfdistrict8.go SFD8
	$(GOPATH)/bin/gobike-geo geojson/sf-districts/9.geojson geo/sfdistrict9.go SFD9
	$(GOPATH)/bin/gobike-geo geojson/sf-districts/10.geojson geo/sfdistrict10.go SFD10
	$(GOPATH)/bin/gobike-geo geojson/sf-districts/11.geojson geo/sfdistrict11.go SFD11

# Geojson mappings
geo/berkeley.go: geojson/berkeley.geojson $(GOPATH)/bin/gobike-geo
	$(GOPATH)/bin/gobike-geo geojson/berkeley.geojson geo/berkeley.go Berkeley

geo/sanfrancisco.go: geojson/sanfrancisco.geojson $(GOPATH)/bin/gobike-geo
	$(GOPATH)/bin/gobike-geo geojson/sanfrancisco.geojson geo/sanfrancisco.go SF

geo/oakland.go: geojson/oakland.geojson $(GOPATH)/bin/gobike-geo
	$(GOPATH)/bin/gobike-geo geojson/oakland.geojson geo/oakland.go Oakland

geo/emeryville.go: geojson/emeryville.geojson $(GOPATH)/bin/gobike-geo
	$(GOPATH)/bin/gobike-geo geojson/emeryville.geojson geo/emeryville.go Emeryville

geo/sanjose.go: geojson/sanjose.geojson $(GOPATH)/bin/gobike-geo
	$(GOPATH)/bin/gobike-geo geojson/sanjose.geojson geo/sanjose.go SanJose

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
