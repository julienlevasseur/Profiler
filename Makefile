.PHONY: all

GOARCH=amd64
GO_BUILD=go build

BINARY_PATH=bin/profiler
DIST_PATH=dist

clean:
	$(RM) $(BINARY_PATH)
	$(RM) -r $(DIST_PATH)

build: linux darwin

linux:
	GO112MODULE=on CGO_ENABLED=1 GOOS=linux GOARCH=$(GOARCH) $(GO_BUILD) -o $(BINARY_PATH)

darwin:
	GO112MODULE=on CGO_ENABLED=1 GOOS=darwin GOARCH=$(GOARCH) $(GO_BUILD) -o $(BINARY_PATH)

release-snapshot:
	goreleaser --snapshot

release:
	goreleaser
