.PHONY: build test lint clean run

BINARY=routerforge
GO=nix-shell -p go_1_24 --run "go"
GOBIN=/nix/store/9s1r393dnb5mygiq5f9yxy76nxpkz1gw-go-1.24.4/bin

all: build

build:
	GOBIN=$(GOBIN) $(GOBIN)/go build -o $(BINARY) .

test:
	GOBIN=$(GOBIN) $(GOBIN)/go test ./... -v

lint:
	GOBIN=$(GOBIN) $(GOBIN)/go vet ./...

fmt:
	GOBIN=$(GOBIN) $(GOBIN)/go fmt ./...

tidy:
	GOBIN=$(GOBIN) $(GOBIN)/go mod tidy

clean:
	rm -f $(BINARY)
	rm -rf .routerforge/

run: build
	./$(BINARY)
