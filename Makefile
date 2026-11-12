.PHONY: clean

.DEFAULT_GOAL := build

# default build target
build:
	@go build -o ipdb_transformer

clean:
	@rm -f ipdb_transformer && go clean
