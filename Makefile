GO ?= go

SDK_MODULE_ENV := GOPRIVATE=github.com/akmalabbasov/evroc-sdk GONOSUMDB=github.com/akmalabbasov/evroc-sdk

.PHONY: test schema example generate

test:
	$(SDK_MODULE_ENV) GOWORK=off $(GO) test ./...

schema:
	$(SDK_MODULE_ENV) $(GO) run ./cmd/evroc-machine-driver schema > schema.json

example:
	$(SDK_MODULE_ENV) $(GO) run ./cmd/evroc-machine-driver example-config > machine-config.example.yaml

generate: schema example
