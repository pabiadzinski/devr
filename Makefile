APP := devr
PKG := ./cmd/$(APP)
VERSION ?= dev
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: run build install snapshot

run:
	go run $(PKG) $(ARGS)

build:
	go build $(LDFLAGS) -o /tmp/$(APP) $(PKG)

install:
	go install $(LDFLAGS) $(PKG)

snapshot:
	goreleaser release --snapshot --clean
