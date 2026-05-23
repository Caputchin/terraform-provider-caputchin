BINARY = terraform-provider-caputchin
VERSION ?= 0.1.0-dev

.PHONY: build test testacc lint fmt vet docs release-snapshot clean

build:
	go build -o $(BINARY) .

test:
	go test ./... -timeout=120s

testacc:
	TF_ACC=1 go test ./... -timeout=300s -v

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .
	goimports -w -local github.com/caputchin/terraform-provider-caputchin .

vet:
	go vet ./...

docs:
	tfplugindocs generate --provider-name caputchin

release-snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -f $(BINARY)
	rm -rf dist/
