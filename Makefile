.PHONY: build test lint clean run

BINARY=routerforge
GO=nix-shell -p go_1_24 --command
GOSUM=GONOSUMCHECK=* GONOSUMDB=*

all: build

build:
	$(GO) "$(GOSUM) go build -o $(BINARY) ."

test:
	$(GO) "$(GOSUM) go test ./... -v"

lint:
	$(GO) "$(GOSUM) go vet ./..."

fmt:
	$(GO) "$(GOSUM) go fmt ./..."

tidy:
	$(GO) "$(GOSUM) go mod tidy"

clean:
	rm -f $(BINARY)
	rm -rf .routerforge/

run: build
	./$(BINARY)
