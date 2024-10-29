build:
	go build ./cmd/slogctx/main.go

run: build
	go vet -vettool=$(shell pwd)/main -slogctx.target=testdata/src/a/a.go ./...
