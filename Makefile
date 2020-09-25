build:
	go build ./cmd/mock-apollo-go

.PHONY: test
test:
	go test -race ./...
