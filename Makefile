BIN = bin/kube-plex
PACKAGES = $$(go list ./... | grep -v '/vendor/')
FILES = $(shell find . -type f -name '*.go' -print)

export GO111MODULE=on
export CGO_ENABLED=0

default: clean build

build: $(BIN)

$(BIN):
	go build -i -v \
                -tags release \
                -ldflags="-X main.version=1.1" \
                -o $(BIN) \
                cmd/kube-plex/main.go

clean:
	rm -rfv bin

fmt:
	gofmt -l -s -w $(FILES)

test:
	TEST=1 go test $(PACKAGES)

docker:
	docker build -t docker.nikore.net/mattbot:latest .
