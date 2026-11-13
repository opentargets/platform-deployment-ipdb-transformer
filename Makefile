.PHONY: clean test

.DEFAULT_GOAL := build

# default build target
build:
	@go build -o ipdb_transformer

test:
	@go test ./...

clean:
	@rm -f ipdb_transformer && go clean
