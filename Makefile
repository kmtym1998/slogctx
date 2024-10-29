build:
	go build ./cmd/slogctx/main.go

run: build
	go vet -vettool=$(shell pwd)/main -slogctx.target=debug/model.go,debug/not_slog.go,debug/slog.go ./...
