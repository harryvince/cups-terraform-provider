.PHONY: build fmt test vet test-acc testenv-up testenv-down

build:
	go build -o bin/terraform-provider-cups .

fmt:
	gofmt -w main.go internal
	terraform fmt -recursive examples

test:
	go test ./...

vet:
	go vet ./...

# Endpoint/credentials must be supplied explicitly; see docs/testing.md.
test-acc:
	CUPS_ACC=1 go test -v -count=1 -timeout=10m ./internal/provider -run TestAcceptance

testenv-up:
	docker compose up --build --wait --wait-timeout 120

testenv-down:
	docker compose down
