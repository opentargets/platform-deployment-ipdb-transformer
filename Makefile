.PHONY: clean

# default build target
build:
	@go build -o ipdb_transformer

clean:
	@rm -f ipdb_transformer && go clean
