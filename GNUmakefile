default: build

build:
	go build ./...

# Unit tests (mock the op CLI; no live 1Password account).
test:
	go test ./... -count=1

# Acceptance tests: run locally against a dedicated 1Password Business test
# org (never CI, never a production org). Requires `op` authenticated and
# TF_ACC=1.
testacc:
	TF_ACC=1 go test ./... -v -count=1 -timeout 30m

install:
	go install .

fmt:
	gofmt -s -w .

vet:
	go vet ./...

.PHONY: default build test testacc install fmt vet
